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

import (
	"github.com/gravitational/gravity/lib/lens"

	"gopkg.in/alecthomas/kingpin.v2"
)

func RegisterCommands(app *kingpin.Application) Application {
	l := Application{
		Application: app,
	}

	l.Debug = app.Flag("debug", "Enable verbose logging output.").Bool()

	l.StartCmd.CmdClause = app.Command("start", "Start admission server.")
	l.StartCmd.ListenAddress = l.StartCmd.Flag("listen-address", "Address and port to listen on.").String()
	l.StartCmd.KubeConfig = l.StartCmd.Flag("kubeconfig", "Path to kubeconfig file with API server credentials. If not provided, in-cluster config will be used.").String()
	l.StartCmd.CertificatePath = l.StartCmd.Flag("cert-path", "Path to TLS certificate.").Required().String()
	l.StartCmd.KeyPath = l.StartCmd.Flag("key-path", "Path to TLS certificate private key.").Required().String()
	l.StartCmd.CAPath = l.StartCmd.Flag("ca-path", "Path to TLS CA certificate.").Required().String()
	l.StartCmd.DefaultRegistry = l.StartCmd.Flag("default-registry", "Default registry to rewrite images to.").Default(lens.DefaultRegistry).String()
	l.StartCmd.ServiceNamespace = l.StartCmd.Flag("service-namespace", "Lens service namespace for registering webhook.").Default(lens.DefaultServiceNamespace).String()
	l.StartCmd.ServiceName = l.StartCmd.Flag("service-name", "Lens service name for registering webhook.").Default(lens.DefaultServiceName).String()

	return l
}
