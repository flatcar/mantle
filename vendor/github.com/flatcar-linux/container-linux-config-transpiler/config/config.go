// Copyright 2016 CoreOS, Inc.
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

package config

import (
	"reflect"

	ignTypes "github.com/flatcar-linux/ignition/config/v2_3/types"
	"github.com/flatcar-linux/ignition/config/validate"
	"github.com/flatcar-linux/ignition/config/validate/astnode"
	"github.com/flatcar-linux/ignition/config/validate/report"
	yaml "gopkg.in/yaml.v3"

	"github.com/flatcar-linux/container-linux-config-transpiler/config/astyaml"
	"github.com/flatcar-linux/container-linux-config-transpiler/config/platform"
	"github.com/flatcar-linux/container-linux-config-transpiler/config/types"
)

// Parse will convert a byte slice containing a Container Linux Config into a
// golang struct representing the config, the parse tree from parsing the yaml
// and a report of any warnings or errors that occurred during the parsing.
func Parse(data []byte) (types.Config, astnode.AstNode, report.Report) {
	var docNode yaml.Node

	if err := yaml.Unmarshal(data, &docNode); err != nil {
		return types.Config{}, nil, report.ReportFromError(err, report.EntryError)
	}

	var cfg types.Config

	if err := docNode.Decode(&cfg); err != nil {
		return types.Config{}, nil, report.ReportFromError(err, report.EntryError)
	}

	var root astnode.AstNode
	var r report.Report
	if docNode.IsZero() {
		r.Add(report.Entry{
			Kind:    report.EntryWarning,
			Message: "Configuration is empty",
		})
		r.Merge(validate.ValidateWithoutSource(reflect.ValueOf(cfg)))
	} else {
		var err error
		root, err = astyaml.FromYamlDocumentNode(docNode)
		if err != nil {
			return types.Config{}, nil, report.ReportFromError(err, report.EntryError)
		}

		r.Merge(validate.Validate(reflect.ValueOf(cfg), root, nil, true))
	}

	if r.IsFatal() {
		return types.Config{}, nil, r
	}
	return cfg, root, r
}

// Convert will convert a golang struct representing a Container Linux
// Config into an Ignition Config, and a report of any warnings or errors. It
// takes the parse tree from parsing the Container Linux config as well.
// Convert also accepts a platform string, which can either be one of the
// platform strings defined in config/templating/templating.go or an empty
// string if [dynamic data](doc/dynamic-data.md) isn't used.
func Convert(in types.Config, p string, ast astnode.AstNode) (ignTypes.Config, report.Report) {
	if !platform.IsSupportedPlatform(p) {
		r := report.Report{}
		r.Add(report.Entry{
			Kind:    report.EntryError,
			Message: "unsupported platform",
		})
		return ignTypes.Config{}, r
	}
	return types.Convert(in, p, ast)
}
