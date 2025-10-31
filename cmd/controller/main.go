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
	"os"
	"runtime"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	rookapi "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	lcmapi "github.com/Mirantis/pelagia/pkg/apis"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	lcmcontroller "github.com/Mirantis/pelagia/pkg/controller"
	lcmversion "github.com/Mirantis/pelagia/version"
)

func main() {
	log := lcmcommon.InitLogger(false)
	log.Info().Msgf("Contoller code version: %s", lcmversion.Version)
	log.Info().Msgf("Go Version: %s", runtime.Version())
	log.Info().Msgf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)

	var controllerName, leaderElectionID string
	flag.StringVar(&controllerName, "controller-name", "", "controller name")
	flag.StringVar(&leaderElectionID, "leader-election-id", "", "leader election id")
	flag.Parse()

	if controllerName == "" {
		log.Fatal().Msg("argument 'controller-name' is required, but not set")
		os.Exit(1)
	}
	if leaderElectionID == "" {
		log.Fatal().Msg("argument 'leader-election-id' is required, but not set")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("")
		os.Exit(1)
	}

	// env var WATCH_NAMESPACES specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	namespacesVar, found := os.LookupEnv("WATCH_NAMESPACES")
	if !found {
		log.Fatal().Msg("required env variable 'WATCH_NAMESPACES' is not set")
		os.Exit(1)
	}
	// cut unexpected spaces if any
	defaultNamespaces := map[string]cache.Config{}
	namespaces := strings.Split(namespacesVar, ",")
	for _, ns := range namespaces {
		defaultNamespaces[strings.TrimSpace(ns)] = cache.Config{}
	}

	// Set default manager options
	options := manager.Options{
		Cache:            cache.Options{DefaultNamespaces: defaultNamespaces},
		LeaderElection:   true,
		LeaderElectionID: leaderElectionID,
		Metrics: metricsserver.Options{
			// BindAddress is the bind address for controller runtime metrics server. Defaulted to "0" which is off.
			BindAddress: "0",
		},
	}

	// Create a new manager to provide shared dependencies and start components
	mgr, err := manager.New(cfg, options)
	if err != nil {
		log.Fatal().Err(err).Msg("")
		os.Exit(1)
	}

	log.Info().Msg("registering сomponents and APIs")

	// Setup Scheme for required resources
	if err := lcmapi.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatal().Err(err).Msg("")
		os.Exit(1)
	}

	if err := rookapi.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatal().Err(err).Msg("")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := lcmcontroller.AddToManager(mgr, controllerName); err != nil {
		log.Fatal().Err(err).Msg("")
		os.Exit(1)
	}

	log.Info().Msgf("starting controller '%s'", controllerName)

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatal().Err(err).Msg("")
		os.Exit(1)
	}
}
