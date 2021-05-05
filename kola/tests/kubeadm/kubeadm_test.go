// Copyright 2021 Kinvolk GmbH
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

package kubeadm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTemplate(t *testing.T) {
	t.Run("SuccessWithBase64", func(t *testing.T) {
		res, err := render(
			"Hello, {{ .World }}",
			map[string]interface{}{
				"World": "world !",
			},
			true,
		)
		require.Nil(t, err)
		assert.Equal(t, "SGVsbG8sIHdvcmxkICE=", res.String())

	})
	t.Run("Success", func(t *testing.T) {
		res, err := render(
			"Hello, {{ .World }}",
			map[string]interface{}{
				"World": "world !",
			},
			false,
		)
		require.Nil(t, err)
		assert.Equal(t, "Hello, world !", res.String())

	})
}
