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

package lcmconfig

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	faketestclients "github.com/Mirantis/pelagia/v3/test/unit/clients"
	faketestscheme "github.com/Mirantis/pelagia/v3/test/unit/scheme"
)

func FakeReconciler(configmap *corev1.ConfigMap) *ReconcileCephDeploymentHealthConfig {
	return &ReconcileCephDeploymentHealthConfig{
		Client: faketestclients.GetClientBuilderWithObjects(configmap).Build(),
		Scheme: faketestscheme.Scheme,
	}
}

func TestInitReconcile(t *testing.T) {
	defaultConfig := defaultLcmConfig
	defaultConfig.HealthParams = &defaultHealthConfig
	defaultConfig.TaskParams = &defaultTaskConfig
	defaultConfig.DeployParams = &defaultDeployParams
	tests := []struct {
		name              string
		namespace         string
		configMap         *corev1.ConfigMap
		controlParams     ControlParams
		expectedLcmConfig map[string]LcmConfig
		expectedResult    reconcile.Result
	}{
		{
			name:      "configmap is added with custom params",
			namespace: "some-namespace",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: LcmConfigMapName, Namespace: "some-namespace"},
				Data: map[string]string{
					"ROOK_NAMESPACE":                "another-rook-ceph",
					"DEPLOYMENT_CSI_DRIVERS_MANAGE": "false",
				},
			},
			controlParams:  ControlParamsAll,
			expectedResult: reconcile.Result{},
			expectedLcmConfig: map[string]LcmConfig{
				"some-namespace": func() LcmConfig {
					newConfig := defaultLcmConfig
					newConfig.RookNamespace = "another-rook-ceph"
					newConfig.HealthParams = &defaultHealthConfig
					newConfig.TaskParams = &defaultTaskConfig
					deployParams := defaultDeployParams
					deployParams.CSIParams.Manage = false
					newConfig.DeployParams = &deployParams
					return newConfig
				}(),
			},
		},
		{
			name:      "new configmap is added default params",
			namespace: "some-namespace-2",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: LcmConfigMapName, Namespace: "some-namespace-2"},
			},
			controlParams:  ControlParamsAll,
			expectedResult: reconcile.Result{},
			expectedLcmConfig: map[string]LcmConfig{
				"some-namespace": func() LcmConfig {
					newConfig := defaultLcmConfig
					newConfig.RookNamespace = "another-rook-ceph"
					newConfig.HealthParams = &defaultHealthConfig
					newConfig.TaskParams = &defaultTaskConfig
					deployParams := defaultDeployParams
					deployParams.CSIParams.Manage = false
					newConfig.DeployParams = &deployParams
					return newConfig
				}(),
				"some-namespace-2": defaultConfig,
			},
		},
		{
			name:      "configmap is not found",
			namespace: "some-namespace",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: LcmConfigMapName, Namespace: "fake-ns"},
			},
			controlParams:  ControlParamsAll,
			expectedResult: reconcile.Result{},
			expectedLcmConfig: map[string]LcmConfig{
				"some-namespace-2": defaultConfig,
			},
		},
		{
			name:      "configmap is updated",
			namespace: "some-namespace-2",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: LcmConfigMapName, Namespace: "some-namespace-2"},
				Data: map[string]string{
					"ROOK_NAMESPACE":                                "custom-rook-ceph",
					"GATEWAY_API_ENABLED":                           "false",
					"GATEWAY_BASE_NAME":                             "custom-gateway",
					"GATEWAY_BASE_NAMESPACE":                        "custom-namespace",
					"KEEP_INGRESS":                                  "true",
					"DISK_DAEMON_API_PORT":                          "9998",
					"DISK_DAEMON_PLACEMENT_NODES_SELECTOR":          "custom-node-label=true",
					"HEALTH_CHECKS_CEPH_ISSUES_TO_IGNORE":           "MON_DOWN,HOST_DOWN",
					"HEALTH_CHECKS_SKIP":                            "ceph_daemons,rgw_info",
					"HEALTH_CHECKS_USAGE_CLASS_FILTER":              "hdd",
					"HEALTH_CHECKS_USAGE_POOLS_FILTER":              "pool-.+",
					"RGW_PUBLIC_ACCESS_SERVICE_SELECTOR":            "custom-access-label=true",
					"HEALTH_LOG_LEVEL":                              "warn",
					"TASK_LOG_LEVEL":                                "warn",
					"DEPLOYMENT_LOG_LEVEL":                          "warn",
					"TASK_OSD_PG_REBALANCE_TIMEOUT_MIN":             "10",
					"TASK_ALLOW_REMOVE_MANUALLY_CREATED_LVMS":       "true",
					"DEPLOYMENT_OPENSTACK_CEPH_SHARED_NAMESPACE":    "custom-openstack-ns",
					"DEPLOYMENT_LABEL_TO_EXCLUDE_CEPH_DAEMONSETS":   "no-ceph=true",
					"DEPLOYMENT_DRAIN_REQUEST_LABEL_KEY":            "custom-label/drain-request",
					"DEPLOYMENT_DRAIN_READY_LABEL_KEY":              "custom-label/csi-drain-ready",
					"DEPLOYMENT_CSI_DRIVERS_MANAGE":                 "true",
					"DEPLOYMENT_CSI_RBD_DEFAULT_DRIVER_CREATE":      "false",
					"DEPLOYMENT_CSI_CEPHFS_DEFAULT_DRIVER_CREATE":   "false",
					"DEPLOYMENT_CSI_NFS_DEFAULT_DRIVER_CREATE":      "true",
					"DEPLOYMENT_CSI_KEEP_EXISTING_ON_UPGRADE":       "false",
					"DEPLOYMENT_CSI_KUBELET_PATH":                   "/var/lib/kubelet-custom",
					"DEPLOYMENT_CSI_ENABLE_CSIADDONS":               "true",
					"DEPLOYMENT_CSI_CONTROLLER_PLUGIN_NODEAFFINITY": "some-label=true",
					"DEPLOYMENT_CSI_NODE_PLUGIN_NODEAFFINITY":       "some-label-2=true",
					"DEPLOYMENT_CSI_CONTROLLER_PLUGIN_TOLERATIONS": `
- effect: NoSchedule
  key: node-role.kubernetes.io/controlplane
  operator: Exist`,
					"DEPLOYMENT_CSI_NODE_PLUGIN_TOLERATIONS": `
- effect: NoSchedule
  key: node-role.kubernetes.io/controlplane2
  operator: Exist`,
				},
			},
			controlParams:  ControlParamsAll,
			expectedResult: reconcile.Result{},
			expectedLcmConfig: map[string]LcmConfig{
				"some-namespace-2": func() LcmConfig {
					newConfig := defaultLcmConfig
					newConfig.RookNamespace = "custom-rook-ceph"
					newConfig.CommonParams.BaseGatewayName = "custom-gateway"
					newConfig.CommonParams.BaseGatewayNamespace = "custom-namespace"
					newConfig.CommonParams.DiskDaemonPort = 9998
					newConfig.CommonParams.DiskDaemonPlacementLabel = "custom-node-label=true"
					newConfig.CommonParams.RgwPublicAccessLabel = "custom-access-label=true"
					newConfig.CommonParams.KeepIngress = true
					newConfig.CommonParams.GatewayAPIEnabled = false
					newConfig.HealthParams = &HealthParams{
						LogLevel:                  2,
						ChecksSkip:                []string{"ceph_daemons", "rgw_info"},
						CephIssuesToIgnore:        []string{"MON_DOWN", "HOST_DOWN"},
						UsageDetailsClassesFilter: "hdd",
						UsageDetailsPoolsFilter:   "pool-.+",
					}
					newConfig.TaskParams = &TaskParams{
						LogLevel:                        2,
						OsdPgRebalanceTimeout:           10 * time.Minute,
						AllowToRemoveManuallyCreatedLVM: true,
					}
					newConfig.DeployParams = &DeployParams{
						LogLevel:                           2,
						OpenstackCephSharedNamespace:       "custom-openstack-ns",
						CephDaemonsetPlacementLabelExclude: "no-ceph=true",
						DrainRequestLabelKey:               "custom-label/drain-request",
						DrainReadyLabelKey:                 "custom-label/csi-drain-ready",
						CSIParams: CSIDeployParams{
							Manage:                 true,
							KubeletPath:            "/var/lib/kubelet-custom",
							CreateDefaultNFSDriver: true,
							DeployCSIAddons:        true,
							ControllerPluginNodeAffinity: &corev1.NodeAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
									NodeSelectorTerms: []corev1.NodeSelectorTerm{
										{
											MatchExpressions: []corev1.NodeSelectorRequirement{
												{
													Key:      "some-label",
													Operator: "In",
													Values: []string{
														"true",
													},
												},
											},
										},
									},
								},
							},
							ControllerPluginToleration: []corev1.Toleration{
								{
									Key:      "node-role.kubernetes.io/controlplane",
									Operator: "Exist",
									Effect:   "NoSchedule",
								},
							},
							NodePluginNodeAffinity: &corev1.NodeAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
									NodeSelectorTerms: []corev1.NodeSelectorTerm{
										{
											MatchExpressions: []corev1.NodeSelectorRequirement{
												{
													Key:      "some-label-2",
													Operator: "In",
													Values: []string{
														"true",
													},
												},
											},
										},
									},
								},
							},
							NodePluginToleration: []corev1.Toleration{
								{
									Key:      "node-role.kubernetes.io/controlplane2",
									Operator: "Exist",
									Effect:   "NoSchedule",
								},
							},
						},
					}
					return newConfig
				}(),
			},
		},
		{
			name:      "configmap has incorrect values for some params, defaults used",
			namespace: "some-namespace-2",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: LcmConfigMapName, Namespace: "some-namespace-2"},
				Data: map[string]string{
					"DISK_DAEMON_API_PORT":                          "99ww98",
					"DISK_DAEMON_PLACEMENT_NODES_SELECTOR":          "custom-^^-label=true,asss",
					"HEALTH_CHECKS_USAGE_CLASS_FILTER":              "(hdd|",
					"HEALTH_CHECKS_USAGE_POOLS_FILTER":              "(pool-|",
					"RGW_PUBLIC_ACCESS_SERVICE_SELECTOR":            "custom&^^^-access-label",
					"GATEWAY_API_ENABLED":                           "fa;sfla",
					"KEEP_INGRESS":                                  "asr32",
					"HEALTH_LOG_LEVEL":                              "fakelevel",
					"TASK_LOG_LEVEL":                                "fakelevel",
					"TASK_OSD_PG_REBALANCE_TIMEOUT_MIN":             "10asdasd",
					"TASK_ALLOW_REMOVE_MANUALLY_CREATED_LVMS":       "dsf3",
					"DEPLOYMENT_LABEL_TO_EXCLUDE_CEPH_DAEMONSETS":   "no-ceph@@@true",
					"DEPLOYMENT_CSI_DRIVERS_MANAGE":                 "true",
					"DEPLOYMENT_CSI_RBD_DEFAULT_DRIVER_CREATE":      "faasdsadlse",
					"DEPLOYMENT_CSI_CEPHFS_DEFAULT_DRIVER_CREATE":   "faasslse",
					"DEPLOYMENT_CSI_NFS_DEFAULT_DRIVER_CREATE":      "trdsdue",
					"DEPLOYMENT_CSI_KEEP_EXISTING_ON_UPGRADE":       "falssdsde",
					"DEPLOYMENT_CSI_ENABLE_CSIADDONS":               "trusdsde",
					"DEPLOYMENT_CSI_CONTROLLER_PLUGIN_NODEAFFINITY": "s@#$ome-label=true",
					"DEPLOYMENT_CSI_NODE_PLUGIN_NODEAFFINITY":       "some-label-2W@@$=true",
					"DEPLOYMENT_CSI_CONTROLLER_PLUGIN_TOLERATIONS":  "|asdasd",
					"DEPLOYMENT_CSI_NODE_PLUGIN_TOLERATIONS":        "|asdasd",
				},
			},
			controlParams:  ControlParamsAll,
			expectedResult: reconcile.Result{},
			expectedLcmConfig: map[string]LcmConfig{
				"some-namespace-2": defaultConfig,
			},
		},
		{
			name:      "configmap loaded controller specific params #1",
			namespace: "some-namespace-2",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: LcmConfigMapName, Namespace: "some-namespace-2"},
				Data:       map[string]string{},
			},
			controlParams:  ControlParamsHealth,
			expectedResult: reconcile.Result{},
			expectedLcmConfig: map[string]LcmConfig{
				"some-namespace-2": func() LcmConfig {
					newConfig := defaultLcmConfig
					newConfig.HealthParams = &defaultHealthConfig
					return newConfig
				}(),
			},
		},
		{
			name:      "configmap loaded controller specific params #2",
			namespace: "some-namespace-2",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: LcmConfigMapName, Namespace: "some-namespace-2"},
				Data:       map[string]string{},
			},
			controlParams:  ControlParamsTask,
			expectedResult: reconcile.Result{},
			expectedLcmConfig: map[string]LcmConfig{
				"some-namespace-2": func() LcmConfig {
					newConfig := defaultLcmConfig
					newConfig.TaskParams = &defaultTaskConfig
					return newConfig
				}(),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: test.namespace,
					Name:      LcmConfigMapName,
				},
			}

			ParamsToControl = test.controlParams

			r := FakeReconciler(test.configMap)
			result, err := r.Reconcile(context.TODO(), request)
			assert.Nil(t, err)

			ParamsToControl = ControlParamsAll

			assert.Equal(t, test.expectedResult, result)
			assert.Equal(t, test.expectedLcmConfig, lcmConfigs)
		})
	}
}
