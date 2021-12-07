// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"github.com/coreos/go-omaha/omaha"
)

// OmahaWrapper wraps the omaha trivial server to log any errors returned by destroy
// and doesn't return anything instead
type OmahaWrapper struct {
	*omaha.TrivialServer
}

func (o OmahaWrapper) Destroy() {
	if err := o.TrivialServer.Destroy(); err != nil {
		plog.Errorf("Error destroying omaha server: %v", err)
	}
}
