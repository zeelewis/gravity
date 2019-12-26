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

package lens

import (
	clusterv1beta1 "github.com/gravitational/gravity/lib/client/clientset/versioned"

	"github.com/gravitational/trace"
	"k8s.io/client-go/rest"
)

type AdmissionServerConfig struct {
	Config *rest.Config
}

func (c *AdmissionServerConfig) Check() error {
	if c.Config == nil {
		return trace.BadParameter("missing Config")
	}
	return nil
}

type AdmissionServer struct {
	AdmissionServerConfig
	clusterClient *clusterv1beta1.Clientset
}

func NewAdmissionServer(config AdmissionServerConfig) (*AdmissionServer, error) {
	if err := config.Check(); err != nil {
		return trace.Wrap(err)
	}
	clusterClient, err := clusterv1beta1.NewForConfig(config.Config)
	if err != nil {
		return trace.Wrap(err)
	}
	return &AdmissionServer{
		AdmissionServerConfig: config,
		clusterClient:         clusterClient,
	}, nil
}
