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

package input

import (
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

var CephDeploymentDeprecated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		DashboardEnabled: &[]bool{true}[0],
		DataDirHostPath:  "/var/lib/custom-path",
		HealthCheck: &cephlcmv1alpha1.CephClusterHealthCheckSpec{
			DaemonHealth: cephv1.DaemonHealthSpec{
				Status:              cephv1.HealthCheckSpec{Disabled: true},
				ObjectStorageDaemon: cephv1.HealthCheckSpec{Timeout: "60s"},
				Monitor:             cephv1.HealthCheckSpec{Timeout: "60s"},
			},
			LivenessProbe: map[cephv1.KeyType]*cephv1.ProbeSpec{
				"osd": {
					Probe: &v1.Probe{
						TimeoutSeconds:   10,
						FailureThreshold: 10,
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
			},
		},
		HyperConverge: &cephlcmv1alpha1.CephDeploymentHyperConverge{
			Resources: cephv1.ResourceSpec{
				"osd-nvme": v1.ResourceRequirements{
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("100m"),
						v1.ResourceMemory: resource.MustParse("156Mi"),
					},
					Requests: v1.ResourceList{
						v1.ResourceMemory: resource.MustParse("28Mi"),
						v1.ResourceCPU:    resource.MustParse("10m"),
					},
				},
			},
			Tolerations: map[string]cephlcmv1alpha1.CephDeploymentToleration{
				"all": {
					Rules: []v1.Toleration{
						{
							Key:      "test.kubernetes.io/testkey",
							Effect:   "Schedule",
							Operator: "Exists",
						},
					},
				},
				"mgr": {
					Rules: []v1.Toleration{
						{
							Key:      "test.kubernetes.io/testkey",
							Effect:   "Schedule",
							Operator: "Exists",
						},
					},
				},
				"mon": {
					Rules: []v1.Toleration{
						{
							Key:      "test.kubernetes.io/testkey",
							Effect:   "Schedule",
							Operator: "Exists",
						},
					},
				},
				"osd": {
					Rules: []v1.Toleration{
						{
							Key:      "test.kubernetes.io/testkey",
							Effect:   "Schedule",
							Operator: "Exists",
						},
					},
				},
			},
		},
		Mgr: &cephlcmv1alpha1.Mgr{
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
		},
		Network: &cephlcmv1alpha1.CephNetworkSpec{
			ClusterNet: "127.0.0.0/16",
			PublicNet:  "192.168.0.0/16",
		},
		Nodes: CephNodesOk,
	},
}

var CephDeploymentMigrated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Cluster: &cephlcmv1alpha1.CephCluster{
			ClusterSpec: cephv1.ClusterSpec{
				Dashboard:       cephv1.DashboardSpec{Enabled: true},
				DataDirHostPath: "/var/lib/custom-path",
				HealthCheck: cephv1.CephClusterHealthCheckSpec{
					DaemonHealth: cephv1.DaemonHealthSpec{
						Status:              cephv1.HealthCheckSpec{Disabled: true},
						ObjectStorageDaemon: cephv1.HealthCheckSpec{Timeout: "60s"},
						Monitor:             cephv1.HealthCheckSpec{Timeout: "60s"},
					},
					LivenessProbe: map[cephv1.KeyType]*cephv1.ProbeSpec{
						"osd": {
							Probe: &v1.Probe{
								TimeoutSeconds:   10,
								FailureThreshold: 10,
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
					},
				},
				Mgr: cephv1.MgrSpec{
					Modules: []cephv1.Module{
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
					},
				},
				Network: cephv1.NetworkSpec{
					AddressRanges: &cephv1.AddressRangesSpec{
						Public:  []cephv1.CIDR{cephv1.CIDR("192.168.0.0/16")},
						Cluster: []cephv1.CIDR{cephv1.CIDR("127.0.0.0/16")},
					},
				},
				Placement: cephv1.PlacementSpec{
					"all": {
						Tolerations: []v1.Toleration{
							{
								Key:      "test.kubernetes.io/testkey",
								Effect:   "Schedule",
								Operator: "Exists",
							},
						},
					},
					"mgr": {
						Tolerations: []v1.Toleration{
							{
								Key:      "test.kubernetes.io/testkey",
								Effect:   "Schedule",
								Operator: "Exists",
							},
						},
					},
					"mon": {
						Tolerations: []v1.Toleration{
							{
								Key:      "test.kubernetes.io/testkey",
								Effect:   "Schedule",
								Operator: "Exists",
							},
						},
					},
					"osd": {
						Tolerations: []v1.Toleration{
							{
								Key:      "test.kubernetes.io/testkey",
								Effect:   "Schedule",
								Operator: "Exists",
							},
						},
					},
				},
				Resources: cephv1.ResourceSpec{
					"osd-nvme": v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("100m"),
							v1.ResourceMemory: resource.MustParse("156Mi"),
						},
						Requests: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("28Mi"),
							v1.ResourceCPU:    resource.MustParse("10m"),
						},
					},
				},
			},
		},
		Nodes: CephNodesOk,
	},
}

var CephDeploymentMultusDeprecated = func() cephlcmv1alpha1.CephDeployment {
	cd := CephDeploymentDeprecated.DeepCopy()
	cd.Spec.Network.Provider = "multus"
	cd.Spec.Network.Selector = map[cephv1.CephNetworkType]string{
		cephv1.CephNetworkPublic:  "192.168.0.0/16",
		cephv1.CephNetworkCluster: "127.0.0.0/16",
	}
	return *cd
}()

var CephDeploymentMultusMigrated = func() cephlcmv1alpha1.CephDeployment {
	cd := CephDeploymentMigrated.DeepCopy()
	cd.Spec.Cluster.Network.Provider = "multus"
	cd.Spec.Cluster.Network.Selectors = map[cephv1.CephNetworkType]string{
		cephv1.CephNetworkPublic:  "192.168.0.0/16",
		cephv1.CephNetworkCluster: "127.0.0.0/16",
	}
	return *cd
}()

var CephDeployExternalDeprecated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Network: &cephlcmv1alpha1.CephNetworkSpec{
			ClusterNet: "127.0.0.0/32",
			PublicNet:  "127.0.0.0/32",
		},
		External: &[]bool{true}[0],
	},
}

var CephDeployExternalMigrated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Cluster: &cephlcmv1alpha1.CephCluster{
			ClusterSpec: cephv1.ClusterSpec{External: cephv1.ExternalSpec{Enable: true}},
		},
	},
}
