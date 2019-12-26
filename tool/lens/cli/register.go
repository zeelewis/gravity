/*
Copyright 2019 Gravitational, Inc.

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

package cli

import "gopkg.in/alecthomas/kingpin.v2"

func RegisterCommands(app *kingpin.Application) Application {
	lens := Application{
		Application: app,
	}

	lens.Debug = app.Flag("debug", "Enable verbose logging output.").Bool()

	lens.StartCmd.CmdClause = app.Command("start", "Start admission server.")
	lens.StartCmd.ListenAddress = lens.StartCmd.Flag("listen-address", "Address and port to listen on.").String()
	lens.StartCmd.KubeConfig = lens.StartCmd.Flag("kubeconfig", "Path to kubeconfig file with API server credentials. If not provided, in-cluster config will be used.").String()

	return lens
}
