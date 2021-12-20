// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package exec

import (
	"fmt"
	"os"
	"testing"
)

var testMulticall Entrypoint

func init() {
	testMulticall = NewEntrypoint("testMulticall", testMulticallHelper)
}

func testMulticallHelper(args []string) error {
	fmt.Println(args)
	return nil
}

func TestMain(m *testing.M) {
	MaybeExec()
	os.Exit(m.Run())
}

func TestMulticallNoArgs(t *testing.T) {
	cmd := testMulticall.Command()
	out, err := cmd.Output()
	if err != nil {
		t.Error(err)
	}
	if string(out) != "[]\n" {
		t.Errorf("Unexpected output: %q", string(out))
	}
}

func TestMulticallWithArgs(t *testing.T) {
	cmd := testMulticall.Command("arg1", "020")
	out, err := cmd.Output()
	if err != nil {
		t.Error(err)
	}
	if string(out) != "[arg1 020]\n" {
		t.Errorf("Unexpected output: %q", string(out))
	}
}
