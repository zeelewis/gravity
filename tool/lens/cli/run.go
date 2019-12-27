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
	"os"

	"github.com/gravitational/teleport/lib/utils"
	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
)

func Run(lens Application) error {
	cmd, err := lens.Parse(os.Args[1:])
	if err != nil {
		return trace.Wrap(err)
	}

	trace.SetDebug(*lens.Debug)
	if *lens.Debug {
		utils.InitLogger(utils.LoggingForDaemon, logrus.DebugLevel)
	} else {
		utils.InitLogger(utils.LoggingForDaemon, logrus.InfoLevel)
	}

	switch cmd {
	case lens.StartCmd.FullCommand():
		return startAdmissionServer(admissionServerConfig{
			listenAddress:   *lens.StartCmd.ListenAddress,
			kubeConfigPath:  *lens.StartCmd.KubeConfig,
			certificatePath: *lens.StartCmd.CertificatePath,
			keyPath:         *lens.StartCmd.KeyPath,
			defaultRegistry: *lens.StartCmd.DefaultRegistry,
		})
	}

	return trace.NotFound("unknown command %v", cmd)
}
