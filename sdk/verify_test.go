// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

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
iQIzBAABCAAdFiEEYozCEpOAZdq047lJqKvwBYluOU8FAlu3c0gACgkQqKvwBYlu
OU/1wA/+Ow8Z55z915Ags2u641x+MHzJ5biQde8j81w7CbCMvYXQcv3O++mh++A+
rXqYwnfjCaK9+ueQbiZj2hhA0AEvM+sSqRdtz25geK0ZQYGImk97C+ABxZGWs0t6
6PPxrpGoNRAWZmP/ZOjSfZfCHmD+N0D9YR8MpT/1UZWc34a8U4DxIC9QJSRFIt91
wM31ZUUVvHZYPhnFyhZr6M6Mp2SqT9lNO8QxTdNyPOEInG0Nn3FdgERIkr/kYsQE
LWOp3N0SND6V0DKd0if0kZIFeRfI4aiIO93M5J1LdHmKwIL2yTsYfOd0Ci8VsN0F
RHUtq/5kb9/veNksVOGuNdCGT1EbmtJLBUMfzZ2RKKXftFADmMg8TxOMxt7e1N0G
TXTXC+znSCAJBxo0PM0oIascEWZb1yUxIEGFYBuGbTvGHL/W4PcUAe0lfIYRCQn3
GFCcezOVm7/VCaB0pLMEVYn5ltTqJb3hEMbQMjpQ3YFpBODPpFd8Iu9w/07hYsmB
i9NONTQXGvxRCpMFx3HHPFhOCEVuPm95t8vmHWm+roC54gT9dtdZjmc9lho+tK7r
OnqfjZvLtvbe6lipz6NLCF+PCTFucz3ZJNOYzoobrbXGXZVkXcEFhWjSK9SrynLZ
mxngCncc/vD2AxsEo/wvid6qykYdev1bOraog0P3rCIp9OYqnkQ=
=x2Ob
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
