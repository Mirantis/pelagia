/*
Copyright 2025 Mirantis IT.

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
	"flag"
	"fmt"
	"os"
	"runtime"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	"github.com/Mirantis/pelagia/pkg/connector"
	lcmversion "github.com/Mirantis/pelagia/version"
)

func main() {
	log := lcmcommon.InitLogger(false)

	var rookNamespace, clientName, rgwUserName string
	var useRbd, useCephFS, useRgw, encodedBase64, version bool
	flag.StringVar(&rookNamespace, "rook-namespace", "rook-ceph", "Rook namespace")
	flag.StringVar(&clientName, "client-name", "", "name of ceph client which will be used for connecting to cluster, without 'client' prefix")
	flag.BoolVar(&useRbd, "use-rbd", true, "allow to consume Ceph RBD")
	flag.BoolVar(&useCephFS, "use-cephfs", false, "allow to consume CephFS")
	flag.StringVar(&rgwUserName, "rgw-username", "rgw-admin-ops-user", "rgw username to share keys for RGW connection")
	flag.BoolVar(&useRgw, "use-rgw", false, "allow to consume Ceph RGW")
	flag.BoolVar(&encodedBase64, "base64", false, "show connection string as base64 encoded")
	flag.BoolVar(&version, "version", false, "show binary version")
	flag.Parse()

	if version {
		log.Info().Msgf("Connector code version: %s", lcmversion.Version)
		log.Info().Msgf("Go Version: %s", runtime.Version())
		log.Info().Msgf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	if clientName == "" {
		log.Fatal().Msg("argument '--client-name' is required, but not set")
		os.Exit(1)
	}

	if !useRbd && !useCephFS && !useRgw {
		log.Fatal().Msg("at least one mode is required to set: '--use-rbd', '--use-cephfs', '--use-rgw'")
		os.Exit(1)
	}

	c, err := connector.GetConnector()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize connector")
		os.Exit(1)
	}

	opts := connector.Opts{
		RookNamespace: rookNamespace,
		AuthClient:    clientName,
		UseRBD:        useRbd,
		UseCephFS:     useCephFS,
		UseRgw:        useRgw,
		RgwUserName:   rgwUserName,
		EncodedBase64: encodedBase64,
	}

	s, err := c.PrepareConnectionString(opts)
	if err != nil {
		log.Fatal().Err(err).Msg("connection string failed to prepare")
		os.Exit(1)
	}
	fmt.Println(s)
}
