/*
Copyright 2025 The Mirantis Authors.

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

package osdremove

import (
	"context"
	"time"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/rs/zerolog"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmconfig "github.com/Mirantis/pelagia/pkg/controller/config"
)

const diskCleanupJobLabel = "pelagia-lcm-cleanup-disks"

// interval for reconcile
var requeueAfterInterval = 1 * time.Minute

// cephDeploymentHealthConfig main type for health reconcilation for each CephDeploymentHealth object
type cephOsdRemoveConfig struct {
	context    context.Context
	api        *ReconcileCephOsdRemoveTask
	log        *zerolog.Logger
	lcmConfig  *lcmconfig.LcmConfig
	taskConfig taskConfig
}

type taskConfig struct {
	task                  *lcmv1alpha1.CephOsdRemoveTask
	cephCluster           *cephv1.CephCluster
	cephHealthOsdAnalysis *lcmv1alpha1.OsdSpecAnalysisState
	requeueNow            bool
}
