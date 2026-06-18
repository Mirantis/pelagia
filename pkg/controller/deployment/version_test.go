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
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestVerifyCephVersions(t *testing.T) {
	tests := []struct {
		name                  string
		cephDpl               *cephlcmv1alpha1.CephDeployment
		inputResources        map[string]runtime.Object
		lcmConfigData         map[string]string
		cmdOutputs            map[string]string
		apiErrors             map[string]error
		osdpl                 *fakeclient.ClientBuilder
		expectedVersion       *lcmcommon.CephVersion
		expectedImage         string
		expectedStatusVersion string
		expectedError         string
	}{
		{
			name:          "ceph image is not set",
			expectedError: "Pelagia lcmconfig has no required 'DEPLOYMENT_CEPH_IMAGE' parameter set",
		},
		{
			name:          "failed to get cephcluster",
			cephDpl:       &unitinputs.CephDeployMosk,
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{}},
			},
			apiErrors:     map[string]error{"get-cephclusters": errors.New("failed to get cephcluster")},
			expectedError: "failed to get rook-ceph/cephcluster cephcluster: failed to get cephcluster",
		},
		{
			name:    "failed to find supported ceph version for specified image",
			cephDpl: unitinputs.CephDeployMosk.DeepCopy(),
			lcmConfigData: func() map[string]string {
				cm := unitinputs.PelagiaConfig.DeepCopy().Data
				cm["DEPLOYMENT_CEPH_RELEASE"] = "octopus"
				return cm
			}(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{}},
			},
			expectedError: "specified not supported Ceph release 'octopus'. Is version correct?",
		},
		{
			name:          "cephcluster not found, fresh deployment, failed to prepare version-check deployment",
			cephDpl:       &unitinputs.CephDeployMosk,
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{}},
				"deployments":  &appsv1.DeploymentList{},
			},
			apiErrors:     map[string]error{"get-deployments": errors.New("failed to get deployments")},
			expectedError: "failed to check 'ceph --version' for provided image 'mirantis.azurecr.io/ceph/ceph:v20.2.2': failed to prepare version-check deployment: failed to get 'lcm-namespace/pelagia-check-ceph-version' deployment: failed to get deployments",
		},
		{
			name:          "cephcluster not found, fresh deployment, incorrect ceph version inside image",
			cephDpl:       &unitinputs.CephDeployMosk,
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{}},
				"deployments":  &appsv1.DeploymentList{Items: []appsv1.Deployment{unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])}},
			},
			cmdOutputs:    map[string]string{"ceph --version": "ceph version 3.3.3 (stable)"},
			expectedError: "failed to check 'ceph --version' for provided image 'mirantis.azurecr.io/ceph/ceph:v20.2.2': unsupported Ceph major version 'v3.3' provided. Supported are: [Tentacle (v20.2) Squid (v19.2)]",
		},
		{
			name:          "cephcluster not found, fresh deployment, version detected",
			cephDpl:       &unitinputs.CephDeployMosk,
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{}},
				"deployments":  &appsv1.DeploymentList{Items: []appsv1.Deployment{unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])}},
			},
			cmdOutputs: map[string]string{"ceph --version": unitinputs.CephVersionLatest},
			expectedVersion: &lcmcommon.CephVersion{
				Name:         "Tentacle",
				MajorVersion: "v20.2",
				MinorVersion: "2",
				Order:        20,
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v20.2.2",
			expectedStatusVersion: "",
		},
		{
			name:          "cephcluster found, but not deployed yet, failed to detect version",
			cephDpl:       &unitinputs.CephDeployMosk,
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"deployments":  &appsv1.DeploymentList{Items: []appsv1.Deployment{unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])}},
				"configmaps":   &corev1.ConfigMapList{},
			},
			expectedError: "failed to check 'ceph --version' for used in cluster image 'mirantis.azurecr.io/ceph/ceph:v20.2.2': failed to run command 'ceph --version': unexpected run ceph command: ceph --version",
		},
		{
			name:          "cephcluster found, but not deployed yet, version detected",
			cephDpl:       &unitinputs.CephDeployMosk,
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"deployments":  &appsv1.DeploymentList{Items: []appsv1.Deployment{unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])}},
				"configmaps":   &corev1.ConfigMapList{},
			},
			cmdOutputs: map[string]string{"ceph --version": unitinputs.CephVersionLatest},
			expectedVersion: &lcmcommon.CephVersion{
				Name:         "Tentacle",
				MajorVersion: "v20.2",
				MinorVersion: "2",
				Order:        20,
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v20.2.2",
			expectedStatusVersion: "",
		},
		{
			name:          "cephcluster deployed, but pelagia toolbox deployment is not ready",
			cephDpl:       &unitinputs.CephDeployMosk,
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"deployments":  &appsv1.DeploymentList{},
				"configmaps":   &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			expectedError: "Pelagia toolbox deployment 'rook-ceph/pelagia-ceph-toolbox' is not ready, waiting before proceed any actions",
		},
		{
			name:          "cephcluster deployed, pelagia toolbox available, failed to get versions",
			cephDpl:       &unitinputs.CephDeployMosk,
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"configmaps":   &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments":  &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady}},
			},
			cmdOutputs:    map[string]string{"ceph versions --format json": "{||}"},
			expectedError: "failed to get current Ceph versions: failed to parse output for command 'ceph versions --format json': invalid character '|' looking for beginning of object key string",
		},
		{
			name:          "cephcluster not deployed and different image set",
			cephDpl:       &unitinputs.CephDeployMosk,
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cluster := unitinputs.CephClusterReady.DeepCopy()
						cluster.Spec.CephVersion.Image = "some-registry.com/ceph:v19.2.4"
						return *cluster
					}(),
				}},
				"deployments": &appsv1.DeploymentList{Items: []appsv1.Deployment{unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])}},
				"configmaps":  &corev1.ConfigMapList{},
			},
			cmdOutputs: map[string]string{"ceph --version": unitinputs.CephVersionPrevious},
			expectedVersion: &lcmcommon.CephVersion{
				Name:         "Squid",
				MajorVersion: "v19.2",
				MinorVersion: "4",
				Order:        19,
			},
			expectedImage:         "some-registry.com/ceph:v19.2.4",
			expectedStatusVersion: "",
		},
		{
			name:    "cephcluster image changed, but new release is different to image",
			cephDpl: &unitinputs.CephDeployMosk,
			lcmConfigData: func() map[string]string {
				cm := unitinputs.PelagiaConfigForPrevCephVersion.DeepCopy().Data
				cm["DEPLOYMENT_CEPH_IMAGE"] = "mirantis.azurecr.io/ceph/ceph:v19.2.3"
				cm["DEPLOYMENT_CEPH_RELEASE"] = "tentacle"
				return cm
			}(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"configmaps":   &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments":  &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady, unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])}},
			},
			cmdOutputs: map[string]string{
				"ceph versions --format json": unitinputs.CephVersionsLatest,
				"ceph --version":              unitinputs.CephVersionPrevious,
			},
			expectedError: "expected Ceph release Tentacle 'v20.2' version, but specified Squid 'v19.2' version (image: mirantis.azurecr.io/ceph/ceph:v19.2.3)",
		},
		{
			name:    "cephcluster image changed, but new release is major downgrade",
			cephDpl: &unitinputs.CephDeployMosk,
			lcmConfigData: func() map[string]string {
				cm := unitinputs.PelagiaConfigForPrevCephVersion.DeepCopy().Data
				cm["DEPLOYMENT_CEPH_IMAGE"] = "mirantis.azurecr.io/ceph/ceph:v19.2.3"
				cm["DEPLOYMENT_CEPH_RELEASE"] = "squid"
				return cm
			}(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"configmaps":   &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments":  &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady, unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])}},
			},
			cmdOutputs: map[string]string{
				"ceph versions --format json": unitinputs.CephVersionsLatest,
				"ceph --version":              unitinputs.CephVersionPrevious,
			},
			expectedError: "detected Ceph version downgrade from 'v20.2.2' to 'v19.2.4': major downgrade is not possible",
		},
		/* TODO: uncomment if more than 2 releases are supported at the time
		{
			name: "cephdeployment cluster version is set, but new release is upgrade step over one",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Status.ClusterVersion = "v17.2.7"
				return mc
			}(),
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
			},
			expectedError: "detected Ceph version upgrade from 'v17.2.7' to 'v20.2.0': upgrade with step over one major version is not possible",
		},*/
		{
			name:          "ceph image different from desired image, upgrade is not needed",
			cephDpl:       &unitinputs.CephDeployMosk,
			osdpl:         faketestclients.GetClientBuilder().WithLists(unitinputs.GetOpenstackDeploymentStatusList("cur", "APPLIED", true)),
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						c := unitinputs.TestCephCluster.DeepCopy()
						c.Spec.CephVersion.Image = "mirantis.azurecr.io/ceph/ceph:v20.2.2-0"
						return *c
					}(),
				}},
				"configmaps":  &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments": &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady, unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])}},
			},
			cmdOutputs: map[string]string{
				"ceph versions --format json": unitinputs.CephVersionsLatest,
				"ceph --version":              unitinputs.CephVersionLatest,
			},
			expectedVersion: &lcmcommon.CephVersion{
				Name:         "Tentacle",
				MajorVersion: "v20.2",
				MinorVersion: "2",
				Order:        20,
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v20.2.2",
			expectedStatusVersion: "v20.2.2",
		},
		{
			name:          "ceph image different from desired image, failed to check upgrade allowed",
			cephDpl:       &unitinputs.CephDeployMosk,
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						c := unitinputs.TestCephCluster.DeepCopy()
						c.Spec.CephVersion.Image = "some-registry/ceph:v19.2.3"
						return *c
					}(),
				}},
				"configmaps":  &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments": &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady, unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])}},
			},
			cmdOutputs: map[string]string{
				"ceph versions --format json": unitinputs.CephVersionsPrevious,
				"ceph --version":              unitinputs.CephVersionLatest,
			},
			expectedError: "failed to check is Ceph upgrade allowed: required env variable 'CEPH_CONTROLLER_CLUSTER_RELEASE' is not set",
		},
		{
			name:          "ceph image different from desired image, upgrade is not allowed",
			cephDpl:       &unitinputs.CephDeployMosk,
			osdpl:         faketestclients.GetClientBuilder().WithLists(unitinputs.GetOpenstackDeploymentStatusList("new", "APPLIED", true)),
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						c := unitinputs.TestCephCluster.DeepCopy()
						c.Spec.CephVersion.Image = "mirantis.azurecr.io/ceph/ceph:v19.2.4"
						return *c
					}(),
				}},
				"configmaps":  &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments": &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady, unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])}},
			},
			cmdOutputs: map[string]string{
				"ceph versions --format json": unitinputs.CephVersionsPrevious,
				"ceph --version":              unitinputs.CephVersionLatest,
			},
			expectedVersion: &lcmcommon.CephVersion{
				Name:         "Squid",
				MajorVersion: "v19.2",
				MinorVersion: "4",
				Order:        19,
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v19.2.4",
			expectedStatusVersion: "v19.2.4",
		},
		{
			name:          "ceph image different from desired image, upgrade is allowed",
			cephDpl:       &unitinputs.CephDeployMosk,
			osdpl:         faketestclients.GetClientBuilder().WithLists(unitinputs.GetOpenstackDeploymentStatusList("cur", "APPLIED", true)),
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						c := unitinputs.TestCephCluster.DeepCopy()
						c.Spec.CephVersion.Image = "mirantis.azurecr.io/ceph/ceph:v19.2.4"
						return *c
					}(),
				}},
				"configmaps":  &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments": &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady, unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])}},
			},
			cmdOutputs: map[string]string{
				"ceph versions --format json": unitinputs.CephVersionsPrevious,
				"ceph --version":              unitinputs.CephVersionLatest,
			},
			expectedVersion: &lcmcommon.CephVersion{
				Name:         "Squid",
				MajorVersion: "v19.2",
				MinorVersion: "4",
				Order:        19,
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v20.2.2",
			expectedStatusVersion: "v19.2.4",
		},
		{
			name:    "ceph image different from desired image, minor downgrade is allowed",
			cephDpl: &unitinputs.CephDeployMosk,
			osdpl:   faketestclients.GetClientBuilder().WithLists(unitinputs.GetOpenstackDeploymentStatusList("cur", "APPLIED", true)),
			lcmConfigData: map[string]string{
				"DEPLOYMENT_CEPH_IMAGE": "mirantis.azurecr.io/ceph/ceph:v20.2.0",
			},
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"configmaps":   &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments":  &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady, unitinputs.VersionCheckDeploymentReady("mirantis.azurecr.io/ceph/ceph:v20.2.0")}},
			},
			cmdOutputs: map[string]string{
				"ceph versions --format json": unitinputs.CephVersionsLatest,
				"ceph --version":              "ceph version 20.2.0 (commit) stable",
			},
			expectedVersion: &lcmcommon.CephVersion{
				Name:         "Tentacle",
				MajorVersion: "v20.2",
				MinorVersion: "2",
				Order:        20,
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v20.2.0",
			expectedStatusVersion: "v20.2.2",
		},
		{
			name:          "image versions aligned, remove check version deployment failed",
			cephDpl:       &unitinputs.CephDeployMosk,
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"deployments":  &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady, unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])}},
				"configmaps":   &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cmdOutputs: map[string]string{"ceph versions --format json": unitinputs.CephVersionsLatest},
			expectedVersion: &lcmcommon.CephVersion{
				Name:         "Tentacle",
				MajorVersion: "v20.2",
				MinorVersion: "2",
				Order:        20,
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v20.2.2",
			expectedStatusVersion: "v20.2.2",
			apiErrors:             map[string]error{"delete-deployments": errors.New("failed to delete deployment")},
		},
		{
			name:          "image versions aligned",
			cephDpl:       &unitinputs.CephDeployMosk,
			lcmConfigData: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"deployments":  &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady, unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])}},
				"configmaps":   &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
			},
			cmdOutputs: map[string]string{"ceph versions --format json": unitinputs.CephVersionsLatest},
			expectedVersion: &lcmcommon.CephVersion{
				Name:         "Tentacle",
				MajorVersion: "v20.2",
				MinorVersion: "2",
				Order:        20,
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v20.2.2",
			expectedStatusVersion: "v20.2.2",
		},
	}
	oldRunCmd := lcmcommon.RunPodCommandWithValidation
	oldInterval := versionCheckPollInterval
	oldTimeout := versionCheckPollTimeout
	versionCheckPollInterval = 1 * time.Second
	versionCheckPollTimeout = 2 * time.Second

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, test.lcmConfigData)
			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if v, ok := test.cmdOutputs[e.Command]; ok {
					return v, "", nil
				}
				return "", "", errors.New("unexpected run ceph command: " + e.Command)
			}

			if test.osdpl == nil {
				os.Unsetenv("CEPH_CONTROLLER_CLUSTER_RELEASE")
			} else {
				t.Setenv("CEPH_CONTROLLER_CLUSTER_RELEASE", "cur")
			}

			if test.osdpl != nil {
				c.api.Client = faketestclients.GetClient(test.osdpl)
			} else {
				c.api.Client = faketestclients.GetClient(nil)
			}

			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephclusters"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"deployments"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "delete", []string{"deployments"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"configmaps"}, test.inputResources, nil)

			cephVersion, cephImage, cephStatusVersion, err := c.verifyCephVersions()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedVersion, cephVersion)
			assert.Equal(t, test.expectedImage, cephImage)
			assert.Equal(t, test.expectedStatusVersion, cephStatusVersion)
			// clean reactions before next test
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldRunCmd
	versionCheckPollInterval = oldInterval
	versionCheckPollTimeout = oldTimeout
}

func TestPrepareVersionCheckDeployment(t *testing.T) {
	tests := []struct {
		name              string
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		expectedResources map[string]runtime.Object
		expectedError     string
	}{
		{
			name: "failed to get version-check deployment",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{},
			},
			apiErrors:     map[string]error{"get-deployments": errors.New("failed to get deployment")},
			expectedError: "failed to get 'lcm-namespace/pelagia-check-ceph-version' deployment: failed to get deployment",
		},
		{
			name: "failed to create version-check deployment",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{},
			},
			apiErrors:     map[string]error{"create-deployments": errors.New("failed to create deployment")},
			expectedError: "failed to create 'lcm-namespace/pelagia-check-ceph-version' deployment: failed to create deployment",
		},
		{
			name: "failed to wait version-check deployment readiness",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{},
			},
			expectedResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{unitinputs.VersionCheckDeployment(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])},
				},
			},
			expectedError: "timeout reached for waiting version-check deployment ready: context deadline exceeded",
		},
		{
			name: "version-check deployment update failed",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{unitinputs.VersionCheckDeploymentReady("old-image")},
				},
			},
			apiErrors:     map[string]error{"update-deployments": errors.New("failed to update deployment")},
			expectedError: "failed to update 'lcm-namespace/pelagia-check-ceph-version' deployment: failed to update deployment",
		},
		{
			name: "version-check deployment updated",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{unitinputs.VersionCheckDeploymentReady("old-image")},
				},
			},
			expectedResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])},
				},
			},
		},
		{
			name: "version-check deployment just updated and ready yet",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{
						func() appsv1.Deployment {
							dpl := unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])
							dpl.Generation = 1
							return dpl
						}(),
					},
				},
			},
			expectedError: "timeout reached for waiting version-check deployment ready: context deadline exceeded",
		},
		{
			name: "version-check deployment ready",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{unitinputs.VersionCheckDeploymentReady(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])},
				},
			},
		},
	}

	oldInterval := versionCheckPollInterval
	oldTimeout := versionCheckPollTimeout
	versionCheckPollInterval = 1 * time.Second
	versionCheckPollTimeout = 2 * time.Second
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"deployments"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "create", []string{"deployments"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "update", []string{"deployments"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			err := c.prepareVersionCheckDeployment(unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"])
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
		})
	}
	versionCheckPollInterval = oldInterval
	versionCheckPollTimeout = oldTimeout
}

func TestEnsureRookImage(t *testing.T) {
	tests := []struct {
		name              string
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		expectedResources map[string]runtime.Object
		expectedError     string
	}{
		{
			name: "ensure rook images - all rook images are consistent, success",
			inputResources: map[string]runtime.Object{
				"configmaps":  &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments": unitinputs.DeploymentList.DeepCopy(),
				"daemonsets":  &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
			},
		},
		{
			name: "ensure rook images - rook-ceph-operator not scaled, success",
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.RookDeploymentNotScaled.DeepCopy()}},
			},
		},
		{
			name: "ensure rook images - rook-ceph-operator image updated, progress",
			inputResources: map[string]runtime.Object{
				"configmaps":  &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments": &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.RookDeploymentPrevVersion.DeepCopy()}},
			},
			expectedResources: map[string]runtime.Object{
				"deployments": unitinputs.DeploymentList.DeepCopy(),
			},
			expectedError: "deployment rook-ceph/rook-ceph-operator rook image update is in progress",
		},
		{
			name: "ensure rook images - rook-discover image updated, progress",
			inputResources: map[string]runtime.Object{
				"configmaps":  &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments": unitinputs.DeploymentList.DeepCopy(),
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{
						func() appsv1.DaemonSet {
							ds := unitinputs.RookDiscover.DeepCopy()
							ds.Spec.Template.Spec.Containers[0].Image = "ceph/rook/v1.0.0-old-image"
							return *ds
						}(),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"deployments": unitinputs.DeploymentList.DeepCopy(),
				"daemonsets":  &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{unitinputs.RookDiscover}},
			},
			expectedError: "daemonset rook-ceph/rook-discover rook image update is in progress",
		},
		{
			name: "ensure rook images - rook-ceph-operator not ready, progress",
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{
						func() appsv1.Deployment {
							deploy := unitinputs.RookDeploymentLatestVersion.DeepCopy()
							deploy.Status.ReadyReplicas = 0
							return *deploy
						}(),
					},
				},
			},
			expectedError: "deployment rook-ceph/rook-ceph-operator rook image update still is in progress",
		},
		{
			name: "ensure rook images - rook-discover not ready, progress",
			inputResources: map[string]runtime.Object{
				"configmaps":  &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments": unitinputs.DeploymentList.DeepCopy(),
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{
						func() appsv1.DaemonSet {
							ds := unitinputs.RookDiscover.DeepCopy()
							ds.Status.NumberReady = 0
							return *ds
						}(),
					},
				},
			},
			expectedError: "daemonset rook-ceph/rook-discover rook image update still is in progress",
		},
		{
			name: "ensure rook images - rook-discover not used, external case",
			inputResources: map[string]runtime.Object{
				"configmaps":  &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments": unitinputs.DeploymentList.DeepCopy(),
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{
						func() appsv1.DaemonSet {
							ds := unitinputs.RookDiscover.DeepCopy()
							ds.Status.NumberReady = 0
							ds.Status.DesiredNumberScheduled = 0
							return *ds
						}(),
					},
				},
			},
		},
		{
			name: "ensure rook images - rook-ceph-operator get failed, failed",
			inputResources: map[string]runtime.Object{
				"configmaps":  &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments": unitinputs.DeploymentList.DeepCopy(),
			},
			apiErrors:     map[string]error{"get-deployments": errors.New("failed to get deployment")},
			expectedError: "failed to get rook-ceph/rook-ceph-operator deployment: failed to get deployment",
		},
		{
			name: "ensure rook images - consistent, but ceph is not deployed",
			inputResources: map[string]runtime.Object{
				"configmaps":  unitinputs.ConfigMapListEmpty,
				"deployments": unitinputs.DeploymentList.DeepCopy(),
			},
		},
		{
			name: "ensure rook images - rook-discover get failed, failed",
			inputResources: map[string]runtime.Object{
				"configmaps":  &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments": unitinputs.DeploymentList.DeepCopy(),
				"daemonsets":  &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
			},
			apiErrors:     map[string]error{"get-daemonsets": errors.New("failed to get daemonset")},
			expectedError: "failed to get rook-ceph/rook-discover daemonset: failed to get daemonset",
		},
		{
			name: "ensure rook images - rook-ceph-operator image update error, failed",
			inputResources: map[string]runtime.Object{
				"configmaps":  &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments": &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.RookDeploymentPrevVersion.DeepCopy()}},
			},
			apiErrors:     map[string]error{"update-deployments": errors.New("failed to update deployment")},
			expectedError: "failed to update rook-ceph/rook-ceph-operator deployment with new rook image: failed to update deployment",
		},
		{
			name: "ensure rook images - rook-discover image update error, progress",
			inputResources: map[string]runtime.Object{
				"configmaps":  &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments": unitinputs.DeploymentList.DeepCopy(),
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{
						func() appsv1.DaemonSet {
							ds := unitinputs.RookDiscover.DeepCopy()
							ds.Spec.Template.Spec.Containers[0].Image = "ceph/rook/v1.0.0-old-image"
							return *ds
						}(),
					},
				},
			},
			apiErrors:     map[string]error{"update-daemonsets": errors.New("failed to update daemonsets")},
			expectedError: "failed to update rook-ceph/rook-discover daemonset with new rook image: failed to update daemonsets",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			c.lcmConfig.DeployParams.RookImage = unitinputs.PelagiaConfig.Data["DEPLOYMENT_ROOK_IMAGE"]
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"daemonsets", "deployments"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "update", []string{"daemonsets", "deployments"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"configmaps"}, test.inputResources, nil)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			err := c.ensureRookImage()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
		})
	}
}

func TestEnsureCephClusterVersion(t *testing.T) {
	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		imageToUse        string
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		expectedResources map[string]runtime.Object
		expectedError     string
	}{
		{
			name:    "no ceph cluster - skip",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListEmpty,
			},
		},
		{
			name:    "get ceph cluster failed - fail",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListEmpty,
			},
			apiErrors:     map[string]error{"get-cephclusters": errors.New("failed to get cephcluster")},
			expectedError: "failed to get rook-ceph/cephcluster CephCluster: failed to get cephcluster",
		},
		{
			name:    "ceph cluster image equals to actual one - skip",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterGenerated.DeepCopy()}},
			},
			imageToUse: unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"],
		},
		{
			name:    "ceph cluster image not equals to actual one - update has failed",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterGenerated.DeepCopy()}},
			},
			imageToUse:    "fake/fake:v2.3.3",
			apiErrors:     map[string]error{"update-cephclusters": errors.New("failed to update cephcluster")},
			expectedError: "failed to update CephCluster rook-ceph/cephcluster version: failed to update cephcluster",
		},
		{
			name:    "ceph cluster image not equals to actual one - update in progress",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterGenerated.DeepCopy()}},
			},
			imageToUse: "fake/fake:v2.3.3",
			expectedResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cc := unitinputs.CephClusterGenerated.DeepCopy()
						cc.Spec.CephVersion.Image = "fake/fake:v2.3.3"
						return *cc
					}(),
				}},
			},
			expectedError: "update CephCluster rook-ceph/cephcluster version is in progress",
		},
		{
			name:    "ceph cluster image not equals to actual one - update in progress and drop osd restart reason",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cl := unitinputs.CephClusterGenerated.DeepCopy()
						cl.Spec.CephVersion.Image = "fake/fake:v2.3.3"
						cl.Annotations = map[string]string{
							"cephdeployment.lcm.mirantis.com/restart-osd-reason":    "cephcluster unit test",
							"cephdeployment.lcm.mirantis.com/restart-osd-requested": "time-9",
						}
						cl.Spec.Annotations[cephv1.KeyOSD] = map[string]string{
							"cephdeployment.lcm.mirantis.com/restart-osd-reason":    "cephcluster unit test",
							"cephdeployment.lcm.mirantis.com/restart-osd-requested": "time-9",
						}
						return *cl
					}(),
				}},
			},
			imageToUse: unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"],
			expectedResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cl := unitinputs.CephClusterGenerated.DeepCopy()
						cl.Annotations = map[string]string{}
						return *cl
					}(),
				}},
			},
			expectedError: "update CephCluster rook-ceph/cephcluster version is in progress",
		},
		{
			name: "ceph cluster image not equals to actual one - update in progress, but keep osd restart reason",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.BaseCephDeployment.DeepCopy()
				mc.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{
					OsdRestartReason: "cephcluster unit test",
				}
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cl := unitinputs.CephClusterGenerated.DeepCopy()
						cl.Spec.CephVersion.Image = "fake/fake:v2.3.3"
						cl.Annotations = map[string]string{
							"cephdeployment.lcm.mirantis.com/restart-osd-reason":    "cephcluster unit test",
							"cephdeployment.lcm.mirantis.com/restart-osd-requested": "time-9",
						}
						cl.Spec.Annotations[cephv1.KeyOSD] = map[string]string{
							"cephdeployment.lcm.mirantis.com/restart-osd-reason":    "cephcluster unit test",
							"cephdeployment.lcm.mirantis.com/restart-osd-requested": "time-9",
						}
						return *cl
					}(),
				}},
			},
			imageToUse: unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"],
			expectedResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cl := unitinputs.CephClusterGenerated.DeepCopy()
						cl.Annotations = map[string]string{
							"cephdeployment.lcm.mirantis.com/restart-osd-reason":    "cephcluster unit test",
							"cephdeployment.lcm.mirantis.com/restart-osd-requested": "time-9",
						}
						cl.Spec.Annotations[cephv1.KeyOSD] = map[string]string{
							"cephdeployment.lcm.mirantis.com/restart-osd-reason":    "cephcluster unit test",
							"cephdeployment.lcm.mirantis.com/restart-osd-requested": "time-9",
						}
						return *cl
					}(),
				}},
			},
			expectedError: "update CephCluster rook-ceph/cephcluster version is in progress",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			c.cdConfig.currentCephVersion = lcmcommon.LatestRelease
			c.cdConfig.currentCephImage = test.imageToUse
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephclusters"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", []string{"cephclusters"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			err := c.ensureCephClusterVersion()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}
