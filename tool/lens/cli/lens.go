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

	"github.com/gravitational/trace"
	"k8s.io/client-go/tools/clientcmd"
)

type admissionServerConfig struct {
	listenAddress  string
	kubeConfigPath string
}

func startAdmissionServer(config admissionServerConfig) error {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", config.kubeConfigPath)
	if err != nil {
		return trace.Wrap(err)
	}
	server, err := lens.NewAdmissionServer(lens.AdmissionServerConfig{
		Config: kubeConfig,
	})
	if err != nil {
		return trace.Wrap(err)
	}
	return nil
}
