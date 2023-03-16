// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"net/http"
)

// SimpleHTTP provides a single http server.
type SimpleHTTP struct{}

func (s *SimpleHTTP) Serve() error {
	return http.ListenAndServe(":8080", http.FileServer(http.Dir("/var/www")))
}
