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
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	fakecephv1 "github.com/rook/rook/pkg/client/clientset/versioned/typed/ceph.rook.io/v1/fake"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	gotesting "k8s.io/client-go/testing"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestGenerateCephCluster(t *testing.T) {
	resourceUpdateTimestamps = updateTimestamps{
		cephConfigMap: map[string]string{
			"global": "some-time",
			"mon":    "some-time",
			"mgr":    "some-time",
		},
	}
	tests := []struct {
		name                string
		cephDpl             *cephlcmv1alpha1.CephDeployment
		expectedClusterSpec cephv1.ClusterSpec
	}{
		{
			name:                "generate base ceph cluster",
			cephDpl:             &unitinputs.BaseCephDeployment,
			expectedClusterSpec: unitinputs.CephClusterGenerated.Spec,
		},
		{
			name: "generate ceph cluster with resource requirements and tolerations",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.BaseCephDeployment.DeepCopy()
				cd.Spec.HyperConverge = unitinputs.HyperConvergeCephDeploy
				return cd
			}(),
			expectedClusterSpec: func() cephv1.ClusterSpec {
				newSpec := *unitinputs.CephClusterGenerated.Spec.DeepCopy()
				newSpec.Resources = unitinputs.HyperConvergeCephDeploy.Resources
				monPl := newSpec.Placement[cephv1.KeyMon]
				monPl.Tolerations = append(monPl.Tolerations, unitinputs.HyperConvergeCephDeploy.Tolerations["mon"].Rules...)
				newSpec.Placement[cephv1.KeyMon] = monPl
				mgrPl := newSpec.Placement[cephv1.KeyMgr]
				mgrPl.Tolerations = append(mgrPl.Tolerations, unitinputs.HyperConvergeCephDeploy.Tolerations["mgr"].Rules...)
				newSpec.Placement[cephv1.KeyMgr] = mgrPl
				newSpec.Placement[cephv1.KeyAll] = cephv1.Placement{
					Tolerations: unitinputs.HyperConvergeCephDeploy.Tolerations["all"].Rules,
				}
				newSpec.Placement[cephv1.KeyOSD] = cephv1.Placement{
					Tolerations: unitinputs.HyperConvergeCephDeploy.Tolerations["osd"].Rules,
				}
				newSpec.Placement[cephv1.KeyCleanup] = cephv1.Placement{
					Tolerations: unitinputs.HyperConvergeCephDeploy.Tolerations["osd"].Rules,
				}
				newSpec.Placement[cephv1.KeyOSDPrepare] = cephv1.Placement{
					Tolerations: unitinputs.HyperConvergeCephDeploy.Tolerations["osd"].Rules,
				}
				return newSpec
			}(),
		},
		{
			name: "generate ceph cluster with node resource",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				resource := unitinputs.BaseCephDeployment.DeepCopy()
				resource.Spec.Nodes[0].Resources = v1.ResourceRequirements{
					Limits:   unitinputs.ResourceListLimitsDefault,
					Requests: unitinputs.ResourceListRequestsDefault,
				}
				return resource
			}(),
			expectedClusterSpec: func() cephv1.ClusterSpec {
				newSpec := *unitinputs.CephClusterGenerated.Spec.DeepCopy()
				newSpec.Storage.Nodes[0].Resources = v1.ResourceRequirements{
					Limits:   unitinputs.ResourceListLimitsDefault,
					Requests: unitinputs.ResourceListRequestsDefault,
				}
				return newSpec
			}(),
		},
		{
			name: "generate ceph cluster with mgr modules and differenet data host path",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				resource := unitinputs.BaseCephDeployment.DeepCopy()
				resource.Spec.Mgr = &cephlcmv1alpha1.Mgr{
					MgrModules: []cephlcmv1alpha1.CephMgrModule{
						{
							Name:    "balancer",
							Enabled: true,
							Settings: &cephlcmv1alpha1.CephMgrModuleSettings{
								BalancerMode: "upmap",
							},
						},
						{
							Name:    "fake",
							Enabled: true,
						},
					},
				}
				resource.Spec.DataDirHostPath = "/var/lib/fake"
				return resource
			}(),
			expectedClusterSpec: func() cephv1.ClusterSpec {
				newSpec := *unitinputs.CephClusterGenerated.Spec.DeepCopy()
				newSpec.DataDirHostPath = "/var/lib/fake"
				newSpec.Mgr.Modules = []cephv1.Module{
					{
						Name:    "balancer",
						Enabled: true,
						Settings: cephv1.ModuleSettings{
							BalancerMode: "upmap",
						},
					},
					{
						Name:    "fake",
						Enabled: true,
					},
					{
						Name:    "pg_autoscaler",
						Enabled: true,
					},
				}
				return newSpec
			}(),
		},
		{
			name: "generate ceph cluster with healthCheck",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				resource := unitinputs.BaseCephDeployment.DeepCopy()
				duration, _ := time.ParseDuration("60s")
				resource.Spec.SharedFilesystem = &cephlcmv1alpha1.CephSharedFilesystem{}
				resource.Spec.HealthCheck = &cephlcmv1alpha1.CephClusterHealthCheckSpec{
					DaemonHealth: cephv1.DaemonHealthSpec{
						Status: cephv1.HealthCheckSpec{
							Disabled: true,
						},
						ObjectStorageDaemon: cephv1.HealthCheckSpec{
							Timeout:  "60s",
							Interval: &metav1.Duration{Duration: duration},
						},
						Monitor: cephv1.HealthCheckSpec{
							Timeout: "60s",
						},
					},
					LivenessProbe: map[cephv1.KeyType]*cephv1.ProbeSpec{
						"osd": {
							Probe: &v1.Probe{
								TimeoutSeconds:   10,
								FailureThreshold: 10,
							},
						},
						"mgr": {
							Probe: &v1.Probe{
								TimeoutSeconds:   7,
								FailureThreshold: 7,
							},
						},
					},
					StartupProbe: map[cephv1.KeyType]*cephv1.ProbeSpec{
						"osd": {
							Probe: &v1.Probe{
								TimeoutSeconds:   5,
								FailureThreshold: 5,
							},
						},
						"mon": {
							Probe: &v1.Probe{
								TimeoutSeconds:   5,
								FailureThreshold: 5,
							},
						},
					},
				}
				return resource
			}(),
			expectedClusterSpec: func() cephv1.ClusterSpec {
				resource := *unitinputs.CephClusterGenerated.Spec.DeepCopy()
				duration, _ := time.ParseDuration("60s")
				resource.HealthCheck = cephv1.CephClusterHealthCheckSpec{
					DaemonHealth: cephv1.DaemonHealthSpec{
						Status: cephv1.HealthCheckSpec{
							Disabled: true,
						},
						Monitor: cephv1.HealthCheckSpec{
							Timeout: "60s",
						},
						ObjectStorageDaemon: cephv1.HealthCheckSpec{
							Timeout:  "60s",
							Interval: &metav1.Duration{Duration: duration},
						},
					},
					LivenessProbe: map[cephv1.KeyType]*cephv1.ProbeSpec{
						"osd": {
							Probe: &v1.Probe{
								TimeoutSeconds:   10,
								FailureThreshold: 10,
							},
						},
						"mgr": {
							Probe: &v1.Probe{
								TimeoutSeconds:   7,
								FailureThreshold: 7,
							},
						},
						"mon": {
							Probe: &v1.Probe{
								TimeoutSeconds:   5,
								FailureThreshold: 5,
							},
						},
					},
					StartupProbe: map[cephv1.KeyType]*cephv1.ProbeSpec{
						"osd": {
							Probe: &v1.Probe{
								TimeoutSeconds:   5,
								FailureThreshold: 5,
							},
						},
						"mon": {
							Probe: &v1.Probe{
								TimeoutSeconds:   5,
								FailureThreshold: 5,
							},
						},
					},
				}
				return resource
			}(),
		},
		{
			name:                "generate cluster external",
			cephDpl:             unitinputs.CephDeployExternal.DeepCopy(),
			expectedClusterSpec: unitinputs.CephClusterExternal.Spec,
		},
		{
			name: "generate ceph cluster with by-id in fullPath and device filters",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				resource := unitinputs.BaseCephDeployment.DeepCopy()
				resource.Spec.Nodes = append(resource.Spec.Nodes,
					[]cephlcmv1alpha1.CephDeploymentNode{
						{
							Node: cephv1.Node{
								Name: "node-4",
								Selection: cephv1.Selection{
									Devices: []cephv1.Device{
										{
											FullPath: "/dev/disk/by-id/sata-230:3000:004:01",
											Config:   map[string]string{"deviceClass": "hdd"},
										},
									},
								},
							},
						},
						{
							Node: cephv1.Node{
								Name:   "node-5",
								Config: map[string]string{"deviceClass": "hdd"},
								Selection: cephv1.Selection{
									DeviceFilter: "sd[fg]",
								},
							},
						},
						{
							Node: cephv1.Node{
								Name:   "node-6",
								Config: map[string]string{"deviceClass": "hdd"},
								Selection: cephv1.Selection{
									DevicePathFilter: "/dev/disk/by-id/sata-230:3000:004:0[15]",
								},
							},
						},
					}...)
				return resource
			}(),
			expectedClusterSpec: func() cephv1.ClusterSpec {
				cephSpec := unitinputs.CephClusterGenerated.DeepCopy().Spec
				cephSpec.Storage.Nodes = append(cephSpec.Storage.Nodes, []cephv1.Node{
					{
						Name: "node-4",
						Selection: cephv1.Selection{
							UseAllDevices: nil,
							DeviceFilter:  "",
							Devices: []cephv1.Device{
								{
									Name:   "/dev/disk/by-id/sata-230:3000:004:01",
									Config: map[string]string{"deviceClass": "hdd"},
								},
							},
							DevicePathFilter:     "",
							VolumeClaimTemplates: nil,
						},
						Config: nil,
					},
					{
						Name: "node-5",
						Selection: cephv1.Selection{
							DeviceFilter:         "sd[fg]",
							DevicePathFilter:     "",
							VolumeClaimTemplates: nil,
						},
						Config: map[string]string{"deviceClass": "hdd"},
					},
					{
						Name: "node-6",
						Selection: cephv1.Selection{
							DeviceFilter:         "",
							DevicePathFilter:     "/dev/disk/by-id/sata-230:3000:004:0[15]",
							VolumeClaimTemplates: nil,
						},
						Config: map[string]string{"deviceClass": "hdd"},
					},
				}...)
				// because the number of the nodes > 3
				cephSpec.ContinueUpgradeAfterChecksEvenIfNotHealthy = false
				cephSpec.SkipUpgradeChecks = false
				return cephSpec
			}(),
		},
		{
			name: "generate ceph cluster with addressRanges",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.BaseCephDeployment.DeepCopy()
				mc.Spec.Network.MonOnPublicNet = true
				return mc
			}(),
			expectedClusterSpec: func() cephv1.ClusterSpec {
				cs := unitinputs.CephClusterGenerated.Spec.DeepCopy()
				cs.Network.AddressRanges = &cephv1.AddressRangesSpec{
					Cluster: cephv1.CIDRList{
						"127.0.0.0/16",
					},
					Public: cephv1.CIDRList{
						"192.168.0.0/16",
					},
				}
				return *cs
			}(),
		},
		{
			name:    "generate ceph cluster with multus provider",
			cephDpl: &unitinputs.BaseCephDeploymentMultus,
			expectedClusterSpec: func() cephv1.ClusterSpec {
				cs := unitinputs.CephClusterGenerated.Spec.DeepCopy()
				cs.Network.Provider = "multus"
				cs.Network.Selectors = unitinputs.BaseCephDeploymentMultus.Spec.Network.Selector
				return *cs
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			expandedNodes, err := c.buildExpandedNodeList()
			assert.Nil(t, err)
			cephClusterSpec := generateCephClusterSpec(test.cephDpl, unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"], expandedNodes)
			assert.Equal(t, test.expectedClusterSpec, cephClusterSpec)
		})
	}
	unsetTimestampsVar()
}

func TestEnsureCluster(t *testing.T) {
	getCluster := func(annotations map[cephv1.KeyType]cephv1.Annotations) cephv1.CephCluster {
		cc := unitinputs.CephClusterGenerated.DeepCopy()
		cc.Spec.Annotations = annotations
		return *cc
	}
	getClusterEnsure := getCluster(map[cephv1.KeyType]cephv1.Annotations{
		cephv1.KeyMon: map[string]string{
			"cephdeployment.lcm.mirantis.com/config-global-updated": "time-6",
			"cephdeployment.lcm.mirantis.com/config-mon-updated":    "time-6",
		},
		cephv1.KeyMgr: {
			"cephdeployment.lcm.mirantis.com/config-global-updated": "time-6",
		},
	})

	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		expectedResources map[string]runtime.Object
		updated           bool
		expectedError     string
	}{
		{
			name:    "cluster get failed",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListEmpty,
			},
			apiErrors:     map[string]error{"get-cephclusters": errors.New("cephcluster get failed")},
			expectedError: "failed to get rook-ceph/cephcluster cephcluster: cephcluster get failed",
		},
		{
			name:    "failed to build ceph config",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterExternal.DeepCopy()}},
				"configmaps":   unitinputs.ConfigMapListEmpty.DeepCopy(),
			},
			apiErrors:     map[string]error{"get-configmaps": errors.New("failed to get config map")},
			expectedError: "failed to ensure ceph config for rook-ceph/cephcluster cephcluster: failed to get config map",
		},
		{
			name:    "cluster status check is not ready, configmap is updated, cephcluster is not updated",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{unitinputs.ReefCephClusterNotReady}},
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{*unitinputs.BaseRookConfigOverride.DeepCopy()}},
			},
			expectedError: "failed to ensure cephcluster rook-ceph/cephcluster: ceph cluster rook-ceph/cephcluster is not ready to be updated: cluster healthy = true, cluster state = 'Created', cluster phase = 'Progressing'",
		},
		{
			name:    "cephcluster create failed",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephclusters": unitinputs.CephClusterListEmpty.DeepCopy(),
				"configmaps":   unitinputs.ConfigMapListEmpty.DeepCopy(),
			},
			apiErrors: map[string]error{"create-cephclusters": errors.New("failed to create cluster")},
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						cm := unitinputs.BaseRookConfigOverride.DeepCopy()
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-generated"] = "time-3"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-mon-updated"] = "time-3"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-global-updated"] = "time-3"
						return *cm
					}(),
				}},
			},
			expectedError: "failed to create cephcluster rook-ceph/cephcluster: failed to create cluster",
		},
		{
			name:    "cluster created, image from ccsettings",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephclusters": unitinputs.CephClusterListEmpty.DeepCopy(),
				"configmaps":   unitinputs.ConfigMapListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{
					Items: []cephv1.CephCluster{
						getCluster(map[cephv1.KeyType]cephv1.Annotations{
							cephv1.KeyMon: map[string]string{
								"cephdeployment.lcm.mirantis.com/config-global-updated": "time-4",
								"cephdeployment.lcm.mirantis.com/config-mon-updated":    "time-4",
							},
							cephv1.KeyMgr: map[string]string{
								"cephdeployment.lcm.mirantis.com/config-global-updated": "time-4",
							},
						}),
					},
				},
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						cm := unitinputs.BaseRookConfigOverride.DeepCopy()
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-generated"] = "time-4"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-mon-updated"] = "time-4"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-global-updated"] = "time-4"
						return *cm
					}(),
				}},
			},
			updated: true,
		},
		{
			name:    "update cluster failed",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterGenerated.DeepCopy()}},
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{*unitinputs.RookCephMonEndpoints.DeepCopy()}},
			},
			apiErrors: map[string]error{"update-cephclusters": errors.New("failed to update cluster")},
			expectedResources: map[string]runtime.Object{
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					unitinputs.RookCephMonEndpoints,
					func() v1.ConfigMap {
						cm := unitinputs.BaseRookConfigOverride.DeepCopy()
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-generated"] = "time-5"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-mon-updated"] = "time-5"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-global-updated"] = "time-5"
						return *cm
					}(),
				}},
			},
			expectedError: "failed to update cephcluster rook-ceph/cephcluster: failed to update cluster",
		},
		{
			name:    "update cluster - restart required for services, annotations already set",
			cephDpl: &unitinputs.CephDeployRookConfigNoRuntimeNoOsd,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{
					Items: []cephv1.CephCluster{*getClusterEnsure.DeepCopy()},
				},
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						cm := unitinputs.BaseRookConfigOverride.DeepCopy()
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-mon-updated"] = "time-6"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-global-updated"] = "time-6"
						return *cm
					}(),
					unitinputs.RookCephMonEndpoints,
				}},
			},
			updated: true,
		},
		{
			name:    "update cluster - no restart required for services, annotations already set",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{
					Items: []cephv1.CephCluster{*getClusterEnsure.DeepCopy()},
				},
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						cm := unitinputs.BaseRookConfigOverride.DeepCopy()
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-generated"] = "time-6"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-mon-updated"] = "time-6"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-global-updated"] = "time-6"
						return *cm
					}(),
					unitinputs.RookCephMonEndpoints,
				}},
			},
		},
		{
			name:    "update cluster - ceph cluster not deployed, runtime params not set",
			cephDpl: &unitinputs.BaseCephDeployment,
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{
					Items: []cephv1.CephCluster{*unitinputs.CephClusterGenerated.DeepCopy()},
				},
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						cm := unitinputs.BaseRookConfigOverride.DeepCopy()
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-generated"] = "time-6"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-mon-updated"] = "time-6"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-global-updated"] = "time-6"
						return *cm
					}(),
				}},
			},
			expectedResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{
					Items: []cephv1.CephCluster{*getClusterEnsure.DeepCopy()},
				},
			},
			updated: true,
		},
		{
			name: "update cluster - set osd restart reason",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.BaseCephDeployment.DeepCopy()
				mc.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{
					OsdRestartReason: "cephcluster unit test",
				}
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{
					Items: []cephv1.CephCluster{*unitinputs.CephClusterGenerated.DeepCopy()},
				},
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						cm := unitinputs.BaseRookConfigOverride.DeepCopy()
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-generated"] = "time-6"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-mon-updated"] = "time-6"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-global-updated"] = "time-6"
						return *cm
					}(),
				}},
			},
			expectedResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cl := *getClusterEnsure.DeepCopy()
						cl.Annotations = map[string]string{
							"cephdeployment.lcm.mirantis.com/restart-osd-reason":    "cephcluster unit test",
							"cephdeployment.lcm.mirantis.com/restart-osd-requested": "time-9",
						}
						cl.Spec.Annotations[cephv1.KeyOSD] = map[string]string{
							"cephdeployment.lcm.mirantis.com/restart-osd-reason":    "cephcluster unit test",
							"cephdeployment.lcm.mirantis.com/restart-osd-requested": "time-9",
						}
						return cl
					}(),
				}},
			},
			updated: true,
		},
		{
			name: "no update cluster - osd restart reason not changed",
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
						cl := *getClusterEnsure.DeepCopy()
						cl.Annotations = map[string]string{
							"cephdeployment.lcm.mirantis.com/restart-osd-reason":    "cephcluster unit test",
							"cephdeployment.lcm.mirantis.com/restart-osd-requested": "time-9",
						}
						cl.Spec.Annotations[cephv1.KeyOSD] = map[string]string{
							"cephdeployment.lcm.mirantis.com/restart-osd-reason":    "cephcluster unit test",
							"cephdeployment.lcm.mirantis.com/restart-osd-requested": "time-9",
						}
						return cl
					}(),
				}},
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						cm := unitinputs.BaseRookConfigOverride.DeepCopy()
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-generated"] = "time-6"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-mon-updated"] = "time-6"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-global-updated"] = "time-6"
						return *cm
					}(),
				}},
			},
		},
		{
			name:    "no update cluster - osd restart reason removed but no osd restart",
			cephDpl: unitinputs.BaseCephDeployment.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cl := *getClusterEnsure.DeepCopy()
						cl.Annotations = map[string]string{
							"cephdeployment.lcm.mirantis.com/restart-osd-reason":    "cephcluster unit test",
							"cephdeployment.lcm.mirantis.com/restart-osd-requested": "time-9",
						}
						cl.Spec.Annotations[cephv1.KeyOSD] = map[string]string{
							"cephdeployment.lcm.mirantis.com/restart-osd-reason":    "cephcluster unit test",
							"cephdeployment.lcm.mirantis.com/restart-osd-requested": "time-9",
						}
						return cl
					}(),
				}},
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						cm := unitinputs.BaseRookConfigOverride.DeepCopy()
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-generated"] = "time-6"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-mon-updated"] = "time-6"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-global-updated"] = "time-6"
						return *cm
					}(),
				}},
			},
		},
		{
			name:    "external cephcluster create failed",
			cephDpl: unitinputs.CephDeployExternal.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephclusters": unitinputs.CephClusterListEmpty.DeepCopy(),
				"secrets":      unitinputs.SecretsListEmpty.DeepCopy(),
			},
			expectedError: "unable to create external cluster configuration: failed to get secret 'lcm-namespace/pelagia-external-connection' with external connection info: secrets \"pelagia-external-connection\" not found",
		},
		{
			name:    "create cluster - external",
			cephDpl: unitinputs.CephDeployExternal.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephclusters": unitinputs.CephClusterListEmpty.DeepCopy(),
				"configmaps":   unitinputs.ConfigMapListEmpty.DeepCopy(),
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.ExternalConnectionSecretWithAdmin},
				},
			},
			expectedResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cl := unitinputs.CephClusterExternal.DeepCopy()
						cl.Status = cephv1.ClusterStatus{}
						return *cl
					}(),
				}},
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{unitinputs.RookCephMonEndpointsExternal}},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{unitinputs.ExternalConnectionSecretWithAdmin, unitinputs.RookCephMonSecret},
				},
			},
			updated: true,
		},
		{
			name:    "update cluster - external",
			cephDpl: unitinputs.CephDeployExternal.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterExternal.DeepCopy()}},
				"configmaps":   &v1.ConfigMapList{Items: []v1.ConfigMap{*unitinputs.RookCephMonEndpointsExternal.DeepCopy()}},
				"secrets": &v1.SecretList{
					Items: []v1.Secret{*unitinputs.RookCephMonSecretNonAdmin.DeepCopy(), unitinputs.ExternalConnectionSecretWithAdmin},
				},
			},
			updated: true,
		},
		{
			name:    "no update cluster - external",
			cephDpl: unitinputs.CephDeployExternal.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterExternal.DeepCopy()}},
				"configmaps": &v1.ConfigMapList{Items: []v1.ConfigMap{
					func() v1.ConfigMap {
						cm := unitinputs.RookCephMonEndpointsExternal.DeepCopy()
						cm.OwnerReferences = []metav1.OwnerReference{
							{
								APIVersion: "ceph.rook.io/v1",
								Kind:       "CephCluster",
								Name:       "cephcluster",
							},
						}
						return *cm
					}(),
				}},
				"secrets": &v1.SecretList{Items: []v1.Secret{
					unitinputs.ExternalConnectionSecretWithAdmin,
					func() v1.Secret {
						secret := unitinputs.RookCephMonSecret.DeepCopy()
						secret.OwnerReferences = []metav1.OwnerReference{
							{
								APIVersion: "ceph.rook.io/v1",
								Kind:       "CephCluster",
								Name:       "cephcluster",
							},
						}
						return *secret
					}(),
				}},
			},
		},
	}

	oldTimeFunc := lcmcommon.GetCurrentTimeString
	oldRunCmd := lcmcommon.RunPodCommandWithValidation
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			c.cdConfig.currentCephVersion = lcmcommon.LatestRelease
			c.cdConfig.currentCephImage = unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"]
			expandedNodes, err := c.buildExpandedNodeList()
			assert.Nil(t, err)
			c.cdConfig.nodesListExpanded = expandedNodes

			lcmcommon.GetCurrentTimeString = func() string {
				return fmt.Sprintf("time-%d", idx)
			}

			res := []string{"cephclusters"}
			faketestclients.FakeReaction(c.api.Rookclientset, "get", res, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", res, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", res, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"configmaps", "secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"configmaps", "secrets"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if strings.Contains(e.Command, "config dump") {
					return unitinputs.CephConfigDumpDefaults, "", nil
				}
				return "", "", errors.New("cant run ceph cmd: unknown command: " + e.Command)
			}

			updated, err := c.ensureCluster()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			assert.Equal(t, test.updated, updated)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
	// unset global timestamps
	unsetTimestampsVar()
	lcmcommon.GetCurrentTimeString = oldTimeFunc
	lcmcommon.RunPodCommandWithValidation = oldRunCmd
}

func TestHealthCluster(t *testing.T) {
	tests := []struct {
		name           string
		cephStatus     *cephv1.CephStatus
		expectedResult bool
	}{
		{
			name:           "verify health cluster - ceph cluster is HEALTH_OK",
			cephStatus:     unitinputs.ReefCephClusterReady.Status.CephStatus,
			expectedResult: true,
		},
		{
			name: "verify health cluster - ceph cluster is HEALTH_WARN, allowed issues",
			cephStatus: &cephv1.CephStatus{
				Health: "HEALTH_WARN",
				Details: map[string]cephv1.CephHealthMessage{
					"RECENT_CRASH": {
						Message:  "2 daemons have recently crashed",
						Severity: "HEALTH_WARN",
					},
				},
			},
			expectedResult: true,
		},
		{
			name:           "verify health cluster - ceph cluster is HEALTH_WARN, critical issues",
			cephStatus:     unitinputs.ReefCephClusterHasHealthIssues.Status.CephStatus,
			expectedResult: false,
		},
		{
			name: "verify health cluster - ceph cluster is HEALTH_WARN, allowed issues",
			cephStatus: &cephv1.CephStatus{
				Health: "HEALTH_ERR",
				Details: map[string]cephv1.CephHealthMessage{
					"PG_DAMAGED": {
						Message:  "Possible data damage: 5 pgs recovery_unfound",
						Severity: "HEALTH_ERR",
					},
				},
			},
			expectedResult: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			result := c.healthCluster(test.cephStatus)
			assert.Equal(t, test.expectedResult, result)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}

func TestStatusCluster(t *testing.T) {
	tests := []struct {
		name           string
		inputResources map[string]runtime.Object
		expectedError  string
	}{
		{
			name: "cephcluster status check - success",
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListReady,
			},
		},
		{
			name: "cephcluster status check - failed, state updating",
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cl := unitinputs.ReefCephClusterReady.DeepCopy()
						cl.Status.State = cephv1.ClusterStateUpdating
						return *cl
					}(),
				}},
			},
			expectedError: "ceph cluster rook-ceph/cephcluster is not ready to be updated: cluster healthy = true, cluster state = 'Updating', cluster phase = 'Ready'",
		},
		{
			name: "cephcluster status check - failed, phase progressing",
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{unitinputs.ReefCephClusterNotReady}},
			},
			expectedError: "ceph cluster rook-ceph/cephcluster is not ready to be updated: cluster healthy = true, cluster state = 'Created', cluster phase = 'Progressing'",
		},
		{
			name: "cephcluster status check - failed, creating, phase progressing",
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cl := unitinputs.ReefCephClusterNotReady.DeepCopy()
						cl.Status.State = cephv1.ClusterStateUpdating
						return *cl
					}(),
				}},
			},
			expectedError: "ceph cluster rook-ceph/cephcluster is not ready to be updated: cluster healthy = true, cluster state = 'Updating', cluster phase = 'Progressing'",
		},
		{
			name: "cephcluster status check - failed, phase progressing, not healthy",
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cl := unitinputs.ReefCephClusterHasHealthIssues.DeepCopy()
						cl.Status.State = cephv1.ClusterStateUpdating
						return *cl
					}(),
				}},
			},
			expectedError: "ceph cluster rook-ceph/cephcluster is not ready to be updated: cluster healthy = false, cluster state = 'Updating', cluster phase = 'Failure'",
		},
		{
			name: "cephcluster status check - success, not healthy",
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListHealthIssues,
			},
		},
		{
			name: "cephcluster status check - cephcluster status empty",
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{unitinputs.CephClusterGenerated}},
			},
		},
		{
			name:           "cephcluster status check - cephcluster get failed",
			inputResources: map[string]runtime.Object{"cephclusters": &unitinputs.CephClusterListEmpty},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephclusters"}, test.inputResources, nil)

			err := c.statusCluster()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}

func TestDeleteCluster(t *testing.T) {
	tests := []struct {
		name              string
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		deleted           bool
		expectedResources map[string]runtime.Object
		expectedError     string
	}{
		{
			name: "delete ceph cluster - in progress",
			inputResources: map[string]runtime.Object{
				"cephclusters": unitinputs.CephClusterListReady.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListEmpty,
			},
		},
		{
			name: "delete ceph cluster - not found",
			inputResources: map[string]runtime.Object{
				"cephclusters": &unitinputs.CephClusterListEmpty,
			},
			deleted: true,
		},
		{
			name: "delete ceph cluster - get failed",
			inputResources: map[string]runtime.Object{
				"cephclusters": unitinputs.CephClusterListReady.DeepCopy(),
			},
			apiErrors:     map[string]error{"get-cephclusters": errors.New("cephcluster get failed")},
			expectedError: "failed to get ceph cluster: cephcluster get failed",
		},
		{
			name: "delete ceph cluster - update failed",
			inputResources: map[string]runtime.Object{
				"cephclusters": unitinputs.CephClusterListReady.DeepCopy(),
			},
			apiErrors:     map[string]error{"update-cephclusters": errors.New("cephcluster update failed")},
			expectedError: "failed to update ceph cluster with cleanupPolicy: cephcluster update failed",
		},
		{
			name: "delete ceph cluster - delete failed",
			inputResources: map[string]runtime.Object{
				"cephclusters": unitinputs.CephClusterListReady.DeepCopy(),
			},
			apiErrors:     map[string]error{"delete-cephclusters": errors.New("cephcluster delete failed")},
			expectedError: "failed to delete ceph cluster: cephcluster delete failed",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: unitinputs.BaseCephDeployment.DeepCopy()}, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephclusters"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", []string{"cephclusters"}, test.inputResources, test.apiErrors)
			c.api.Rookclientset.CephV1().(*fakecephv1.FakeCephV1).AddReactor("update", "cephclusters", func(action gotesting.Action) (handled bool, ret runtime.Object, err error) {
				actual := action.(gotesting.UpdateActionImpl).Object.(*cephv1.CephCluster)
				assert.Equal(t, cephv1.CleanupPolicySpec{Confirmation: "yes-really-destroy-data", AllowUninstallWithVolumes: true}, actual.Spec.CleanupPolicy)
				return true, nil, test.apiErrors["update-cephclusters"]
			})
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			deleted, err := c.deleteCluster()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Contains(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.deleted, deleted)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
		})
	}
}
