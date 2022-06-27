// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package storage

import "context"

// Storage defines a medium used to store the data temporary
// created for a test.
type Storage interface {
	// Upload the []byte data to the implemented storage under the given name.
	// The contentType is used by the underlying implementation to store and distribute the data
	// in the right format.
	Upload(name, contentType string, data []byte) (string, string, error)

	// Delete the data associated to the name.
	Delete(ctx context.Context, name string) error

	// Close terminates eventual opened connections.
	Close() error
}
