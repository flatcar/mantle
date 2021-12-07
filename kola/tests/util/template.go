// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bytes"
	"text/template"
)

func ExecTemplate(tmplStr string, tmplData interface{}) (string, error) {
	var out bytes.Buffer

	tmpl, err := template.New("").Parse(tmplStr)
	if err != nil {
		return out.String(), err
	}

	if err := tmpl.Execute(&out, tmplData); err != nil {
		return out.String(), err
	}
	return out.String(), nil
}
