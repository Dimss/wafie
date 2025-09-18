// Copyright 2017 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This is a sample chained plugin that supports multiple CNI versions. It
// parses prevResult according to the cniVersion
package main

import (
	"log"
	"os"

	"github.com/Dimss/wafie/cni/pkg/plugin"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
)

var logger = log.New(os.Stderr, "[wafie-cni] ", log.LstdFlags)

func main() {
	if err := runPlugin(); err != nil {
		logger.Printf("plugin exited with error: %v", err)
		os.Exit(1)
	}
}

func runPlugin() error {
	funcs := skel.CNIFuncs{
		Add:   plugin.CmdAdd,
		Del:   plugin.CmdDel,
		Check: plugin.CmdCheck,
	}
	err := skel.PluginMainFuncsWithError(funcs, version.All, "CNI plugin wafie-cni v0.0.1")
	if err != nil {
		logger.Printf("istio-cni failed with: %v", err)
		if err := err.Print(); err != nil {
			logger.Printf("istio-cni failed to write error JSON to stdout: %v", err)
		}
		return err
	}
	return nil
}
