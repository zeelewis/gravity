/*
Copyright 2018 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helm

import (
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/engine"
	"helm.sh/helm/v3/pkg/getter"

	"github.com/gravitational/trace"
)

// RenderParameters defines parameters to render Helm template.
type RenderParameters struct {
	// Path is a chart path.
	Path string
	// Values is a list of YAML files with values.
	Values []string
	// Set is a list of values set on the CLI.
	Set []string
}

// Render renders templates of a provided Helm chart.
func Render(params RenderParameters) (map[string]string, error) {
	chartRequested, err := loader.Load(params.Path)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	valueOpts := &values.Options{
		ValueFiles: params.Values,
		Values:     params.Set,
	}

	vals, err := valueOpts.MergeValues(getter.All(cli.New()))
	if err != nil {
		return nil, trace.Wrap(err)
	}

	renderedTemplates, err := engine.Render(chartRequested, vals)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	result := make(map[string]string)
	for k, v := range renderedTemplates {
		filename := filepath.Base(k)
		// Render only Kubernetes resources skipping internal Helm
		// files and files that begin with underscore which are not
		// expected to output a Kubernetes spec.
		if filename == "NOTES.txt" || strings.HasPrefix(filename, "_") {
			continue
		}
		result[k] = v
	}
	return result, nil
}
