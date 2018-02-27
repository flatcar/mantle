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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/crypto/openpgp"
)

// Flatcar image signing key:
// $ gpg2 --list-keys --list-options show-unusable-subkeys \
//     --keyid-format SHORT F88CFEDEFF29A5B4D9523864E25D9AED0593B34A
// pub   rsa4096/0593B34A 2018-02-26 [SC]
//       F88CFEDEFF29A5B4D9523864E25D9AED0593B34A
// uid         [ unknown] Flatcar Buildbot (Official Builds) <buildbot@flatcar-linux.org>
// sub   rsa4096/064D542D 2018-02-26 [S] [revoked: 2018-03-14]
// sub   rsa4096/D0FC498C 2018-03-14 [S] [expires: 2019-03-14]
const buildbot_coreos_PubKey = `
-----BEGIN PGP PUBLIC KEY BLOCK-----

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
Tg4yXDs=
=JaX/
-----END PGP PUBLIC KEY BLOCK-----
`

func Verify(signed, signature io.Reader, key string) error {

	keyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(key))

	if err != nil {
		panic(err)
	}

	_, err = openpgp.CheckDetachedSignature(keyring, signed, signature)
	return err
}

func VerifyFile(file, verifyKeyFile string) error {
	signed, err := os.Open(file)
	if err != nil {
		return err
	}
	defer signed.Close()

	signature, err := os.Open(file + ".sig")
	if err != nil {
		return err
	}
	defer signature.Close()

	var key string
	if verifyKeyFile == "" {
		key = buildbot_coreos_PubKey
	} else {
		b, err := ioutil.ReadFile(verifyKeyFile)
		if err != nil {
			return fmt.Errorf("%v: %s", err, verifyKeyFile)
		}
		key = string(b[:])
	}

	if err := Verify(signed, signature, key); err != nil {
		return fmt.Errorf("%v: %s", err, file)
	}
	return nil
}
