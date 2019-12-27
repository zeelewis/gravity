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
	"context"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/gravitational/gravity/lib/lens"

	"github.com/gravitational/trace"
	"k8s.io/client-go/tools/clientcmd"
)

type admissionServerConfig struct {
	listenAddress   string
	kubeConfigPath  string
	certificatePath string
	keyPath         string
	defaultRegistry string
}

func startAdmissionServer(config admissionServerConfig) error {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", config.kubeConfigPath)
	if err != nil {
		return trace.Wrap(err)
	}
	certificatePEM, err := ioutil.ReadFile(config.certificatePath)
	if err != nil {
		return trace.ConvertSystemError(err)
	}
	keyPEM, err := ioutil.ReadFile(config.keyPath)
	if err != nil {
		return trace.ConvertSystemError(err)
	}
	mutator, err := lens.NewMutator(lens.MutatorConfig{
		KubeConfig:      kubeConfig,
		DefaultRegistry: config.defaultRegistry,
	})
	if err != nil {
		return trace.Wrap(err)
	}
	server, err := lens.NewAdmissionServer(lens.AdmissionServerConfig{
		ListenAddress:  config.listenAddress,
		CertificatePEM: certificatePEM,
		KeyPEM:         keyPEM,
		Mutator:        mutator,
	})
	if err != nil {
		return trace.Wrap(err)
	}
	go server.ListenAndServe()
	// TODO(r0mant): Encapsulate inside server.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	server.Shutdown(context.Background())
	return nil
}
