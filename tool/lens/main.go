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

package main

import (
	"log"
	"os"

	"github.com/gravitational/gravity/tool/common"
	"github.com/gravitational/gravity/tool/lens/cli"

	"github.com/gravitational/teleport/lib/utils"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	utils.InitLogger(utils.LoggingForCLI, logrus.WarnLevel)
	log.SetOutput(logrus.StandardLogger().Writer())
	app := kingpin.New("lens", "Mutating admission controller that rewrites registries in the container image references.")
	if err := run(app); err != nil {
		logrus.WithError(err).Error("Command failed.")
		common.PrintError(err)
		os.Exit(255)
	}
}

func run(app *kingpin.Application) error {
	lens := cli.RegisterCommands(app)
	return common.ProcessRunError(cli.Run(lens))
}
