// Copyright 2019 Red Hat
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package platform

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	origExec "os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"

	"github.com/flatcar/mantle/system/exec"
	"github.com/flatcar/mantle/util"
)

type MachineOptions struct {
	AdditionalDisks      []Disk
	ExtraPrimaryDiskSize string
	EnableTPM            bool
	SoftwareTPMSocket    string
	VNC                  string
}

type Disk struct {
	Size          string   // disk image size in bytes, optional suffixes "K", "M", "G", "T" allowed. Incompatible with BackingFile
	BackingFile   string   // raw disk image to use. Incompatible with Size.
	ExtraDiskSize string   // additional disk size to add to the image in bytes, optional suffixes "K", "M", "G", "T" allowed. Incompatible with Size.
	DeviceOpts    []string // extra options to pass to qemu. "serial=XXXX" makes disks show up as /dev/disk/by-id/virtio-<serial>
}

var (
	ErrNeedSizeOrFile    = errors.New("Disks need either Size or BackingFile specified")
	ErrBothSizeAndFile   = errors.New("Only one of Size and BackingFile can be specified")
	ErrExtraWithFileOnly = errors.New("ExtraDiskSize can only be used with BackingFile")
	primaryDiskOptions   = []string{"serial=primary-disk"}
)

// Copy Container Linux input image and specialize copy for running kola tests.
// Return FD to the copy, which is a deleted file.
// This is not mandatory; the tests will do their best without it.
func MakeCLDiskTemplate(inputPath string) (output *os.File, result error) {
	seterr := func(err error) {
		if result == nil {
			result = err
		}
	}

	// create output file
	outputPath, err := mkpath("/var/tmp")
	if err != nil {
		return nil, err
	}
	defer os.Remove(outputPath)

	// copy file
	// cp is used since it supports sparse and reflink.
	cp := exec.Command("cp", "--force",
		"--sparse=always", "--reflink=auto",
		inputPath, outputPath)
	cp.Stdout = os.Stdout
	cp.Stderr = os.Stderr
	if err := cp.Run(); err != nil {
		return nil, fmt.Errorf("copying file: %v", err)
	}

	// create mount point
	tmpdir, err := ioutil.TempDir("", "kola-qemu-")
	if err != nil {
		return nil, fmt.Errorf("making temporary directory: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpdir); err != nil {
			seterr(fmt.Errorf("deleting directory %s: %v", tmpdir, err))
		}
	}()

	// set up loop device
	cmd := exec.Command("losetup", "-Pf", "--show", outputPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("getting stdout pipe: %v", err)
	}
	defer stdout.Close()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("running losetup: %v", err)
	}
	buf, err := ioutil.ReadAll(stdout)
	if err != nil {
		cmd.Wait()
		return nil, fmt.Errorf("reading losetup output: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("setting up loop device: %v", err)
	}
	loopdev := strings.TrimSpace(string(buf))
	defer func() {
		if err := exec.Command("losetup", "-d", loopdev).Run(); err != nil {
			seterr(fmt.Errorf("tearing down loop device: %v", err))
		}
	}()

	// wait for OEM block device
	oemdev := loopdev + "p6"
	err = util.RetryConditional(1000, 5*time.Millisecond, os.IsNotExist,
		func() error {
			_, err := os.Stat(oemdev)
			return err
		})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("timed out waiting for device node %s; did you specify a qcow image by mistake?", oemdev)
		}
		return nil, fmt.Errorf("failed to get loop device %s: %v", oemdev, err)
	}

	// mount OEM partition, wait for exclusive access to the file system in case some other process also mounted an identical OEM btrfs filesystem
	err = util.RetryConditional(600, 1000*time.Millisecond, func(err error) bool {
		if exitCode, ok := err.(*origExec.ExitError); ok && exitCode.ProcessState.ExitCode() == 32 {
			plog.Noticef("waiting for exclusive access to the OEM btrfs filesystem")
			return true
		}
		return false
	}, func() error {
		return exec.Command("mount", oemdev, tmpdir).Run()
	})
	if err != nil {
		if exitCode, ok := err.(*origExec.ExitError); ok && exitCode.ProcessState.ExitCode() == 32 {
			return nil, fmt.Errorf("timed out waiting to mount the OEM btrfs filesystem exclusively from %s on %s: %v", oemdev, tmpdir, err)
		}
		return nil, fmt.Errorf("mounting OEM partition %s on %s: %v", oemdev, tmpdir, err)
	}
	defer func() {
		if err := exec.Command("umount", tmpdir).Run(); err != nil {
			seterr(fmt.Errorf("unmounting %s: %v", tmpdir, err))
		}
	}()

	// write console settings to grub.cfg
	f, err := os.OpenFile(filepath.Join(tmpdir, "grub.cfg"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening grub.cfg: %v", err)
	}
	defer f.Close()
	if _, err = f.WriteString("set linux_console=\"console=ttyS0,115200\"\n"); err != nil {
		return nil, fmt.Errorf("writing grub.cfg: %v", err)
	}

	// return fd to output file
	output, err = os.Open(outputPath)
	if err != nil {
		return nil, fmt.Errorf("opening %v: %v", outputPath, err)
	}
	return
}

func (d Disk) getOpts() string {
	if len(d.DeviceOpts) == 0 {
		return ""
	}
	return "," + strings.Join(d.DeviceOpts, ",")
}

func (d Disk) setupFile() (*os.File, error) {
	if d.Size == "" && d.BackingFile == "" {
		return nil, ErrNeedSizeOrFile
	}
	if d.Size != "" && d.BackingFile != "" {
		return nil, ErrBothSizeAndFile
	}
	if d.Size != "" && d.ExtraDiskSize != "" {
		return nil, ErrExtraWithFileOnly
	}

	if d.Size != "" {
		return setupDisk(d.Size)
	} else {
		return setupDiskFromFile(d.BackingFile, d.ExtraDiskSize)
	}
}

// Create a nameless temporary qcow2 image file backed by a raw image.
func setupDiskFromFile(imageFile, extraDiskSize string) (*os.File, error) {
	// a relative path would be interpreted relative to /tmp
	backingFile, err := filepath.Abs(imageFile)
	if err != nil {
		return nil, err
	}
	// Keep the COW image from breaking if the "latest" symlink changes.
	// Ignore /proc/*/fd/* paths, since they look like symlinks but
	// really aren't.
	if !strings.HasPrefix(backingFile, "/proc/") {
		backingFile, err = filepath.EvalSymlinks(backingFile)
		if err != nil {
			return nil, err
		}
	}
	imgInfo, err := util.GetImageInfo(backingFile)
	if err != nil {
		return nil, err
	}
	sizeOpt := ""
	if extraDiskSize != "" {
		diskSize, err := parseDiskSize(extraDiskSize)
		if err != nil {
			return nil, fmt.Errorf("failed to parse extra disk size %s: %v", extraDiskSize, err)
		}
		diskSize += imgInfo.VirtualSize
		sizeOpt = fmt.Sprintf(",size=%d", diskSize)
	}

	qcowOpts := fmt.Sprintf("backing_file=%s,backing_fmt=%s,lazy_refcounts=on%s", backingFile, imgInfo.Format, sizeOpt)
	return setupDisk("-o", qcowOpts)
}

func parseDiskSize(diskSize string) (uint64, error) {
	multiplier := (uint64)(1)
	last := len(diskSize) - 1
	suffix := diskSize[last]
	digitsOnly := diskSize
	switch suffix {
	case 'T':
		multiplier *= 1024
		fallthrough
	case 'G':
		multiplier *= 1024
		fallthrough
	case 'M':
		multiplier *= 1024
		fallthrough
	case 'K', 'k':
		multiplier *= 1024
		fallthrough
	case 'b':
		digitsOnly = diskSize[0:last]
	default:
		if suffix < '0' || suffix > '9' {
			return 0, fmt.Errorf("invalid suffix %c in %s for disk size", suffix, diskSize)
		}
	}
	result, err := strconv.ParseUint(digitsOnly, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid disk size %s (parsed %s): %v", diskSize, digitsOnly, err)
	}
	return result * multiplier, nil
}

func setupDisk(additionalOptions ...string) (*os.File, error) {
	dstFileName, err := mkpath("")
	if err != nil {
		return nil, err
	}
	defer os.Remove(dstFileName)

	opts := []string{"create", "-f", "qcow2", dstFileName}
	opts = append(opts, additionalOptions...)

	qemuImg := exec.Command("qemu-img", opts...)
	qemuImg.Stderr = os.Stderr

	if err := qemuImg.Run(); err != nil {
		return nil, err
	}

	return os.OpenFile(dstFileName, os.O_RDWR, 0)
}

func mkpath(basedir string) (string, error) {
	f, err := ioutil.TempFile(basedir, "mantle-qemu")
	if err != nil {
		return "", err
	}
	defer f.Close()
	return f.Name(), nil
}

func CreateQEMUCommand(board, uuid, firmware, ovmfVars, consolePath, confPath, diskImagePath string, enableSecureboot, isIgnition bool, options MachineOptions) ([]string, []*os.File, error) {
	var qmCmd []string

	// As we expand this list of supported native + board
	// archs combos we should coordinate with the
	// coreos-assembler folks as they utilize something
	// similar in cosa run
	var qmBinary string
	combo := runtime.GOARCH + "--" + board
	switch combo {
	case "amd64--amd64-usr":
		qmBinary = "qemu-system-x86_64"
		qmCmd = []string{
			"qemu-system-x86_64",
			"-machine", "q35,accel=kvm,smm=on",
			"-cpu", "host",
			"-m", "2512",
		}
	case "amd64--arm64-usr":
		qmBinary = "qemu-system-aarch64"
		qmCmd = []string{
			"qemu-system-aarch64",
			"-machine", "virt",
			"-cpu", "cortex-a57",
			"-m", "2512",
		}
	case "arm64--amd64-usr":
		qmBinary = "qemu-system-x86_64"
		qmCmd = []string{
			"qemu-system-x86_64",
			"-machine", "pc-q35-2.8",
			"-cpu", "kvm64",
			"-m", "2512",
		}
	case "arm64--arm64-usr":
		qmBinary = "qemu-system-aarch64"
		qmCmd = []string{
			"qemu-system-aarch64",
			"-machine", "virt,accel=kvm,gic-version=3",
			"-cpu", "host",
			"-m", "2512",
		}
	default:
		panic("host-guest combo not supported: " + combo)
	}

	qmCmd = append(qmCmd,
		"-smp", "4",
		"-uuid", uuid,
		"-display", "none",
		"-chardev", "file,id=log,path="+consolePath,
		"-serial", "chardev:log",
		"-object", "rng-random,filename=/dev/urandom,id=rng0",
		"-device", "virtio-rng-pci,rng=rng0",
		"-drive", fmt.Sprintf("if=pflash,unit=0,file=%v,format=raw,readonly=on", firmware),
	)

	if enableSecureboot == true {
		// Create a copy of the OVMF Vars
		ovmfVarsSrc, err := os.Open(ovmfVars)
		if err != nil {
			return nil, nil, err
		}
		defer ovmfVarsSrc.Close()

		ovmfVarsCopy, err := ioutil.TempFile("/var/tmp/", "mantle-qemu")
		if err != nil {
			return nil, nil, err
		}

		if _, err := io.Copy(ovmfVarsCopy, ovmfVarsSrc); err != nil {
			return nil, nil, err
		}

		_, err = ovmfVarsCopy.Seek(0, 0)
		if err != nil {
			return nil, nil, err
		}

		qmCmd = append(qmCmd,
			"-global", "ICH9-LPC.disable_s3=1",
			"-global", "driver=cfi.pflash01,property=secure,value=on",
			"-drive", fmt.Sprintf("if=pflash,unit=1,file=%v,format=raw", ovmfVarsCopy.Name()),
		)
	}

	if options.EnableTPM {
		var tpm string
		switch board {
		case "amd64-usr":
			tpm = "tpm-tis"
		case "arm64-usr":
			tpm = "tpm-tis-device"
		default:
			panic(board)
		}
		qmCmd = append(qmCmd,
			"-chardev", fmt.Sprintf("socket,id=chrtpm,path=%v", options.SoftwareTPMSocket),
			"-tpmdev", "emulator,id=tpm0,chardev=chrtpm",
			"-device", fmt.Sprintf("%s,tpmdev=tpm0", tpm),
		)
	}

	if isIgnition {
		qmCmd = append(qmCmd,
			"-fw_cfg", "name=opt/org.flatcar-linux/config,file="+confPath)
	} else {
		qmCmd = append(qmCmd,
			"-fsdev", "local,id=cfg,security_model=none,readonly=on,path="+confPath,
			"-device", Virtio(board, "9p", "fsdev=cfg,mount_tag=config-2"))
	}

	if options.VNC != "" {
		qmCmd = append(qmCmd, "-vnc", fmt.Sprintf(":%s", options.VNC))
	}

	// auto-read-only is only available in 3.1.0 & greater versions of QEMU
	var autoReadOnly string
	version, err := exec.Command(qmBinary, "--version").CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("retrieving qemu version: %v", err)
	}
	pat := regexp.MustCompile(`version (\d+\.\d+\.\d+)`)
	vNum := pat.FindSubmatch(version)
	if len(vNum) < 2 {
		return nil, nil, fmt.Errorf("unable to parse qemu version number")
	}
	qmSemver, err := semver.NewVersion(string(vNum[1]))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing qemu semver: %v", err)
	}
	if !qmSemver.LessThan(*semver.New("3.1.0")) {
		autoReadOnly = ",auto-read-only=off"
		plog.Debugf("disabling auto-read-only for QEMU drives")
	}

	allDisks := append([]Disk{
		{
			BackingFile:   diskImagePath,
			DeviceOpts:    primaryDiskOptions,
			ExtraDiskSize: options.ExtraPrimaryDiskSize,
		},
	}, options.AdditionalDisks...)

	var extraFiles []*os.File
	fdnum := 3 // first additional file starts at position 3
	fdset := 1

	for _, disk := range allDisks {
		bootIndexArg := ""
		if slices.Contains(disk.DeviceOpts, "serial=primary-disk") {
			bootIndexArg = ",bootindex=1"
		}

		optionsDiskFile, err := disk.setupFile()
		if err != nil {
			return nil, nil, err
		}
		//defer optionsDiskFile.Close()
		extraFiles = append(extraFiles, optionsDiskFile)

		id := fmt.Sprintf("d%d", fdnum)
		qmCmd = append(qmCmd, "-add-fd", fmt.Sprintf("fd=%d,set=%d", fdnum, fdset),
			"-drive", fmt.Sprintf("if=none,id=%s,format=qcow2,file=/dev/fdset/%d%s", id, fdset, autoReadOnly),
			"-device", Virtio(board, "blk", fmt.Sprintf("drive=%s%s%s", id, disk.getOpts(), bootIndexArg)))
		fdnum += 1
		fdset += 1
	}

	return qmCmd, extraFiles, nil
}

// The virtio device name differs between machine types but otherwise
// configuration is the same. Use this to help construct device args.
func Virtio(board, device, args string) string {
	var suffix string
	switch board {
	case "amd64-usr":
		suffix = "pci"
	case "arm64-usr":
		suffix = "device"
	default:
		panic(board)
	}
	return fmt.Sprintf("virtio-%s-%s,%s", device, suffix, args)
}
