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

package infra

import (
	"context"
	"time"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmconfig "github.com/Mirantis/pelagia/pkg/controller/config"
)

const (
	controllerImageVar = "CONTROLLER_IMAGE"
)

var (
	// common infra vars
	requeueAfterInterval = 30 * time.Second
	revisionHistoryLimit = int32(5)
	rootUserID           = int64(0)
	rookUserID           = int64(2016)
	trueVar              = true
	falseVar             = false
)

type cephDeploymentInfraConfig struct {
	context     context.Context
	api         *ReconcileLcmResources
	log         *zerolog.Logger
	lcmConfig   *lcmconfig.LcmConfig
	infraConfig infraConfig
}

type infraConfig struct {
	name            string
	namespace       string
	lcmOwnerRefs    []metav1.OwnerReference
	cephOwnerRefs   []metav1.OwnerReference
	externalCeph    bool
	osdPlacement    cephv1.Placement
	cephImage       string
	controllerImage string
}
