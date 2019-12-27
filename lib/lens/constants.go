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
	"fmt"

	"github.com/gravitational/gravity/lib/constants"
)

const (
	// DefaultListenAddress is the default admission server listen address.
	DefaultListenAddress = "0.0.0.0"
	// DefaultListenPort is the default admission server listen port.
	DefaultListenPort = "5367"
)

var (
	// DefaultRegistry is the default registry address images get redirected to.
	DefaultRegistry = fmt.Sprintf("%v:%v", constants.RegistryDomainName,
		constants.DockerRegistryPort)
)
