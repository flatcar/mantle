// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package neterror

import (
	"net"
)

// IsClosed detects if an error is due to a closed network connection,
// working around bug https://github.com/golang/go/issues/4373
func IsClosed(err error) bool {
	if err == nil {
		return false
	}
	if operr, ok := err.(*net.OpError); ok {
		err = operr.Err
	}
	// cry softly
	return err.Error() == "use of closed network connection"
}
