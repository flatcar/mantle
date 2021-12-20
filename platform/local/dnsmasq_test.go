// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

// TODO: use package local_test to validate
package local

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateVethPair(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		pair, err := generateVethPair(32456, []*net.IPNet{
			&net.IPNet{
				IP:   net.IPv4(192, 168, 0, 1),
				Mask: net.IPv4Mask(255, 255, 255, 0),
			},
		})

		require.Nil(t, err)
		assert.Equal(t, []string{
			"kola-23236129|172.23.236.129/31",
			"kola-23236128|172.23.236.128/31",
			"172.23.236.128/32",
		}, pair)
	})
	t.Run("FailWithClash", func(t *testing.T) {
		_, err := generateVethPair(32456, []*net.IPNet{
			&net.IPNet{
				IP:   net.IPv4(172, 23, 236, 13),
				Mask: net.IPv4Mask(255, 255, 0, 0),
			},
		})
		require.NotNil(t, err)
		assert.ErrorIs(t, err, ErrIPClash)
	})
	t.Run("Fail", func(t *testing.T) {
		_, err := generateVethPair(70000, []*net.IPNet{
			&net.IPNet{
				IP:   net.IPv4(192, 168, 0, 1),
				Mask: net.IPv4Mask(255, 255, 255, 0),
			},
		})
		require.NotNil(t, err)
		assert.ErrorIs(t, err, ErrIncorrectSeed)
	})
}
