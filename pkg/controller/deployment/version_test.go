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
		ccsettingsMap         map[string]string
		cmdOutput             string
		osdpl                 *fakeclient.ClientBuilder
		expectedVersion       *lcmcommon.CephVersion
		expectedImage         string
		expectedStatusVersion string
		expectedError         string
	}{
		{
			name:    "failed to find supported ceph version for specified image",
			cephDpl: unitinputs.CephDeployMosk.DeepCopy(),
			ccsettingsMap: func() map[string]string {
				cm := unitinputs.PelagiaConfig.DeepCopy().Data
				cm["DEPLOYMENT_CEPH_RELEASE"] = "blabla"
				return cm
			}(),
			expectedError: "failed to check desired ceph version: failed to find appropriate Ceph version of 'blabla' release. Is release name correct?",
		},
		{
			name: "cephdeployment cluster version is aligned with specified image and release",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Status.ClusterVersion = unitinputs.LatestCephVersionImage
				return mc
			}(),
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			expectedVersion: &lcmcommon.CephVersion{
				Name:            "Tentacle",
				MajorVersion:    "v20.2",
				MinorVersion:    "0",
				Order:           20,
				SupportedMinors: []string{"0"},
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v20.2.0",
			expectedStatusVersion: "v20.2.0",
		},
		{
			name:          "cephdeployment cluster version is not set and failed to get cephcluster",
			cephDpl:       &unitinputs.CephDeployMosk,
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			expectedError: "failed to get rook-ceph/cephcluster cephcluster: failed to get resource(s) kind of 'cephclusters': list object is not specified in test",
		},
		{
			name:          "cephdeployment cluster version is not set and cephcluster not found",
			cephDpl:       &unitinputs.CephDeployMosk,
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{}},
			},
			expectedVersion: &lcmcommon.CephVersion{
				Name:            "Tentacle",
				MajorVersion:    "v20.2",
				MinorVersion:    "0",
				Order:           20,
				SupportedMinors: []string{"0"},
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v20.2.0",
			expectedStatusVersion: "",
		},
		{
			name:          "cephdeployment cluster version is not set and cephcluster not deployed",
			cephDpl:       &unitinputs.CephDeployMosk,
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"configmaps":   &corev1.ConfigMapList{},
			},
			expectedVersion: &lcmcommon.CephVersion{
				Name:            "Tentacle",
				MajorVersion:    "v20.2",
				MinorVersion:    "0",
				Order:           20,
				SupportedMinors: []string{"0"},
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v20.2.0",
			expectedStatusVersion: "",
		},
		{
			name:          "cephdeployment cluster version is not set and cephcluster not deployed and different image set",
			cephDpl:       &unitinputs.CephDeployMosk,
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cluster := unitinputs.CephClusterReady.DeepCopy()
						cluster.Spec.CephVersion.Image = "some-registry.com/ceph:v19.2.3"
						return *cluster
					}(),
				}},
				"configmaps": &corev1.ConfigMapList{},
			},
			expectedVersion: &lcmcommon.CephVersion{
				Name:            "Squid",
				MajorVersion:    "v19.2",
				MinorVersion:    "3",
				Order:           19,
				SupportedMinors: []string{"3"},
			},
			expectedImage:         "some-registry.com/ceph:v19.2.3",
			expectedStatusVersion: "",
		},
		{
			name:          "cephdeployment cluster version is not set and cephcluster not deployed and incorrect image",
			cephDpl:       &unitinputs.CephDeployMosk,
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cl := unitinputs.CephClusterGenerated.DeepCopy()
						cl.Spec.CephVersion.Image = "mirantis.azurecr.io/ceph/ceph:v2.3.4"
						return *cl
					}(),
				}},
				"configmaps": &corev1.ConfigMapList{},
			},
			expectedError: "failed to verify Ceph version in CephCluster spec: failed to find supported Ceph version for specified 'v2.3.4' version. Is version correct?",
		},
		{
			name:          "cephdeployment cluster version is not set, cephcluster deployed, ceph tools is not available",
			cephDpl:       &unitinputs.CephDeployMosk,
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"configmaps":   &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments":  &appsv1.DeploymentList{},
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v20.2.0",
			expectedStatusVersion: "",
		},
		{
			name:          "cephdeployment cluster version is not set, cephcluster deployed, ceph tools available, failed to get versions",
			cephDpl:       &unitinputs.CephDeployMosk,
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"configmaps":   &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments":  &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady}},
			},
			cmdOutput:     "{||}",
			expectedError: "failed to get current Ceph versions: failed to parse output for command 'ceph versions --format json': invalid character '|' looking for beginning of object key string",
		},
		{
			name:          "cephdeployment cluster version is not set, cephcluster deployed, ceph tools available, in mid upgrade",
			cephDpl:       &unitinputs.CephDeployMosk,
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"configmaps":   &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.RookCephMonEndpoints}},
				"deployments":  &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady}},
			},
			cmdOutput: `
{
  "overall": {
    "ceph version 19.2.3 (c44bc49e7a57a87d84dfff2a077a2058aa2172e2) squid (stable)": 12,
    "ceph version 20.2.0 (c44bc49e7a57a87d84dfff2a077a2058aa2172e2) tentacle (stable)": 2
  }
}
			`,
			expectedVersion: &lcmcommon.CephVersion{
				Name:            "Squid",
				MajorVersion:    "v19.2",
				MinorVersion:    "3",
				Order:           19,
				SupportedMinors: []string{"3"},
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v20.2.0",
			expectedStatusVersion: "v19.2.3,v20.2.0",
		},
		{
			name: "cephdeployment cluster version is set, but cephCluster not found",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Status.ClusterVersion = "v19.2.3"
				return mc
			}(),
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{}},
			},
			expectedError: "failed to get rook-ceph/cephcluster CephCluster: cephclusters \"cephcluster\" not found",
		},
		{
			name: "cephdeployment cluster version is wrong",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Status.ClusterVersion = "v19.2.9"
				return mc
			}(),
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
			},
			expectedError: "failed to verify current Ceph version: specified Ceph version 'v19.2.9' is not supported. Please use one of: [v19.2.3]",
		},
		{
			name: "cephdeployment cluster version is set, but new release is major downgrade",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Status.ClusterVersion = unitinputs.LatestCephVersionImage
				return mc
			}(),
			ccsettingsMap: func() map[string]string {
				cm := unitinputs.PelagiaConfigForPrevCephVersion.DeepCopy().Data
				cm["DEPLOYMENT_CEPH_IMAGE"] = "mirantis.azurecr.io/ceph/ceph:v19.2.3"
				cm["DEPLOYMENT_CEPH_RELEASE"] = "squid"
				return cm
			}(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
			},
			expectedError: "detected Ceph version downgrade from 'v20.2.0' to 'v19.2.3': downgrade is not possible",
		},
		/* TODO: uncomment if more than 2 releases are supported at the time
		{
			name: "cephdeployment cluster version is set, but new release is upgrade step over one",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Status.ClusterVersion = "v17.2.7"
				return mc
			}(),
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
			},
			expectedError: "detected Ceph version upgrade from 'v17.2.7' to 'v20.2.0': upgrade with step over one major version is not possible",
		},*/
		{
			name: "cephdeployment cluster version is set, ceph image different from desired image, failed to check upgrade allowed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Status.ClusterVersion = "v19.2.3"
				return mc
			}(),
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						c := unitinputs.TestCephCluster.DeepCopy()
						c.Spec.CephVersion.Image = unitinputs.PelagiaConfigForPrevCephVersion.Data["DEPLOYMENT_CEPH_IMAGE"]
						return *c
					}(),
				}},
			},
			expectedError: "failed to check is Ceph upgrade allowed: required env variable 'CEPH_CONTROLLER_CLUSTER_RELEASE' is not set",
		},
		{
			name: "cephdeployment cluster version is set, ceph image different from desired image, upgrade is not allowed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Status.ClusterVersion = "v19.2.3"
				return mc
			}(),
			osdpl:         faketestclients.GetClientBuilder().WithLists(unitinputs.GetOpenstackDeploymentStatusList("new", "APPLIED", true)),
			ccsettingsMap: unitinputs.PelagiaConfigForPrevCephVersion.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						c := unitinputs.TestCephCluster.DeepCopy()
						c.Spec.CephVersion.Image = unitinputs.PelagiaConfigForPrevCephVersion.Data["DEPLOYMENT_CEPH_IMAGE"]
						return *c
					}(),
				}},
			},
			expectedVersion: &lcmcommon.CephVersion{
				Name:            "Squid",
				MajorVersion:    "v19.2",
				MinorVersion:    "3",
				Order:           19,
				SupportedMinors: []string{"3"},
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v19.2.3",
			expectedStatusVersion: "v19.2.3",
		},
		{
			name: "cephdeployment cluster version is set, ceph image different from desired image, upgrade is allowed",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Status.ClusterVersion = "v19.2.3"
				return mc
			}(),
			osdpl:         faketestclients.GetClientBuilder().WithLists(unitinputs.GetOpenstackDeploymentStatusList("cur", "APPLIED", true)),
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						c := unitinputs.TestCephCluster.DeepCopy()
						c.Spec.CephVersion.Image = unitinputs.PelagiaConfigForPrevCephVersion.Data["DEPLOYMENT_CEPH_IMAGE"]
						return *c
					}(),
				}},
			},
			expectedVersion: &lcmcommon.CephVersion{
				Name:            "Squid",
				MajorVersion:    "v19.2",
				MinorVersion:    "3",
				Order:           19,
				SupportedMinors: []string{"3"},
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v20.2.0",
			expectedStatusVersion: "v19.2.3",
		},
		{
			name: "cephdeployment status version is not updated, ceph cluster is up to date, failed to check versions",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Status.ClusterVersion = "v19.2.3"
				return mc
			}(),
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"deployments":  &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady}},
			},
			cmdOutput:     "{||}",
			expectedError: "failed to get current Ceph versions: failed to parse output for command 'ceph versions --format json': invalid character '|' looking for beginning of object key string",
		},
		{
			name: "cephdeployment status version is not updated, ceph cluster is up to date, check versions",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Status.ClusterVersion = "v19.2.3"
				return mc
			}(),
			ccsettingsMap: unitinputs.PelagiaConfig.DeepCopy().Data,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"deployments":  &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.ToolBoxDeploymentReady}},
			},
			cmdOutput: unitinputs.CephVersionsLatestWithExtraDaemons,
			expectedVersion: &lcmcommon.CephVersion{
				Name:            "Tentacle",
				MajorVersion:    "v20.2",
				MinorVersion:    "0",
				Order:           20,
				SupportedMinors: []string{"0"},
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v20.2.0",
			expectedStatusVersion: "v20.2.0",
		},
		{
			name: "cephdeployment cluster prev major version is set and up to date, ceph tools is not ready",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Status.ClusterVersion = "v19.2.3"
				return mc
			}(),
			ccsettingsMap: func() map[string]string {
				cm := unitinputs.PelagiaConfigForPrevCephVersion.DeepCopy().Data
				cm["DEPLOYMENT_CEPH_IMAGE"] = "mirantis.azurecr.io/ceph/ceph:v19.2.3"
				cm["DEPLOYMENT_CEPH_RELEASE"] = "squid"
				return cm
			}(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
				"deployments":  &appsv1.DeploymentList{},
			},
			expectedVersion: &lcmcommon.CephVersion{
				Name:            "Squid",
				MajorVersion:    "v19.2",
				MinorVersion:    "3",
				Order:           19,
				SupportedMinors: []string{"3"},
			},
			expectedImage:         "mirantis.azurecr.io/ceph/ceph:v19.2.3",
			expectedStatusVersion: "v19.2.3",
		},
	}
	oldRunCmd := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, test.ccsettingsMap)
			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "ceph versions --format json" {
					return test.cmdOutput, "", nil
				}
				return "", "", errors.New("unexpected run ceph command call")
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

			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephclusters"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"deployments"}, test.inputResources, nil)
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
