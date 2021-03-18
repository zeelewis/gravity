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
	"bytes"
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/releaseutil"

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
	settings := cli.New()
	// TODO: namespace?
	actionConfig, err := helmInit(settings, "")
	if err != nil {
		return nil, trace.Wrap(err)
	}

	client := action.NewInstall(actionConfig)
	client.Timeout = helmTimeout
	client.DryRun = true // Skip the name check
	client.ReleaseName = "RELEASE-NAME"
	client.Replace = true
	client.ClientOnly = true

	valueOpts := &values.Options{
		ValueFiles: params.Values,
		Values:     params.Set,
	}

	rel, err := runInstall(settings, client, params.Path, valueOpts)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	var manifests bytes.Buffer
	fmt.Fprintln(&manifests, strings.TrimSpace(rel.Manifest))
	return releaseutil.SplitManifests(manifests.String()), nil
}
