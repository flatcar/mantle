// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package torcx

import (
	"encoding/json"
	"errors"
)

const packageListKind = "torcx-package-list-v0"

type Manifest struct {
	Packages []Package
}

// a copy of the Manifest type with no UnmarshalJSON method so the below doesn't do a round of recursion
type torcxManifestUnmarshalValue struct {
	Packages []Package
}

func (t *Manifest) UnmarshalJSON(b []byte) error {
	if t == nil {
		return errors.New("Unmarshal(nil *Manifest)")
	}
	wrappingType := struct {
		Kind  string                      `json:"kind"`
		Value torcxManifestUnmarshalValue `json:"value"`
	}{}

	if err := json.Unmarshal(b, &wrappingType); err != nil {
		return err
	}
	if wrappingType.Kind != packageListKind {
		return errors.New("Unrecognized torcx packagelist kind: " + wrappingType.Kind)
	}

	t.Packages = wrappingType.Value.Packages
	return nil
}

type Package struct {
	Name           string
	DefaultVersion *string
	Versions       []Version
}

type Version struct {
	Version       string
	Hash          string
	CasDigest     string
	SourcePackage string
	Locations     []Location
}

type Location struct {
	Path *string
	URL  *string
}
