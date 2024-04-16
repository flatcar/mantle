package local

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar/mantle/system/exec"
	"github.com/flatcar/mantle/util"
)

type SoftwareTPM struct {
	process        *exec.ExecCmd
	dirFromKolaCwd string
	dirFromTestDir string
}

func NewSwtpm(testDir string, tpmDir string) (*SoftwareTPM, error) {
	dirFromKolaCwd := filepath.Join(testDir, tpmDir)
	swtpm := &SoftwareTPM{dirFromKolaCwd: dirFromKolaCwd, dirFromTestDir: tpmDir}

	if err := os.Mkdir(swtpm.dirFromKolaCwd, 0700); err != nil {
		return nil, fmt.Errorf("Failed to create TPM dir: %v", err)
	}

	swtpm.process = exec.Command("swtpm", "socket", "--tpmstate", fmt.Sprintf("dir=./%v", swtpm.dirFromTestDir), "--ctrl", fmt.Sprintf("type=unixio,path=./%v", swtpm.SocketRelativePathFromTestDir()), "--tpm2")
	// Use the test directory as current working directory
	// so that we don't have a socket path argument that
	// exceeds 108 chars which is the limit for UNIX sockets
	// (Using ./ as prefix helps to know that these are relative
	// path arguments).
	swtpm.process.Dir = testDir
	plog.Debugf("Prepared swtpm process %q with CWD %q", swtpm.process, swtpm.process.Dir)
	out, err := swtpm.process.StderrPipe()
	if err != nil {
		return nil, err
	}
	go util.LogFrom(capnslog.INFO, out)

	if err = swtpm.process.Start(); err != nil {
		return nil, err
	}

	plog.Debugf("swtpm PID: %v", swtpm.process.Pid())

	return swtpm, nil
}

func (swtpm *SoftwareTPM) Stop() {
	if err := swtpm.process.Kill(); err != nil {
		plog.Errorf("Error killing swtpm: %v", err)
	}
	plog.Debugf("Delete swtpm temporary directory %v", swtpm.dirFromKolaCwd)
	os.RemoveAll(swtpm.dirFromKolaCwd)
}

func (swtpm *SoftwareTPM) SocketRelativePathFromTestDir() string {
	const socket string = "socket"
	return filepath.Join(swtpm.dirFromTestDir, socket)
}
