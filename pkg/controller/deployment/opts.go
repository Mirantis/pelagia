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

package deployment

import (
	"context"
	"time"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"

	"github.com/rs/zerolog"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"

	lcmconfig "github.com/Mirantis/pelagia/pkg/controller/config"
)

var (
	requeueAfterInterval = 1 * time.Minute

	failTriesLeft  = 3
	currentFailTry = 0
)

// cephDeploymentConfig main type for reconcilation for each CephDeployment object
type cephDeploymentConfig struct {
	context   context.Context
	api       *ReconcileCephDeployment
	log       *zerolog.Logger
	lcmConfig *lcmconfig.LcmConfig
	cdConfig  deployConfig
}

type deployConfig struct {
	// cephDpl is a full cephdeployment object pointer
	cephDpl *cephlcmv1alpha1.CephDeployment
	// expanded node list w/o groups and labels, like it passed to ceph cluster
	nodesListExpanded []cephlcmv1alpha1.CephDeploymentNode
	// parsed currentCephVersion for current cephDpl
	currentCephVersion *lcmcommon.CephVersion
	// parsed ceph image for current cephDpl
	currentCephImage string
}
