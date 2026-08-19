// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package oraclecloud

import (
	"testing"

	"github.com/oracle/oci-go-sdk/v65/core"
)

func TestLaunchOptionsForBoard(t *testing.T) {
	tests := []struct {
		name     string
		board    string
		firmware core.LaunchOptionsFirmwareEnum
		wantNil  bool
		wantErr  bool
	}{
		{name: "empty preserves defaults", wantNil: true},
		{name: "amd64 preserves defaults", board: amd64Board, wantNil: true},
		{name: "arm64 uses UEFI", board: arm64Board, firmware: core.LaunchOptionsFirmwareUefi64},
		{name: "unknown board", board: "riscv64-usr", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, err := launchOptionsForBoard(tt.board)
			if (err != nil) != tt.wantErr {
				t.Fatalf("launchOptionsForBoard() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if tt.wantNil {
				if options != nil {
					t.Fatalf("launchOptionsForBoard() = %#v, want nil", options)
				}
				return
			}
			if options == nil {
				t.Fatal("launchOptionsForBoard() returned nil")
			}
			if options.Firmware != tt.firmware {
				t.Errorf("Firmware = %q, want %q", options.Firmware, tt.firmware)
			}
			if options.BootVolumeType != core.LaunchOptionsBootVolumeTypeParavirtualized {
				t.Errorf("BootVolumeType = %q, want PARAVIRTUALIZED", options.BootVolumeType)
			}
			if options.NetworkType != core.LaunchOptionsNetworkTypeParavirtualized {
				t.Errorf("NetworkType = %q, want PARAVIRTUALIZED", options.NetworkType)
			}
			if options.RemoteDataVolumeType != core.LaunchOptionsRemoteDataVolumeTypeParavirtualized {
				t.Errorf("RemoteDataVolumeType = %q, want PARAVIRTUALIZED", options.RemoteDataVolumeType)
			}
			if options.IsConsistentVolumeNamingEnabled == nil || !*options.IsConsistentVolumeNamingEnabled {
				t.Error("IsConsistentVolumeNamingEnabled is not true")
			}
			if options.IsPvEncryptionInTransitEnabled != nil {
				t.Error("IsPvEncryptionInTransitEnabled must not be overridden")
			}
		})
	}
}

func TestArm64ImageCapabilitySchemaData(t *testing.T) {
	want := map[string]string{
		"Compute.Firmware":             "UEFI_64",
		"Compute.LaunchMode":           "PARAVIRTUALIZED",
		"Network.AttachmentType":       "PARAVIRTUALIZED",
		"Storage.BootVolumeType":       "PARAVIRTUALIZED",
		"Storage.RemoteDataVolumeType": "PARAVIRTUALIZED",
	}

	data := arm64ImageCapabilitySchemaData()
	if len(data) != len(want) {
		t.Fatalf("schema has %d entries, want %d", len(data), len(want))
	}
	for name, value := range want {
		descriptor, ok := data[name]
		if !ok {
			t.Errorf("schema is missing %q", name)
			continue
		}
		enum, ok := descriptor.(core.EnumStringImageCapabilitySchemaDescriptor)
		if !ok {
			t.Errorf("schema %q has type %T, want EnumStringImageCapabilitySchemaDescriptor", name, descriptor)
			continue
		}
		if enum.Source != core.ImageCapabilitySchemaDescriptorSourceImage {
			t.Errorf("schema %q source = %q, want IMAGE", name, enum.Source)
		}
		if enum.DefaultValue == nil || *enum.DefaultValue != value {
			t.Errorf("schema %q default = %v, want %q", name, enum.DefaultValue, value)
		}
		if len(enum.Values) != 1 || enum.Values[0] != value {
			t.Errorf("schema %q values = %v, want [%q]", name, enum.Values, value)
		}
	}
}

func TestValidateBoard(t *testing.T) {
	for _, board := range []string{amd64Board, arm64Board} {
		if err := validateBoard(board); err != nil {
			t.Errorf("validateBoard(%q) returned %v", board, err)
		}
	}
	if err := validateBoard("riscv64-usr"); err == nil {
		t.Error("validateBoard() accepted unsupported board")
	}
}
