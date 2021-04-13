// Copyright 2021 Kinvolk
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// local package import and it's behavior as a package

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
