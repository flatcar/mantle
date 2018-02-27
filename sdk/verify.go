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
// uid         [ultimate] Flatcar Buildbot (Official Builds) <buildbot@flatcar-linux.org>
// sub   rsa4096/064D542D 2018-02-26 [S] [expires: 2019-02-26]
const buildbot_coreos_PubKey = `
-----BEGIN PGP PUBLIC KEY BLOCK-----

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
sKT7yjxBy2rzil4H+o155ks4uRm5wYE=
=tHZ4
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
