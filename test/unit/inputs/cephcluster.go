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

package input

import (
	"time"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var CephClusterListEmpty = cephv1.CephClusterList{Items: []cephv1.CephCluster{}}
var CephClusterListNotSupported = cephv1.CephClusterList{Items: []cephv1.CephCluster{OctopusCephCluster}}
var CephClusterListReady = cephv1.CephClusterList{Items: []cephv1.CephCluster{ReefCephClusterReady}}
var CephClusterListNotReady = cephv1.CephClusterList{Items: []cephv1.CephCluster{ReefCephClusterNotReady}}
var CephClusterListHealthIssues = cephv1.CephClusterList{Items: []cephv1.CephCluster{ReefCephClusterHasHealthIssues}}
var CephClusterListExternal = cephv1.CephClusterList{Items: []cephv1.CephCluster{CephClusterExternal}}

func BuildBaseCephCluster(name, namespace string) cephv1.CephCluster {
	cephcluster := cephv1.CephCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  namespace,
			Name:       name,
			Generation: 4,
		},
		Spec: cephv1.ClusterSpec{
			Mon: cephv1.MonSpec{Count: 3},
			Mgr: cephv1.MgrSpec{Count: 1},
			Storage: cephv1.StorageScopeSpec{
				Nodes: StorageNodesForAnalysisOk,
			},
		},
		Status: cephv1.ClusterStatus{},
	}
	return cephcluster
}

var OctopusCephCluster = func() cephv1.CephCluster {
	newcluster := BuildBaseCephCluster(LcmObjectMeta.Name, RookNamespace)
	newcluster.Status = cephv1.ClusterStatus{
		Phase: cephv1.ConditionReady,
		CephStatus: &cephv1.CephStatus{
			Health: "HEALTH_OK",
			FSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
		},
		CephVersion: &cephv1.ClusterVersion{
			Image:   "some-registry.com/ceph:v15.2.8",
			Version: "15.2.8-0",
		},
	}
	return newcluster
}()

var ReefCephClusterReady = func() cephv1.CephCluster {
	newcluster := BuildBaseCephCluster(LcmObjectMeta.Name, RookNamespace)
	newcluster.Spec.CephVersion.Image = "some-registry.com/ceph:v18.2.4"
	newcluster.Status = cephv1.ClusterStatus{
		Phase: cephv1.ConditionReady,
		State: cephv1.ClusterStateCreated,
		CephStatus: &cephv1.CephStatus{
			Health:      "HEALTH_OK",
			FSID:        "8668f062-3faa-358a-85f3-f80fe6c1e306",
			LastChecked: time.Now().Format(time.RFC3339),
		},
		CephVersion: &cephv1.ClusterVersion{
			Image:   "some-registry.com/ceph:v18.2.4",
			Version: "18.2.4-0",
		},
	}
	return newcluster
}()

var ReefCephClusterNotReady = func() cephv1.CephCluster {
	newcluster := BuildBaseCephCluster(LcmObjectMeta.Name, RookNamespace)
	newcluster.Status = cephv1.ClusterStatus{
		Phase:      cephv1.ConditionProgressing,
		State:      cephv1.ClusterStateCreated,
		CephStatus: ReefCephClusterReady.Status.CephStatus,
	}
	return newcluster
}()

var ReefCephClusterHasHealthIssues = func() cephv1.CephCluster {
	newcluster := BuildBaseCephCluster(LcmObjectMeta.Name, RookNamespace)
	newcluster.Spec.Storage.Nodes = StorageNodesForAnalysisNotAllSpecified
	newcluster.Status = cephv1.ClusterStatus{
		Phase: cephv1.ConditionFailure,
		CephStatus: &cephv1.CephStatus{
			Health: "HEALTH_WARN",
			FSID:   "8668f062-3faa-358a-85f3-f80fe6c1e306",
			Details: map[string]cephv1.CephHealthMessage{
				"RECENT_MGR_MODULE_CRASH": {
					Severity: "HEALTH_WARN",
					Message:  "2 mgr modules have recently crashed",
				},
			},
		},
		CephVersion: &cephv1.ClusterVersion{
			Image:   "some-registry.com/ceph:v18.2.4",
			Version: "18.2.4-0",
		},
	}
	return newcluster
}()

var CephClusterExternal = cephv1.CephCluster{
	ObjectMeta: metav1.ObjectMeta{
		Name:      LcmObjectMeta.Name,
		Namespace: RookNamespace,
	},
	Spec: cephv1.ClusterSpec{
		CephVersion: cephv1.CephVersionSpec{Image: PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"]},
		ContinueUpgradeAfterChecksEvenIfNotHealthy: true,
		DataDirHostPath:   "/var/lib/rook",
		SkipUpgradeChecks: true,
		External:          cephv1.ExternalSpec{Enable: true},
	},
	Status: cephv1.ClusterStatus{
		Phase: cephv1.ConditionConnected,
		CephStatus: &cephv1.CephStatus{
			Health:      "HEALTH_OK",
			FSID:        "8668f062-3faa-358a-85f3-f80fe6c1e306",
			LastChecked: time.Now().Format(time.RFC3339),
		},
		CephVersion: &cephv1.ClusterVersion{
			Image:   "some-registry.com/ceph:v18.2.4",
			Version: "18.2.4-0",
		},
	},
}

var CephClusterGenerated = cephv1.CephCluster{
	ObjectMeta: metav1.ObjectMeta{
		Name:      LcmObjectMeta.Name,
		Namespace: RookNamespace,
	},
	Spec: cephv1.ClusterSpec{
		CephVersion: cephv1.CephVersionSpec{Image: PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"]},
		Annotations: map[cephv1.KeyType]cephv1.Annotations{
			cephv1.KeyMon: map[string]string{
				"cephdeployment.lcm.mirantis.com/config-global-updated": "some-time",
				"cephdeployment.lcm.mirantis.com/config-mon-updated":    "some-time",
			},
			cephv1.KeyMgr: map[string]string{
				"cephdeployment.lcm.mirantis.com/config-global-updated": "some-time",
				"cephdeployment.lcm.mirantis.com/config-mgr-updated":    "some-time",
			},
		},
		ContinueUpgradeAfterChecksEvenIfNotHealthy: true,
		DataDirHostPath: "/var/lib/rook",
		Mon:             cephv1.MonSpec{Count: 3},
		Mgr: cephv1.MgrSpec{
			Count: 1,
			Modules: []cephv1.Module{
				{
					Name:    "balancer",
					Enabled: true,
				},
				{
					Name:    "pg_autoscaler",
					Enabled: true,
				},
			},
		},
		Network: cephv1.NetworkSpec{
			Provider: "host",
		},
		Placement: cephv1.PlacementSpec{
			cephv1.KeyMon: cephv1.Placement{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "ceph_role_mon",
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
				PodAffinity:     &corev1.PodAffinity{},
				PodAntiAffinity: &corev1.PodAntiAffinity{},
				Tolerations: []corev1.Toleration{
					{
						Key:      "ceph_role_mon",
						Operator: "Exists",
					},
				},
			},
			cephv1.KeyMgr: cephv1.Placement{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "ceph_role_mgr",
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
				PodAffinity:     &corev1.PodAffinity{},
				PodAntiAffinity: &corev1.PodAntiAffinity{},
				Tolerations: []corev1.Toleration{
					{
						Key:      "ceph_role_mgr",
						Operator: "Exists",
					},
				},
			},
		},
		Storage: cephv1.StorageScopeSpec{
			UseAllNodes: false,
			Selection:   cephv1.Selection{UseAllDevices: &[]bool{false}[0]},
			Nodes: []cephv1.Node{
				{
					Name: "node-1",
					Selection: cephv1.Selection{
						UseAllDevices: nil,
						DeviceFilter:  "",
						Devices: []cephv1.Device{
							{
								Name:   "sda",
								Config: map[string]string{"deviceClass": "hdd"},
							},
						},
						DevicePathFilter:     "",
						VolumeClaimTemplates: nil,
					},
					Config: nil,
				},
				{
					Name: "node-2",
					Selection: cephv1.Selection{
						UseAllDevices: nil,
						DeviceFilter:  "",
						Devices: []cephv1.Device{
							{
								Name:   "sda",
								Config: map[string]string{"osdsPerDevice": "1", "deviceClass": "hdd"},
							},
						},
						DevicePathFilter:     "",
						VolumeClaimTemplates: nil,
					},
					Config: nil,
				},
				{
					Name: "node-3",
					Selection: cephv1.Selection{
						UseAllDevices: nil,
						DeviceFilter:  "",
						Devices: []cephv1.Device{
							{
								Name:   "sda",
								Config: map[string]string{"deviceClass": "hdd"},
							},
						},
						DevicePathFilter:     "",
						VolumeClaimTemplates: nil,
					},
					Config: map[string]string{"osdsPerDevice": "2"},
				},
			},
		},
		SkipUpgradeChecks: true,
		HealthCheck: cephv1.CephClusterHealthCheckSpec{
			LivenessProbe: map[cephv1.KeyType]*cephv1.ProbeSpec{
				"osd": {
					Probe: &corev1.Probe{
						TimeoutSeconds:   5,
						FailureThreshold: 5,
					},
				},
				"mon": {
					Probe: &corev1.Probe{
						TimeoutSeconds:   5,
						FailureThreshold: 5,
					},
				},
				"mgr": {
					Probe: &corev1.Probe{
						TimeoutSeconds:   5,
						FailureThreshold: 5,
					},
				},
			},
		},
	},
}

var TestCephCluster = cephv1.CephCluster{
	ObjectMeta: metav1.ObjectMeta{
		Name:      LcmObjectMeta.Name,
		Namespace: RookNamespace,
	},
	Spec: cephv1.ClusterSpec{
		CephVersion: CephClusterGenerated.Spec.CephVersion,
		ContinueUpgradeAfterChecksEvenIfNotHealthy: true,
		DataDirHostPath: "/var/lib/rook",
		Mon:             CephClusterGenerated.Spec.Mon,
		Mgr:             CephClusterGenerated.Spec.Mgr,
		Network:         CephClusterGenerated.Spec.Network,
		Placement:       CephClusterGenerated.Spec.Placement,
		Storage: cephv1.StorageScopeSpec{
			Selection: cephv1.Selection{UseAllDevices: &[]bool{false}[0]},
			Nodes: []cephv1.Node{
				{
					Name: "node-1",
					Selection: cephv1.Selection{
						UseAllDevices: nil,
						DeviceFilter:  "",
						Devices: []cephv1.Device{
							{
								Name:     "sda",
								FullPath: "",
								Config:   map[string]string{"deviceClass": "hdd"},
							},
							{
								Name:     "sdb",
								FullPath: "",
								Config:   map[string]string{"deviceClass": "hdd"},
							},
						},
						DevicePathFilter:     "",
						VolumeClaimTemplates: nil,
					},
					Config: nil,
				},
				{
					Name: "node-2",
					Selection: cephv1.Selection{
						UseAllDevices: nil,
						DeviceFilter:  "",
						Devices: []cephv1.Device{
							{
								Name:     "sda",
								FullPath: "",
								Config:   map[string]string{"osdsPerDevice": "1", "deviceClass": "hdd"},
							},
							{
								Name:     "sdb",
								FullPath: "",
								Config:   map[string]string{"osdsPerDevice": "2", "deviceClass": "hdd"},
							},
							{
								Name:     "sdc",
								FullPath: "",
								Config:   map[string]string{"metadataDevice": "sde", "deviceClass": "hdd"},
							},
						},
						DevicePathFilter:     "",
						VolumeClaimTemplates: nil,
					},
					Config: nil,
				},
				{
					Name: "node-3",
					Selection: cephv1.Selection{
						UseAllDevices: nil,
						DeviceFilter:  "",
						Devices: []cephv1.Device{
							{
								Name:     "sda",
								FullPath: "",
								Config:   map[string]string{"deviceClass": "hdd"},
							},
						},
						DevicePathFilter:     "",
						VolumeClaimTemplates: nil,
					},
					Config: map[string]string{"osdsPerDevice": "2"},
				},
			},
		},
		SkipUpgradeChecks: true,
		HealthCheck:       CephClusterGenerated.Spec.HealthCheck,
	},
}

func CephClusterOpenstack() *cephv1.CephCluster {
	cephcluster := TestCephCluster.DeepCopy()
	delete(cephcluster.Spec.HealthCheck.LivenessProbe, "mds")
	return cephcluster
}

var StorageNodesForAnalysisOk = []cephv1.Node{
	{
		Name: "node-1",
		Selection: cephv1.Selection{
			Devices: []cephv1.Device{
				{
					Name: "vdb",
					Config: map[string]string{
						"deviceClass":    "hdd",
						"metadataDevice": "/dev/vda14",
					},
				},
				{
					FullPath: "/dev/disk/by-path/pci-0000:00:0f.0",
					Config: map[string]string{
						"deviceClass":    "hdd",
						"metadataDevice": "/dev/disk/by-id/virtio-e8d89e2f-ffc6-4988-9",
					},
				},
				{
					Name: "vdf",
					Config: map[string]string{
						"deviceClass":    "hdd",
						"metadataDevice": "/dev/ceph-metadata/part-2",
					},
				},
			},
		},
	},
	{
		Name: "node-2",
		Selection: cephv1.Selection{
			Devices: []cephv1.Device{
				{
					Name: "vdb",
					Config: map[string]string{
						"deviceClass": "hdd",
					},
				},
				{
					Name: "vdd",
					Config: map[string]string{
						"deviceClass":   "hdd",
						"osdsPerDevice": "2",
					},
				},
			},
		},
	},
}

var StorageNodesForAnalysisNotAllSpecified = []cephv1.Node{
	{
		Name: "node-1",
		Selection: cephv1.Selection{
			Devices: []cephv1.Device{
				{
					Name: "vdb",
					Config: map[string]string{
						"deviceClass":    "hdd",
						"metadataDevice": "/dev/vda14",
					},
				},
				{
					Name: "vdf",
					Config: map[string]string{
						"deviceClass":    "hdd",
						"metadataDevice": "/dev/ceph-metadata/part-2",
					},
				},
			},
		},
	},
	{
		Name: "node-2",
		Selection: cephv1.Selection{
			Devices: []cephv1.Device{
				{
					Name: "vdb",
					Config: map[string]string{
						"deviceClass": "hdd",
					},
				},
			},
		},
	},
}

var StorageNodesForRequestReduced = []cephv1.Node{
	{
		Name: "node-2",
		Selection: cephv1.Selection{
			Devices: []cephv1.Device{
				{
					Name: "vdb",
					Config: map[string]string{
						"deviceClass": "hdd",
					},
				},
				{
					Name: "vdd",
					Config: map[string]string{
						"deviceClass":   "hdd",
						"osdsPerDevice": "2",
					},
				},
			},
		},
	},
}

var StorageNodesForRequestFiltered = []cephv1.Node{
	{
		Name: "node-1",
		Config: map[string]string{
			"metadataDevice": "vdd",
		},
		Selection: cephv1.Selection{
			DevicePathFilter: "/dev/vd[fe]",
		},
	},
	{
		Name: "node-2",
		Selection: cephv1.Selection{
			DeviceFilter: "vdb",
		},
	},
}
