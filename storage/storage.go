// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

// storage provides a high level interface for Google Cloud Storage
package storage

import (
	"github.com/coreos/pkg/capnslog"
)

// Arbitrary limit on the number of concurrent remote API requests.
const MaxConcurrentRequests = 12

var plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "storage")
