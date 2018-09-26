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
iQIzBAABCAAdFiEEpiHx2pbJPGOVBoMtYDRDodD8SYwFAlqw8c8ACgkQYDRDodD8
SYw/IxAAxJ1hvggrv2rUHDv69F1YJHd1jblQja0vMS5rCC5/mbUgUuP03JmGZ9Dy
6e8sIfv73AxhXrM70y0t0DXiSVaBdoxSf6B51VqwKS5fMexW2DWGkAf4IcGb4/g+
tVSPhKuuF/gLvoLKMfJMQ5ppLyaCD32IOrrzU1731FsnKHMyHqwYaoJCISyYEXDF
T36uiTNmqi4szZkUkFRqeo+9kZVCcuj2wkj+uYD6Or4W3/RLg5GxJANNT2T6ZvCE
K1tgOxPPxtGZGfscT1cyXUyzqmc1PJHivF28mYmhrjrV1txq5GfPgMGVjrXGkdoB
AiXwUCXsENObN15/BvwnDEwmwx7bnuthlIj+lzLMwtYu/Gqv1kd8UHEddbHQPzdY
LOGauqKWoIY/gKJ9baYInOguZEcY7CCYDjG+hSR/OLAqe9mghmc7Nd/9X/LbTTZ8
/AB5fxefeySuTAQ129xY6Dvp09JzPWO6WILugG7VdYxGYmrkMxn4GSq1YZmz5tKJ
eR5Wtk50NjZT+RNxM6tL4cgxIGyIZGw1+9DF2ZeyP11dImO90rCrHOno3jWldtLu
4LuIO6kvR4quK6rsJLyl1tn11zW2i4sX+obRbq2OWpA9t/rUzDFrn+mNrOXcQWAx
uKih9X+v/kmYyJU3U+E582//E7LfoQRtWsPXZA5c4tl+8VRQPF4=
=8F+V
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
