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

package health

import (
	"context"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/rs/zerolog"

	lcmconfig "github.com/Mirantis/pelagia/pkg/controller/config"
)

// cephDeploymentHealthConfig main type for health reconcilation for each CephDeploymentHealth object
type cephDeploymentHealthConfig struct {
	context      context.Context
	api          *ReconcileCephDeploymentHealth
	log          *zerolog.Logger
	lcmConfig    *lcmconfig.LcmConfig
	healthConfig healthConfig
}

type healthConfig struct {
	name                 string
	namespace            string
	cephCluster          *cephv1.CephCluster
	rgwOpts              rgwOpts
	sharedFilesystemOpts sharedFilesystemOpts
}

type rgwOpts struct {
	storeName         string
	desiredRgwDaemons int32
	multisite         bool
	external          bool
	externalEndpoint  string
}

type sharedFilesystemOpts struct {
	mdsStandbyDesired int
	mdsDaemonsDesired map[string]map[string]int
}

const (
	cephDaemonsCheck          = "ceph_daemons"
	cephCSIPluginDaemonsCheck = "ceph_csi_plugin_daemons"
	usageDetailsCheck         = "usage_details"
	cephEventsCheck           = "ceph_events"
	poolReplicasCheck         = "pools_replicas"
	rgwInfoCheck              = "rgw_info"
	specAnalysisCheck         = "spec_analysis"
)
