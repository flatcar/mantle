package local

import (
	"fmt"
	"os"

	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar/mantle/system/exec"
	"github.com/flatcar/mantle/util"
)

type SoftwareTPM struct {
	process    *exec.ExecCmd
	socketPath string
	dir        string
}

func NewSwtpm(dir string) (*SoftwareTPM, error) {
	swtpm := &SoftwareTPM{}

	os.Mkdir(dir, 0700)
	swtpm.dir = dir
	swtpm.socketPath = fmt.Sprintf("%v/sock", swtpm.dir)

	swtpm.process = exec.Command("swtpm", "socket", "--tpmstate", fmt.Sprintf("dir=%v", swtpm.dir), "--ctrl", fmt.Sprintf("type=unixio,path=%v", swtpm.socketPath), "--tpm2")
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
	plog.Debugf("Delete swtpm temporary directory %v", swtpm.dir)
	os.RemoveAll(swtpm.dir)
}

func (swtpm *SoftwareTPM) SocketPath() string {
	return swtpm.socketPath
}
