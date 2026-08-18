/*
Copyright 2026 Mirantis IT.

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
	"reflect"
	"strings"
	"testing"

	csiopapi "github.com/ceph/ceph-csi-operator/api/v1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/v3/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/v3/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/v3/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/v3/test/unit/inputs"
)

func TestEnsureCsiResources(t *testing.T) {
	tests := []struct {
		name           string
		cephDeployment *cephlcmv1alpha1.CephDeployment
		lcmConfig      map[string]string
		operatorConfig *csiopapi.OperatorConfig
		driverList     *csiopapi.DriverList
		apiErrors      map[string]error
		expectedError  string
		expectedUpdate bool
	}{
		{
			name:           "failed to ensure operatorconfig",
			cephDeployment: &unitinputs.BaseCephDeployment,
			apiErrors: map[string]error{
				"get-operatorconfigs": errors.New("failed to get object"),
			},
			expectedError: "failed to verify cephcsi OperatorConfig 'rook-ceph/ceph-csi-operator-config': failed to get object",
		},
		{
			name:           "failed to ensure drivers",
			cephDeployment: &unitinputs.BaseCephDeployment,
			apiErrors: map[string]error{
				"get-drivers": errors.New("failed to get object"),
			},
			expectedError: "failed to ensure CephCSI Drivers",
		},
		{
			name:           "updated operatorconfig",
			cephDeployment: &unitinputs.BaseCephDeployment,
			lcmConfig: map[string]string{
				"DEPLOYMENT_CSI_DRIVERS_MANAGE":               "true",
				"DEPLOYMENT_CSI_RBD_DEFAULT_DRIVER_CREATE":    "false",
				"DEPLOYMENT_CSI_CEPHFS_DEFAULT_DRIVER_CREATE": "false",
			},
			expectedUpdate: true,
		},
		{
			name:           "updated drivers",
			cephDeployment: &unitinputs.BaseCephDeployment,
			operatorConfig: unitinputs.OperatorConfigDefault.DeepCopy(),
			expectedUpdate: true,
		},
		{
			name:           "nothing to do",
			cephDeployment: &unitinputs.BaseCephDeployment,
			operatorConfig: unitinputs.OperatorConfigDefault.DeepCopy(),
			lcmConfig: map[string]string{
				"DEPLOYMENT_CSI_DRIVERS_MANAGE":               "true",
				"DEPLOYMENT_CSI_RBD_DEFAULT_DRIVER_CREATE":    "false",
				"DEPLOYMENT_CSI_CEPHFS_DEFAULT_DRIVER_CREATE": "false",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDeployment}, test.lcmConfig)
			castErr := c.castExtensions()
			assert.Nil(t, castErr)

			builder := faketestclients.GetClientBuilder()
			if test.operatorConfig != nil {
				builder = builder.WithObjects(test.operatorConfig)
			}
			if test.driverList != nil {
				builder = builder.WithLists(test.driverList)
			}
			if test.apiErrors != nil {
				interceptorFuncs := interceptor.Funcs{}
				if v, ok := test.apiErrors["get-operatorconfigs"]; ok {
					interceptorFuncs.Get = func(ctx context.Context, client crclient.WithWatch, key crclient.ObjectKey, obj crclient.Object, opts ...crclient.GetOption) error {
						if strings.ToLower(reflect.TypeOf(obj).Elem().Name()) == "operatorconfig" {
							return v
						}
						return client.Get(ctx, key, obj, opts...)
					}
				}
				if v, ok := test.apiErrors["get-drivers"]; ok {
					interceptorFuncs.Get = func(ctx context.Context, client crclient.WithWatch, key crclient.ObjectKey, obj crclient.Object, opts ...crclient.GetOption) error {
						if strings.ToLower(reflect.TypeOf(obj).Elem().Name()) == "driver" {
							return v
						}
						return client.Get(ctx, key, obj, opts...)
					}
				}
				builder = builder.WithInterceptorFuncs(interceptorFuncs)
			}
			c.api.ClientNoCache = faketestclients.GetClient(builder)

			updated, err := c.ensureCsiResources()
			if test.expectedError == "" {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedUpdate, updated)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
				assert.Equal(t, false, updated)
			}
		})
	}
}

func TestEnsureCsiOperatorConfig(t *testing.T) {
	tests := []struct {
		name                   string
		cephDeployment         *cephlcmv1alpha1.CephDeployment
		lcmConfig              map[string]string
		operatorConfig         *csiopapi.OperatorConfig
		apiErrors              map[string]error
		expectedUpdate         bool
		expectedOperatorConfig *csiopapi.OperatorConfig
		expectedError          string
	}{
		{
			name:           "failed to get operatorConfig",
			cephDeployment: &unitinputs.BaseCephDeployment,
			apiErrors: map[string]error{
				"get-operatorconfigs": errors.New("failed to get operatorConfig"),
			},
			expectedError: "failed to verify cephcsi OperatorConfig 'rook-ceph/ceph-csi-operator-config': failed to get operatorConfig",
		},
		{
			name:           "failed to create new operatorConfig",
			cephDeployment: &unitinputs.BaseCephDeployment,
			apiErrors: map[string]error{
				"create-operatorconfigs": errors.New("failed to create operatorConfig"),
			},
			expectedError: "failed to create cephcsi OperatorConfig 'rook-ceph/ceph-csi-operator-config': failed to create operatorConfig",
		},
		{
			name:                   "created new default operatorConfig",
			cephDeployment:         &unitinputs.BaseCephDeployment,
			expectedOperatorConfig: &unitinputs.OperatorConfigDefault,
			expectedUpdate:         true,
		},
		{
			name: "create operatorConfig with spec override lcmconfig",
			cephDeployment: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployWithCSI.DeepCopy()
				cd.Spec.CSIResources.OperatorConfig.Spec = runtime.RawExtension{
					Raw: unitinputs.ConvertStructToRaw(
						csiopapi.OperatorConfigSpec{
							DriverSpecDefaults: &csiopapi.DriverSpec{
								NodePlugin: &csiopapi.NodePluginSpec{
									KubeletDirPath: "/var/lib/new-custom/kubelet",
								},
							},
						},
					)}
				return cd
			}(),
			lcmConfig: map[string]string{
				"DEPLOYMENT_CSI_DRIVERS_MANAGE": "true",
				"DEPLOYMENT_CSI_KUBELET_PATH":   "/var/lib/custom/kubelet",
			},
			expectedOperatorConfig: func() *csiopapi.OperatorConfig {
				opc := unitinputs.OperatorConfigDefault.DeepCopy()
				opc.Spec.DriverSpecDefaults.NodePlugin.KubeletDirPath = "/var/lib/new-custom/kubelet"
				return opc
			}(),
			expectedUpdate: true,
		},
		{
			name:           "failed to update existing operatorConfig",
			cephDeployment: &unitinputs.BaseCephDeployment,
			operatorConfig: unitinputs.GetOperatorConfig("ceph-csi-operator-config", unitinputs.BaseCephDeployment.Name),
			lcmConfig: map[string]string{
				"DEPLOYMENT_CSI_DRIVERS_MANAGE":           "true",
				"DEPLOYMENT_CSI_KEEP_EXISTING_ON_UPGRADE": "false",
			},
			apiErrors: map[string]error{
				"update-operatorconfigs": errors.New("failed to update operatorConfig"),
			},
			expectedOperatorConfig: unitinputs.GetOperatorConfig("ceph-csi-operator-config", unitinputs.BaseCephDeployment.Name),
			expectedError:          "failed to update cephcsi OperatorConfig 'rook-ceph/ceph-csi-operator-config': failed to update operatorConfig",
		},
		{
			name:           "updated existing default operatorConfig with custom lcmconfig options",
			cephDeployment: &unitinputs.BaseCephDeployment,
			operatorConfig: unitinputs.OperatorConfigDefault.DeepCopy(),
			lcmConfig: map[string]string{
				"DEPLOYMENT_CSI_DRIVERS_MANAGE":                 "true",
				"DEPLOYMENT_CSI_ENABLE_CSIADDONS":               "true",
				"DEPLOYMENT_CSI_KUBELET_PATH":                   "/var/lib/custom/kubelet",
				"DEPLOYMENT_CSI_CONTROLLER_PLUGIN_NODEAFFINITY": "some-node=true",
				"DEPLOYMENT_CSI_NODE_PLUGIN_NODEAFFINITY":       "some-node2=true",
				"DEPLOYMENT_CSI_CONTROLLER_PLUGIN_TOLERATIONS": `
- effect: NoSchedule
  key: node-role.kubernetes.io/controlplane
  operator: Exist`,
				"DEPLOYMENT_CSI_NODE_PLUGIN_TOLERATIONS": `
- effect: NoSchedule
  key: node-role.kubernetes.io/controlplane2
  operator: Exist`,
			},
			expectedOperatorConfig: func() *csiopapi.OperatorConfig {
				opc := unitinputs.OperatorConfigDefault.DeepCopy()
				opc.ResourceVersion = "2"
				opc.Spec.DriverSpecDefaults.DeployCsiAddons = lcmcommon.PtrTo(true)
				opc.Spec.DriverSpecDefaults.NodePlugin.KubeletDirPath = "/var/lib/custom/kubelet"
				opc.Spec.DriverSpecDefaults.NodePlugin.Affinity = &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Key:      "some-node2",
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
				}
				opc.Spec.DriverSpecDefaults.NodePlugin.Tolerations = []corev1.Toleration{
					{
						Key:      "node-role.kubernetes.io/controlplane2",
						Operator: "Exist",
						Effect:   "NoSchedule",
					},
				}
				opc.Spec.DriverSpecDefaults.ControllerPlugin.Affinity = &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Key:      "some-node",
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
				}
				opc.Spec.DriverSpecDefaults.ControllerPlugin.Tolerations = []corev1.Toleration{
					{
						Key:      "node-role.kubernetes.io/controlplane",
						Operator: "Exist",
						Effect:   "NoSchedule",
					},
				}
				return opc
			}(),
			expectedUpdate: true,
		},
		{
			name:           "updated existing created not by pelagia operatorConfig with default",
			cephDeployment: &unitinputs.BaseCephDeployment,
			operatorConfig: unitinputs.GetOperatorConfig("ceph-csi-operator-config", unitinputs.BaseCephDeployment.Name),
			lcmConfig: map[string]string{
				"DEPLOYMENT_CSI_DRIVERS_MANAGE":           "true",
				"DEPLOYMENT_CSI_KEEP_EXISTING_ON_UPGRADE": "false",
			},
			expectedOperatorConfig: func() *csiopapi.OperatorConfig {
				opc := unitinputs.OperatorConfigDefault.DeepCopy()
				opc.ResourceVersion = "2"
				return opc
			}(),
			expectedUpdate: true,
		},
		{
			name: "updated existing operatorConfig with spec full override",
			cephDeployment: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployWithCSI.DeepCopy()
				cd.Spec.CSIResources.OperatorConfig.FullOverride = true
				return cd
			}(),
			operatorConfig: unitinputs.GetOperatorConfig("ceph-csi-operator-config", unitinputs.BaseCephDeployment.Name),
			expectedOperatorConfig: func() *csiopapi.OperatorConfig {
				opc := unitinputs.GetOperatorConfig("ceph-csi-operator-config", unitinputs.BaseCephDeployment.Name)
				opc.Labels = unitinputs.OperatorConfigDefault.Labels
				opc.ResourceVersion = "2"
				return opc
			}(),
			expectedUpdate: true,
		},
		{
			name:                   "keep existing created not by pelagia and nothing to do",
			cephDeployment:         &unitinputs.BaseCephDeployment,
			operatorConfig:         unitinputs.OperatorConfigsRook.Items[0].DeepCopy(),
			expectedOperatorConfig: &unitinputs.OperatorConfigsRook.Items[0],
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDeployment}, test.lcmConfig)
			castErr := c.castExtensions()
			assert.Nil(t, castErr)

			builder := faketestclients.GetClientBuilder()
			if test.operatorConfig != nil {
				builder = builder.WithObjects(test.operatorConfig)
			}
			if test.apiErrors != nil {
				interceptorFuncs := interceptor.Funcs{}
				if v, ok := test.apiErrors["get-operatorconfigs"]; ok {
					interceptorFuncs.Get = func(ctx context.Context, client crclient.WithWatch, key crclient.ObjectKey, obj crclient.Object, opts ...crclient.GetOption) error {
						if strings.ToLower(reflect.TypeOf(obj).Elem().Name()) == "operatorconfig" {
							return v
						}
						return client.Get(ctx, key, obj, opts...)
					}
				}
				if v, ok := test.apiErrors["create-operatorconfigs"]; ok {
					interceptorFuncs.Create = func(ctx context.Context, client crclient.WithWatch, obj crclient.Object, opts ...crclient.CreateOption) error {
						if strings.ToLower(reflect.TypeOf(obj).Elem().Name()) == "operatorconfig" {
							return v
						}
						return client.Create(ctx, obj, opts...)
					}
				}
				if v, ok := test.apiErrors["update-operatorconfigs"]; ok {
					interceptorFuncs.Update = func(ctx context.Context, client crclient.WithWatch, obj crclient.Object, opts ...crclient.UpdateOption) error {
						if strings.ToLower(reflect.TypeOf(obj).Elem().Name()) == "operatorconfig" {
							return v
						}
						return client.Update(ctx, obj, opts...)
					}
				}
				builder = builder.WithInterceptorFuncs(interceptorFuncs)
			}
			c.api.ClientNoCache = faketestclients.GetClient(builder)

			updated, err := c.ensureCsiOperatorConfig()
			if test.expectedError == "" {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedUpdate, updated)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
				assert.Equal(t, false, updated)
			}
			// verify changes for operatorconfig
			opConfigList := csiopapi.OperatorConfigList{}
			err = c.api.ClientNoCache.List(c.context, &opConfigList)
			assert.Nil(t, err)
			if test.expectedOperatorConfig == nil {
				assert.Equal(t, 0, len(opConfigList.Items))
			} else {
				assert.Equal(t, 1, len(opConfigList.Items))
				assert.Equal(t, test.expectedOperatorConfig, &opConfigList.Items[0])
			}
		})
	}
}

func TestEnsureCsiDrivers(t *testing.T) {
	tests := []struct {
		name            string
		cephDeployment  *cephlcmv1alpha1.CephDeployment
		lcmConfig       map[string]string
		driverList      *csiopapi.DriverList
		vscCrdPresent   bool
		apiErrors       map[string]error
		expectedUpdate  bool
		expectedDrivers *csiopapi.DriverList
		expectedError   string
	}{
		{
			name:           "failed to get drivers",
			cephDeployment: &unitinputs.BaseCephDeployment,
			apiErrors: map[string]error{
				"get-drivers": errors.New("failed to get driver"),
			},
			expectedError: "failed to ensure CephCSI Drivers",
		},
		{
			name:           "failed to create default drivers",
			cephDeployment: &unitinputs.BaseCephDeployment,
			apiErrors: map[string]error{
				"create-drivers": errors.New("failed to create drivers"),
			},
			expectedError: "failed to ensure CephCSI Drivers",
		},
		{
			name:            "created default drivers",
			cephDeployment:  &unitinputs.BaseCephDeployment,
			vscCrdPresent:   true,
			expectedDrivers: unitinputs.CsiDriversRook,
			expectedUpdate:  true,
		},
		{
			name:           "create drivers with spec override",
			cephDeployment: &unitinputs.CephDeployWithCSI,
			vscCrdPresent:  true,
			expectedDrivers: &csiopapi.DriverList{
				Items: []csiopapi.Driver{
					func() csiopapi.Driver {
						dr := unitinputs.DriverCephFSDefault.DeepCopy()
						dr.Spec.ClusterName = &unitinputs.CephDeployWithCSI.Name
						return *dr
					}(),
					func() csiopapi.Driver {
						dr := unitinputs.DriverNFSDefault.DeepCopy()
						dr.Spec.ClusterName = &unitinputs.CephDeployWithCSI.Name
						return *dr
					}(),
					func() csiopapi.Driver {
						dr := unitinputs.GetCsiDriver("rook-ceph.nvmeof.csi.ceph.com", unitinputs.CephDeployWithCSI.Name)
						dr.Labels = unitinputs.DriverNFSDefault.Labels
						return dr
					}(),
					func() csiopapi.Driver {
						dr := unitinputs.DriverRBDDefault.DeepCopy()
						dr.Spec.ClusterName = &unitinputs.CephDeployWithCSI.Name
						return *dr
					}(),
				},
			},
			expectedUpdate: true,
		},
		{
			name:           "skip create default drivers",
			cephDeployment: &unitinputs.BaseCephDeployment,
			lcmConfig: map[string]string{
				"DEPLOYMENT_CSI_DRIVERS_MANAGE":               "true",
				"DEPLOYMENT_CSI_RBD_DEFAULT_DRIVER_CREATE":    "false",
				"DEPLOYMENT_CSI_CEPHFS_DEFAULT_DRIVER_CREATE": "false",
			},
		},
		{
			name:           "failed to update existing drivers",
			cephDeployment: &unitinputs.CephDeployWithCSI,
			driverList:     unitinputs.CsiDriversRook.DeepCopy(),
			lcmConfig: map[string]string{
				"DEPLOYMENT_CSI_DRIVERS_MANAGE":           "true",
				"DEPLOYMENT_CSI_KEEP_EXISTING_ON_UPGRADE": "false",
			},
			apiErrors: map[string]error{
				"update-drivers": errors.New("failed to update driver"),
			},
			expectedDrivers: &csiopapi.DriverList{
				Items: []csiopapi.Driver{
					unitinputs.DriverCephFSDefault,
					func() csiopapi.Driver {
						dr := unitinputs.DriverNFSDefault.DeepCopy()
						dr.Spec.ClusterName = &unitinputs.CephDeployWithCSI.Name
						return *dr
					}(),
					func() csiopapi.Driver {
						dr := unitinputs.GetCsiDriver("rook-ceph.nvmeof.csi.ceph.com", unitinputs.CephDeployWithCSI.Name)
						dr.Labels = unitinputs.DriverNFSDefault.Labels
						return dr
					}(),
					unitinputs.DriverRBDDefault,
				},
			},
			expectedError: "failed to ensure CephCSI Drivers",
		},
		{
			name:           "update existing drivers created by pelagia and not",
			cephDeployment: &unitinputs.CephDeployWithCSI,
			driverList: &csiopapi.DriverList{
				Items: []csiopapi.Driver{
					unitinputs.DriverCephFSDefault, unitinputs.DriverRBDDefault,
					unitinputs.GetCsiDriver("rook-ceph.nvmeof.csi.ceph.com", unitinputs.CephDeployWithCSI.Name)},
			},
			lcmConfig: map[string]string{
				"DEPLOYMENT_CSI_DRIVERS_MANAGE":           "true",
				"DEPLOYMENT_CSI_KEEP_EXISTING_ON_UPGRADE": "false",
			},
			expectedDrivers: &csiopapi.DriverList{
				Items: []csiopapi.Driver{
					func() csiopapi.Driver {
						dr := unitinputs.DriverCephFSDefault.DeepCopy()
						dr.ResourceVersion = "2"
						dr.Spec.ClusterName = &unitinputs.CephDeployWithCSI.Name
						return *dr
					}(),
					func() csiopapi.Driver {
						dr := unitinputs.DriverNFSDefault.DeepCopy()
						dr.Spec.ClusterName = &unitinputs.CephDeployWithCSI.Name
						return *dr
					}(),
					func() csiopapi.Driver {
						dr := unitinputs.GetCsiDriver("rook-ceph.nvmeof.csi.ceph.com", unitinputs.CephDeployWithCSI.Name)
						dr.ResourceVersion = "2"
						dr.Labels = unitinputs.DriverNFSDefault.Labels
						return dr
					}(),
					func() csiopapi.Driver {
						dr := unitinputs.DriverRBDDefault.DeepCopy()
						dr.ResourceVersion = "2"
						dr.Spec.ClusterName = &unitinputs.CephDeployWithCSI.Name
						return *dr
					}(),
				},
			},
			expectedUpdate: true,
		},
		{
			name: "updated all existing drivers with spec full override",
			cephDeployment: func() *cephlcmv1alpha1.CephDeployment {
				cd := unitinputs.CephDeployWithCSI.DeepCopy()
				cd.Spec.CSIResources.Drivers = []cephlcmv1alpha1.CephCSIDriver{
					{
						Type:         cephlcmv1alpha1.RBDCSIDriver,
						FullOverride: true,
						Spec: runtime.RawExtension{
							Raw: unitinputs.ConvertStructToRaw(
								csiopapi.DriverSpec{
									NodePlugin: &csiopapi.NodePluginSpec{
										KubeletDirPath: "custom-path",
									},
								},
							)},
					},
					{
						Type:         cephlcmv1alpha1.CephFSCSIDriver,
						FullOverride: true,
						Spec: runtime.RawExtension{
							Raw: unitinputs.ConvertStructToRaw(
								csiopapi.DriverSpec{
									NodePlugin: &csiopapi.NodePluginSpec{
										KubeletDirPath: "custom-path",
									},
								},
							)},
					},
					{
						Type:         cephlcmv1alpha1.NVMEoFCSIDriver,
						FullOverride: true,
						Spec: runtime.RawExtension{
							Raw: unitinputs.ConvertStructToRaw(
								csiopapi.DriverSpec{
									NodePlugin: &csiopapi.NodePluginSpec{
										KubeletDirPath: "custom-path",
									},
								},
							)},
					},
				}
				return cd
			}(),
			driverList: &csiopapi.DriverList{
				Items: []csiopapi.Driver{
					*unitinputs.DriverCephFSDefault.DeepCopy(),
					unitinputs.GetCsiDriver("rook-ceph.nvmeof.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					*unitinputs.DriverRBDDefault.DeepCopy(),
				},
			},
			expectedDrivers: &csiopapi.DriverList{
				Items: []csiopapi.Driver{
					func() csiopapi.Driver {
						dr := unitinputs.DriverCephFSDefault.DeepCopy()
						dr.ResourceVersion = "2"
						dr.Spec = csiopapi.DriverSpec{
							NodePlugin: &csiopapi.NodePluginSpec{
								KubeletDirPath: "custom-path",
							},
						}
						return *dr
					}(),
					func() csiopapi.Driver {
						dr := unitinputs.GetCsiDriver("rook-ceph.nvmeof.csi.ceph.com", unitinputs.CephDeployWithCSI.Name)
						dr.Labels = unitinputs.DriverCephFSDefault.Labels
						dr.ResourceVersion = "2"
						dr.Spec = csiopapi.DriverSpec{
							NodePlugin: &csiopapi.NodePluginSpec{
								KubeletDirPath: "custom-path",
							},
						}
						return dr
					}(),
					func() csiopapi.Driver {
						dr := unitinputs.DriverRBDDefault.DeepCopy()
						dr.ResourceVersion = "2"
						dr.Spec = csiopapi.DriverSpec{
							NodePlugin: &csiopapi.NodePluginSpec{
								KubeletDirPath: "custom-path",
							},
						}
						return *dr
					}(),
				},
			},
			expectedUpdate: true,
		},
		{
			name:           "keep existing created not by pelagia and nothing to do",
			cephDeployment: &unitinputs.BaseCephDeployment,
			driverList: &csiopapi.DriverList{
				Items: []csiopapi.Driver{
					unitinputs.GetCsiDriver("rook-ceph.cephfs.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					*unitinputs.DriverRBDDefault.DeepCopy(),
				},
			},
			expectedDrivers: &csiopapi.DriverList{
				Items: []csiopapi.Driver{
					unitinputs.GetCsiDriver("rook-ceph.cephfs.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					*unitinputs.DriverRBDDefault.DeepCopy(),
				},
			},
		},
		{
			name:           "fail to remove some drivers",
			cephDeployment: &unitinputs.BaseCephDeployment,
			lcmConfig: map[string]string{
				"DEPLOYMENT_CSI_DRIVERS_MANAGE":               "true",
				"DEPLOYMENT_CSI_KEEP_EXISTING_ON_UPGRADE":     "false",
				"DEPLOYMENT_CSI_RBD_DEFAULT_DRIVER_CREATE":    "false",
				"DEPLOYMENT_CSI_CEPHFS_DEFAULT_DRIVER_CREATE": "false",
				"DEPLOYMENT_CSI_NFS_DEFAULT_DRIVER_CREATE":    "false",
			},
			driverList: &csiopapi.DriverList{
				Items: []csiopapi.Driver{
					unitinputs.GetCsiDriver("rook-ceph.cephfs.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					unitinputs.GetCsiDriver("rook-ceph.nfs.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					unitinputs.GetCsiDriver("rook-ceph.nvmeof.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					*unitinputs.DriverRBDDefault.DeepCopy(),
				},
			},
			apiErrors: map[string]error{
				"delete-drivers": errors.New("failed to delete driver"),
			},
			expectedError: "failed to ensure CephCSI Drivers",
			expectedDrivers: &csiopapi.DriverList{
				Items: []csiopapi.Driver{
					unitinputs.GetCsiDriver("rook-ceph.cephfs.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					unitinputs.GetCsiDriver("rook-ceph.nfs.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					unitinputs.GetCsiDriver("rook-ceph.nvmeof.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					*unitinputs.DriverRBDDefault.DeepCopy(),
				},
			},
		},
		{
			name:           "do not remove existing not created by pelagia and remove created by pelagia",
			cephDeployment: &unitinputs.BaseCephDeployment,
			lcmConfig: map[string]string{
				"DEPLOYMENT_CSI_DRIVERS_MANAGE":               "true",
				"DEPLOYMENT_CSI_RBD_DEFAULT_DRIVER_CREATE":    "false",
				"DEPLOYMENT_CSI_CEPHFS_DEFAULT_DRIVER_CREATE": "false",
				"DEPLOYMENT_CSI_NFS_DEFAULT_DRIVER_CREATE":    "false",
			},
			driverList: &csiopapi.DriverList{
				Items: []csiopapi.Driver{
					unitinputs.GetCsiDriver("rook-ceph.cephfs.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					unitinputs.GetCsiDriver("rook-ceph.nfs.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					unitinputs.GetCsiDriver("rook-ceph.nvmeof.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					*unitinputs.DriverRBDDefault.DeepCopy(),
				},
			},
			expectedDrivers: &csiopapi.DriverList{
				Items: []csiopapi.Driver{
					unitinputs.GetCsiDriver("rook-ceph.cephfs.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					unitinputs.GetCsiDriver("rook-ceph.nfs.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					unitinputs.GetCsiDriver("rook-ceph.nvmeof.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
				},
			},
			expectedUpdate: true,
		},
		{
			name:           "remove all drivers present",
			cephDeployment: &unitinputs.BaseCephDeployment,
			lcmConfig: map[string]string{
				"DEPLOYMENT_CSI_DRIVERS_MANAGE":               "true",
				"DEPLOYMENT_CSI_KEEP_EXISTING_ON_UPGRADE":     "false",
				"DEPLOYMENT_CSI_RBD_DEFAULT_DRIVER_CREATE":    "false",
				"DEPLOYMENT_CSI_CEPHFS_DEFAULT_DRIVER_CREATE": "false",
				"DEPLOYMENT_CSI_NFS_DEFAULT_DRIVER_CREATE":    "false",
			},
			driverList: &csiopapi.DriverList{
				Items: []csiopapi.Driver{
					unitinputs.GetCsiDriver("rook-ceph.cephfs.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					unitinputs.GetCsiDriver("rook-ceph.nvmeof.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					unitinputs.GetCsiDriver("rook-ceph.nfs.csi.ceph.com", unitinputs.CephDeployWithCSI.Name),
					*unitinputs.DriverRBDDefault.DeepCopy(),
				},
			},
			expectedUpdate: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDeployment}, test.lcmConfig)
			castErr := c.castExtensions()
			assert.Nil(t, castErr)

			builder := faketestclients.GetClientBuilder()
			if test.driverList != nil {
				builder = builder.WithLists(test.driverList)
			}
			if test.vscCrdPresent {
				u := &unstructured.Unstructured{}
				u.SetName("volumegroupsnapshotclasses.groupsnapshot.storage.k8s.io")
				u.SetAPIVersion("apiextensions.k8s.io/v1")
				u.SetKind("CustomResourceDefinition")
				builder = builder.WithObjects(u)
			}
			if test.apiErrors != nil {
				interceptorFuncs := interceptor.Funcs{}
				if v, ok := test.apiErrors["get-drivers"]; ok {
					interceptorFuncs.Get = func(ctx context.Context, client crclient.WithWatch, key crclient.ObjectKey, obj crclient.Object, opts ...crclient.GetOption) error {
						if strings.ToLower(reflect.TypeOf(obj).Elem().Name()) == "driver" {
							return v
						}
						return client.Get(ctx, key, obj, opts...)
					}
				}
				if v, ok := test.apiErrors["create-drivers"]; ok {
					interceptorFuncs.Create = func(ctx context.Context, client crclient.WithWatch, obj crclient.Object, opts ...crclient.CreateOption) error {
						if strings.ToLower(reflect.TypeOf(obj).Elem().Name()) == "driver" {
							return v
						}
						return client.Create(ctx, obj, opts...)
					}
				}
				if v, ok := test.apiErrors["update-drivers"]; ok {
					interceptorFuncs.Update = func(ctx context.Context, client crclient.WithWatch, obj crclient.Object, opts ...crclient.UpdateOption) error {
						if strings.ToLower(reflect.TypeOf(obj).Elem().Name()) == "driver" {
							return v
						}
						return client.Update(ctx, obj, opts...)
					}
				}
				if v, ok := test.apiErrors["delete-drivers"]; ok {
					interceptorFuncs.Delete = func(ctx context.Context, client crclient.WithWatch, obj crclient.Object, opts ...crclient.DeleteOption) error {
						if strings.ToLower(reflect.TypeOf(obj).Elem().Name()) == "driver" {
							return v
						}
						return client.Delete(ctx, obj, opts...)
					}
				}
				builder = builder.WithInterceptorFuncs(interceptorFuncs)
			}
			c.api.ClientNoCache = faketestclients.GetClient(builder)

			updated, err := c.ensureCsiDrivers()
			if test.expectedError == "" {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedUpdate, updated)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
				assert.Equal(t, false, updated)
			}
			// verify changes for operatorconfig
			driverList := csiopapi.DriverList{}
			err = c.api.ClientNoCache.List(c.context, &driverList)
			assert.Nil(t, err)
			if test.expectedDrivers == nil {
				assert.Equal(t, 0, len(driverList.Items))
			} else {
				assert.Equal(t, test.expectedDrivers, &driverList)
			}
		})
	}
}

func TestDropCsiOperatorResources(t *testing.T) {
	builder := faketestclients.GetClientBuilder().WithLists(unitinputs.ClientProfilesRook.DeepCopy(),
		unitinputs.CsiDriversRook.DeepCopy(), unitinputs.CephConnectionsRook.DeepCopy(), unitinputs.OperatorConfigsRook.DeepCopy())
	c := fakeDeploymentConfig(nil, nil)
	c.api.ClientNoCache = faketestclients.GetClient(builder)

	tests := []struct {
		name    string
		present string
		removed bool
	}{
		{
			name: "removing clientprofile first",
		},
		{
			name: "removing other resources",
		},
		{
			name:    "no resources to remove",
			removed: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			removed, err := c.deleteCsiOperatorResources()
			assert.Nil(t, err)
			assert.Equal(t, test.removed, removed)
		})
	}
}

func TestDropCsiClientProfile(t *testing.T) {
	tests := []struct {
		name              string
		clientProfileList *csiopapi.ClientProfileList
		removed           bool
	}{
		{
			name:              "cephconnection is removing",
			clientProfileList: unitinputs.ClientProfilesRook.DeepCopy(),
		},
		{
			name:              "no cephconnection to remove",
			clientProfileList: unitinputs.ClientProfilesEmpty,
			removed:           true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			builder := faketestclients.GetClientBuilder()
			if test.clientProfileList != nil {
				builder = builder.WithLists(test.clientProfileList)
			}
			c.api.ClientNoCache = faketestclients.GetClient(builder)

			removed, err := c.deleteCsiClientProfile()
			assert.Nil(t, err)
			assert.Equal(t, test.removed, removed)
		})
	}
}

func TestDropCsiCephConnection(t *testing.T) {
	tests := []struct {
		name               string
		cephConnectionList *csiopapi.CephConnectionList
		removed            bool
	}{
		{
			name:               "cephconnection is removing",
			cephConnectionList: unitinputs.CephConnectionsRook.DeepCopy(),
		},
		{
			name:               "no cephconnection to remove",
			cephConnectionList: unitinputs.CephConnectionsEmpty,
			removed:            true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			builder := faketestclients.GetClientBuilder()
			if test.cephConnectionList != nil {
				builder = builder.WithLists(test.cephConnectionList)
			}
			c.api.ClientNoCache = faketestclients.GetClient(builder)

			removed, err := c.deleteCsiCephConnection()
			assert.Nil(t, err)
			assert.Equal(t, test.removed, removed)
		})
	}
}

func TestDropCsiOperatorConfig(t *testing.T) {
	tests := []struct {
		name         string
		opConfigList *csiopapi.OperatorConfigList
		removed      bool
	}{
		{
			name:         "operator config is removing",
			opConfigList: unitinputs.OperatorConfigsRook.DeepCopy(),
		},
		{
			name:         "no operator config to remove",
			opConfigList: unitinputs.OperatorConfigsEmpty,
			removed:      true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			builder := faketestclients.GetClientBuilder()
			if test.opConfigList != nil {
				builder = builder.WithLists(test.opConfigList)
			}
			c.api.ClientNoCache = faketestclients.GetClient(builder)

			removed, err := c.deleteCsiOperatorConfig()
			assert.Nil(t, err)
			assert.Equal(t, test.removed, removed)
		})
	}
}

func TestDropCsiDrivers(t *testing.T) {
	tests := []struct {
		name       string
		driverList *csiopapi.DriverList
		removed    bool
	}{
		{
			name:       "drivers are removing",
			driverList: unitinputs.CsiDriversRook.DeepCopy(),
		},
		{
			name:       "no drivers to remove",
			driverList: unitinputs.CsiDriversEmpty,
			removed:    true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			builder := faketestclients.GetClientBuilder()
			if test.driverList != nil {
				builder = builder.WithLists(test.driverList)
			}
			c.api.ClientNoCache = faketestclients.GetClient(builder)

			removed, err := c.deleteCsiDrivers()
			assert.Nil(t, err)
			assert.Equal(t, test.removed, removed)
		})
	}
}
