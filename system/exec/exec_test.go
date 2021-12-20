// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"context"
	"os/exec"
	"syscall"
	"testing"
)

func TestExecCmdKill(t *testing.T) {
	cmd := Command("sleep", "3600")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := cmd.Kill(); err != nil {
		t.Errorf("Kill failed: %v", err)
	}

	if cmd.ProcessState == nil {
		t.Fatalf("ProcessState is nil")
	}

	status := cmd.ProcessState.Sys().(syscall.WaitStatus)
	if status.Signal() != syscall.SIGKILL {
		t.Errorf("Unexpected state: %s", cmd.ProcessState)
	}
}

func TestExecCmdCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := CommandContext(ctx, "sleep", "3600")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	cancel()
	if err := cmd.Wait(); err == nil {
		t.Errorf("Killed without an error")
	} else if state, ok := err.(*exec.ExitError); ok {
		status := state.Sys().(syscall.WaitStatus)
		if status.Signal() != syscall.SIGKILL {
			t.Errorf("Unexpected state: %s", state)
		}
	} else {
		t.Errorf("Unexpected error: %v", err)
	}
}
