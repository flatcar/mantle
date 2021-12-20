// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package targen

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strings"

	"github.com/flatcar-linux/mantle/system/exec"
)

// return a slice of strings that are the library dependencies of binary.
// returns a nil slice if the binary is static
func ldd(binary string) ([]string, error) {
	c := exec.Command("ldd", binary)

	// let's not get LD_PRELOAD invovled.
	c.Env = []string{}

	out, err := c.CombinedOutput()
	if err != nil {
		// static binaries have no libs (barring dlopened ones, which we can't find)
		if strings.Contains(string(out), "not a dynamic executable") {
			return nil, nil
		}

		return nil, err
	}

	buf := bytes.NewBuffer(out)
	sc := bufio.NewScanner(buf)
	sc.Split(bufio.ScanWords)

	var libs []string
	for sc.Scan() {
		w := sc.Text()
		if filepath.IsAbs(w) {
			libs = append(libs, w)
		}
	}

	return libs, nil
}
