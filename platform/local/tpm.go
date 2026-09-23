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
}

func NewSwtpm(testDir string, tpmDir string) (*SoftwareTPM, error) {
	dirFromKolaCwd, err := os.MkdirTemp("", "mantle-tpm-")
	if err != nil {
		return nil, fmt.Errorf("Failed to create TPM temp dir: %v", err)
	}
	swtpm := &SoftwareTPM{dirFromKolaCwd: dirFromKolaCwd}

	swtpm.process = exec.Command("swtpm", "socket", "--tpmstate", fmt.Sprintf("dir=%v", swtpm.dirFromKolaCwd), "--ctrl", fmt.Sprintf("type=unixio,path=%v", swtpm.SocketPath()), "--tpm2")
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

func (swtpm *SoftwareTPM) SocketPath() string {
	return filepath.Join(swtpm.dirFromKolaCwd, "socket")
}
