// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package storage

type Error struct {
	Op  string
	URL string
	Err error
}

func (e *Error) Error() string {
	return e.Op + " " + e.URL + ": " + e.Err.Error()
}
