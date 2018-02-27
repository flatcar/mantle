// Copyright 2015 CoreOS, Inc.
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

package sdk

import (
	"encoding/base64"
	"io"
	"strings"
	"testing"

	"golang.org/x/crypto/openpgp/errors"
)

const (
	versionTxt = `COREOS_BUILD=723
COREOS_BRANCH=1
COREOS_PATCH=0
COREOS_VERSION=723.1.0
COREOS_VERSION_ID=723.1.0
COREOS_BUILD_ID=""
COREOS_SDK_VERSION=717.0.0
`
	versionSig = `
iQIzBAABCAAdFiEEpiHx2pbJPGOVBoMtYDRDodD8SYwFAlqpYXAACgkQYDRDodD8
SYz7mRAAnhSVH+AegPjL2LaZJRwkoy3ocQNUU3hXl/THTF6M517OJUFzOsL1kt0n
rAppAsBeo6iOojvy8Ly+Bzn1cn0mm5++E5JCHUSKvumpmg+KHqoNjYjg07+buHMI
F4GfneMqnJ51Q8jh6CnVguLWFCn6wsvCKGW/V9/XTFw7Ooml/ykWYJRWHbT1WLkf
UYz1ys2tiBFheQNIeb+UL+pH+6JNtTGPoAzS5nUuyzJ9HzxHq+3FLfrBykx3vU9y
CzIj+blZ4HfbZA4Msmm39ewkisf+BbIqgzTVvm4dTszX26AijEbAme+ZE9geT0/O
4v47zzWsz+2pRYz2bThOSVnpAHYPxC7LriWvN9ybec8Vj9fcem0WLh0ihcgQZ8c/
ygLvO6nV1hHwXN4WZ3O7Dl8aXh+IEml5q52lDIX57T4ol7n2syVBs+vDw82OUxVV
j3EQ3OeFx5nJNDq7O52VBRFKxNPQZk4U0iawuXqQiP8hQ25IKUnlH16xlojWiUMB
M3ra119ZJYZ5ox/tb+u5+RXUX8nwBDjakyf8O/rvvTTt3WrQ8qFgPhXWzBCGLaLU
LF4eJV8c33hio2YXFSt63rjh4b03oxxxHSL2gHcFbH22VXDV3miclTAhd4ZMjCwu
OVsUGvmgjrrrb5vUkTjpVI58Q/NuNlZ4hqpbLJM+EAvTO0GmzMw=
=VwCE
`
)

func b64reader(s string) io.Reader {
	return base64.NewDecoder(base64.StdEncoding, strings.NewReader(s))
}

func TestValidSig(t *testing.T) {
	err := Verify(strings.NewReader(versionTxt), b64reader(versionSig), buildbot_coreos_PubKey)
	if err != nil {
		t.Errorf("Verify failed: %v", err)
	}
}

func TestInvalidSig(t *testing.T) {
	err := Verify(strings.NewReader(versionTxt+"bad"), b64reader(versionSig), buildbot_coreos_PubKey)
	if err == nil {
		t.Errorf("Verify failed to report bad signature")
	} else if _, ok := err.(errors.SignatureError); !ok {
		t.Errorf("Verify failed: %v", err)
	}
}
