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
iQIzBAABCAAdFiEEHhAN16Z3pvmlMsn5tR3jdwZNVC0FAlqV36wACgkQtR3jdwZN
VC3f1w//daSZoWPC0WSSwS6nLG8RdpqqMYF4Cpowl4MQktxK5LYX3MMn9+5Ui9rK
EeztggOKffldunajYgqaXr1SM2a9DHMcXeIRjSXjQN8L5UG5efsIzZYiCP6g9rgf
qDWlbnEJD93BCylRaDAgqHAph9g8liJyOCZFtogjIZIIVjxMtiPWBEb7eGRiYhTw
R97z3/aweEbku2tA0zHtpXYnuwEvtgM7yHeUkdqiZkh7g01d8nOpd3T7UBxAHWzR
O/W7Z3n8e8CrGE8nXRoq77kpUU6gxrqHH3TDlera3Ns0mM1N5ve1vkF/uD+YzMwM
DZjpSE3sMrjnU3hqNrNwWkpQEFyVqw3h7pvUZxnTiB2AbCZcZ+qz4IgzjIttvwjW
JfUCxK1HnNYNxrGiOj8wnnG47auUFmOZQJaBvVe66xp93eqq4J6lFUK+kiu2MCAL
tCY9dMKCQsTRY/x+3r2ZNfNjRfQgjwrBveI3hjfA3Bzc3S81LIehHWu+JgCVlYbY
WhXIlxZbKJ//J6eqDU/DLEAgMs+kDirzHIFxJYTTLjG/7KTRPY4xhpWV4DzecIpw
Gn8WtwVSj9Mm/9Mnwi8MEFTRmIoaJrO+xO5xprAGsBB1FO3Pu9wA8wQoU5WWj3fb
wgf9v9a7Pjoi88Z+DzAYqkf+rGPwS52YPMMfwb+f7Fwnlz+5cF4=
=xF+Z
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
