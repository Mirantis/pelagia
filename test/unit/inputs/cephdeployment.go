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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

var BaseCephDeployment = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		DashboardEnabled: false,
		Network: cephlcmv1alpha1.CephNetworkSpec{
			HostNetwork: true,
			ClusterNet:  "127.0.0.0/16",
			PublicNet:   "192.168.0.0/16",
		},
		Nodes: CephNodesOk,
	},
}

var CephDeploymentObjectsRefs = []v1.ObjectReference{
	{
		APIVersion: "lcm.mirantis.com/v1alpha1",
		Kind:       "CephDeploymentHealth",
		Name:       "cephcluster",
		Namespace:  "lcm-namespace",
	},
	{
		APIVersion: "lcm.mirantis.com/v1alpha1",
		Kind:       "CephDeploymentSecret",
		Name:       "cephcluster",
		Namespace:  "lcm-namespace",
	},
	{
		APIVersion: "lcm.mirantis.com/v1alpha1",
		Kind:       "CephDeploymentMaintenance",
		Name:       "cephcluster",
		Namespace:  "lcm-namespace",
	},
}

func GetUpdatedClusterVersionCephDeploy(cephDpl *cephlcmv1alpha1.CephDeployment, clusterVersion string) *cephlcmv1alpha1.CephDeployment {
	cephDpl.Status.ClusterVersion = clusterVersion
	return cephDpl
}

var BaseCephDeploymentDelete = func() cephlcmv1alpha1.CephDeployment {
	cd := BaseCephDeployment.DeepCopy()
	cd.Finalizers = []string{"cephdeployment.lcm.mirantis.com/finalizer"}
	cd.DeletionTimestamp = &metav1.Time{Time: time.Date(2021, 8, 15, 14, 30, 45, 0, time.Local)}
	return *cd
}()

var BaseCephDeploymentDeleting = func() cephlcmv1alpha1.CephDeployment {
	cd := BaseCephDeploymentDelete.DeepCopy()
	cd.Status = cephlcmv1alpha1.CephDeploymentStatus{
		Phase:   cephlcmv1alpha1.PhaseDeleting,
		Message: "Ceph cluster deletion is in progress",
	}
	return *cd
}()

var BaseCephDeploymentMultus = func() cephlcmv1alpha1.CephDeployment {
	cd := BaseCephDeployment.DeepCopy()
	cd.Spec.Network = cephlcmv1alpha1.CephNetworkSpec{
		Provider: "multus",
		Selector: map[cephv1.CephNetworkType]string{
			cephv1.CephNetworkPublic:  "192.168.0.0/16",
			cephv1.CephNetworkCluster: "127.0.0.0/16",
		},
	}
	return *cd
}()

var CephDeployRookConfigNoRuntimeNoOsd = func() cephlcmv1alpha1.CephDeployment {
	cd := BaseCephDeployment.DeepCopy()
	cd.Spec.RookConfig = map[string]string{"mon-max-pg-per-osd": "400"}
	return *cd
}()

var CephDeployRookConfigNoRuntimeOsdParams = func() cephlcmv1alpha1.CephDeployment {
	cd := BaseCephDeployment.DeepCopy()
	cd.Spec.RookConfig = map[string]string{
		"mon-max-pg-per-osd":       "400",
		"osd_max_backfills":        "64",
		"osd_recovery_max_active":  "16",
		"osd_recovery_op_priority": "3",
		"osd_recovery_sleep_hdd":   "0.000000",
	}
	return *cd
}()

var CephDeployObjectStorageCeph = func() cephlcmv1alpha1.CephDeployment {
	cd := BaseCephDeployment.DeepCopy()
	cd.Spec.ObjectStorage = &cephlcmv1alpha1.CephObjectStorage{
		Rgw: CephRgwBaseSpec,
	}
	return *cd
}()

var CephDeployEnsureRbdMirror = func() cephlcmv1alpha1.CephDeployment {
	cd := BaseCephDeployment.DeepCopy()
	cd.Finalizers = []string{"cephdeployment.lcm.mirantis.com/finalizer"}
	cd.Spec.RBDMirror = &cephlcmv1alpha1.CephRBDMirrorSpec{
		Count: 1,
		Peers: []cephlcmv1alpha1.CephRBDMirrorSecret{
			{
				Site:  "mirror1",
				Token: "fake-token",
				Pools: []string{"pool-1", "pool-2"},
			},
		},
	}
	cd.Spec.Pools = []cephlcmv1alpha1.CephPool{CephDeployPoolReplicated}
	return *cd
}()

var CephDeployEnsureRolesCrush = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Nodes: []cephlcmv1alpha1.CephDeploymentNode{
			{
				Node: cephv1.Node{
					Name: "node1",
					Selection: cephv1.Selection{
						DevicePathFilter: "/dev/vd*",
					},
				},
				Roles: []string{"mon", "mgr", "rgw", "osd", "mds"},
				Crush: map[string]string{
					"region": "region1",
					"zone":   "zone1",
					"rack":   "rack1",
				},
			},
		},
	},
}

var CephDeployEnsureMonitorIP = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Nodes: []cephlcmv1alpha1.CephDeploymentNode{
			{
				Node: cephv1.Node{
					Name: "node1",
				},
				Roles:     []string{"mon", "mgr", "rgw", "osd", "mds"},
				MonitorIP: "127.0.0.1",
			},
		},
	},
}

var CephDeployWithWrongNodes = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: metav1.ObjectMeta{
		Namespace:  LcmObjectMeta.Namespace,
		Name:       LcmObjectMeta.Name,
		Generation: int64(10),
	},
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Nodes: []cephlcmv1alpha1.CephDeploymentNode{
			{
				Node: cephv1.Node{
					Name: "wrong-node-group",
					Selection: cephv1.Selection{
						Devices: []cephv1.Device{
							{
								Name:   "sda",
								Config: map[string]string{"osdsPerDevice": "2", "deviceClass": "ssd"},
							},
						},
					},
				},
				Roles:        make([]string, 0),
				Crush:        map[string]string{"rack": "A"},
				NodeGroup:    []string{"node-1-random-uuid", "node-2-random-uuid"},
				NodesByLabel: "test_label=test",
			},
		},
	},
}

var CephDeployNonMosk = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: LcmObjectMeta.Namespace,
		Name:      LcmObjectMeta.Name,
		Finalizers: []string{
			"cephdeployment.lcm.mirantis.com/finalizer",
		},
		Generation: int64(10),
	},
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Pools:   []cephlcmv1alpha1.CephPool{CephDeployPoolReplicated},
		Clients: []cephlcmv1alpha1.CephClient{CephDeployClientTest},
		Nodes:   CephNodesExtendedOk,
		Network: BaseCephDeployment.Spec.Network,
		ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
			Rgw: CephRgwSpecWithUsersBuckets,
		},
		SharedFilesystem: CephSharedFileSystemOk,
	},
	Status: cephlcmv1alpha1.CephDeploymentStatus{
		Validation: cephlcmv1alpha1.CephDeploymentValidation{
			Result:                  cephlcmv1alpha1.ValidationSucceed,
			LastValidatedGeneration: int64(10),
		},
		ObjectsRefs: CephDeploymentObjectsRefs,
	},
}

var CephDeployNonMoskWithIngress = func() cephlcmv1alpha1.CephDeployment {
	cd := CephDeployNonMosk.DeepCopy()
	cd.Spec.IngressConfig = &cephlcmv1alpha1.CephDeploymentIngressConfig{
		TLSConfig: &cephlcmv1alpha1.CephDeploymentIngressTLSConfig{
			Domain: "example.com",
		},
	}
	return *cd
}()

var CephDeployNonMoskForSecret = func() cephlcmv1alpha1.CephDeployment {
	cd := CephDeployNonMosk.DeepCopy()
	cd.Spec.ObjectStorage.Rgw.ObjectUsers = []cephlcmv1alpha1.CephRGWUser{{Name: "test-user"}}
	cd.Spec.ObjectStorage.Rgw.Buckets = []string{"test-bucket"}
	return *cd
}()

var CephDeployMosk = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: LcmObjectMeta.Namespace,
		Name:      LcmObjectMeta.Name,
		Finalizers: []string{
			"cephdeployment.lcm.mirantis.com/finalizer",
		},
	},
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Network: BaseCephDeployment.Spec.Network,
		Pools: []cephlcmv1alpha1.CephPool{
			CephDeployPoolReplicated,
			GetCephDeployPool("vms", "vms"),
			GetCephDeployPool("volumes", "volumes"),
			GetCephDeployPool("images", "images"),
			GetCephDeployPool("backup", "backup"),
		},
		Nodes: CephNodesExtendedOk,
		ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
			Rgw: CephRgwBaseSpec,
		},
		IngressConfig: &CephIngressConfig,
	},
	Status: cephlcmv1alpha1.CephDeploymentStatus{
		Validation: cephlcmv1alpha1.CephDeploymentValidation{
			Result:                  cephlcmv1alpha1.ValidationSucceed,
			LastValidatedGeneration: int64(0),
		},
		ObjectsRefs: CephDeploymentObjectsRefs,
	},
}

var CephDeployMoskWithCephFS = func() cephlcmv1alpha1.CephDeployment {
	cd := CephDeployMosk.DeepCopy()
	cd.Spec.SharedFilesystem = CephSharedFileSystemOk
	return *cd
}()

var CephDeployMoskWithoutIngress = func() cephlcmv1alpha1.CephDeployment {
	cd := CephDeployMosk.DeepCopy()
	cd.Spec.IngressConfig = nil
	return *cd
}()

var CephDeployMoskWithoutIngressRookConfigOverride = func() cephlcmv1alpha1.CephDeployment {
	cd := CephDeployMoskWithoutIngress.DeepCopy()
	cd.Spec.RookConfig = map[string]string{
		"cluster network":           "10.0.0.0/24",
		"public network":            "172.16.0.0/24",
		"rgw_trust_forwarded_https": "false",
		"rgw keystone admin user":   "override-user",
	}
	return *cd
}()

var CephDeployMoskWithoutIngressRookConfigOverrideBarbican = func() cephlcmv1alpha1.CephDeployment {
	cd := CephDeployMoskWithoutIngress.DeepCopy()
	cd.Spec.RookConfig = map[string]string{
		"mon_max_pg_per_osd":                  "400",
		"rgw enforce swift acls":              "false",
		"rgw_user_quota_bucket_sync_interval": "10",
		"rgw_dns_name":                        "rgw-store.ms2.wxlsd.com",
		"rgw keystone barbican user":          "override-user",
	}
	return *cd
}()

var CephDeployMoskWithoutRgw = func() cephlcmv1alpha1.CephDeployment {
	cd := CephDeployMoskWithoutIngress.DeepCopy()
	cd.Spec.ObjectStorage = nil
	return *cd
}()

var CephDeployExternal = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: LcmObjectMeta.Namespace,
		Name:      LcmObjectMeta.Name,
		Finalizers: []string{
			"cephdeployment.lcm.mirantis.com/finalizer",
		},
	},
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Network: cephlcmv1alpha1.CephNetworkSpec{
			ClusterNet: "127.0.0.0/32",
			PublicNet:  "127.0.0.0/32",
		},
		External: true,
		Pools:    []cephlcmv1alpha1.CephPool{CephDeployPoolReplicated},
	},
	Status: cephlcmv1alpha1.CephDeploymentStatus{
		Validation: cephlcmv1alpha1.CephDeploymentValidation{
			Result:                  cephlcmv1alpha1.ValidationSucceed,
			LastValidatedGeneration: int64(0),
		},
	},
}

var CephDeployExternalRgw = func() cephlcmv1alpha1.CephDeployment {
	cd := CephDeployExternal.DeepCopy()
	cd.Spec.ObjectStorage = &cephlcmv1alpha1.CephObjectStorage{
		Rgw: RgwExternalSslEnabled,
	}
	return *cd
}()

var CephDeployExternalCephFS = func() cephlcmv1alpha1.CephDeployment {
	cd := CephDeployExternal.DeepCopy()
	cd.Spec.SharedFilesystem = CephSharedFileSystemOk
	return *cd
}()

var CephDeployMultisiteMasterRgw = func() cephlcmv1alpha1.CephDeployment {
	cd := BaseCephDeployment.DeepCopy()
	cd.Spec.ObjectStorage = &cephlcmv1alpha1.CephObjectStorage{
		MultiSite: &cephlcmv1alpha1.CephMultiSite{
			Realms: []cephlcmv1alpha1.CephRGWRealm{
				{
					Name: "realm1",
				},
			},
			ZoneGroups: []cephlcmv1alpha1.CephRGWZoneGroup{
				{
					Name:  "zonegroup1",
					Realm: "realm1",
				},
			},
			Zones: []cephlcmv1alpha1.CephRGWZone{
				{
					Name:      "zone1",
					ZoneGroup: "zonegroup1",
					DataPool: cephlcmv1alpha1.CephPoolSpec{
						DeviceClass:   "hdd",
						FailureDomain: "host",
						ErasureCoded: &cephlcmv1alpha1.CephPoolErasureCodedSpec{
							CodingChunks: 2,
							DataChunks:   1,
						},
					},
					MetadataPool: cephlcmv1alpha1.CephPoolSpec{
						DeviceClass:   "hdd",
						FailureDomain: "host",
						Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
							Size: 3,
						},
					},
				},
			},
		},
		Rgw: cephlcmv1alpha1.CephRGW{
			Name:    "rgw-store",
			Gateway: CephRgwBaseSpec.Gateway,
			Zone: &cephv1.ZoneSpec{
				Name: "zone1",
			},
		},
	}
	return *cd
}()

var CephDeployMultisiteRgw = func() cephlcmv1alpha1.CephDeployment {
	cd := BaseCephDeployment.DeepCopy()
	cd.Spec.ObjectStorage = &cephlcmv1alpha1.CephObjectStorage{
		MultiSite: &cephlcmv1alpha1.CephMultiSite{
			Realms: []cephlcmv1alpha1.CephRGWRealm{
				{
					Name: "realm1",
					Pull: &cephlcmv1alpha1.CephRGWRealmPull{
						Endpoint:  "http://10.10.0.1",
						AccessKey: "fakekey",
						SecretKey: "fakesecret",
					},
				},
			},
			ZoneGroups: []cephlcmv1alpha1.CephRGWZoneGroup{
				{
					Name:  "zonegroup1",
					Realm: "realm1",
				},
			},
			Zones: []cephlcmv1alpha1.CephRGWZone{
				{
					Name:      "secondary-zone1",
					ZoneGroup: "zonegroup1",
					DataPool: cephlcmv1alpha1.CephPoolSpec{
						DeviceClass:   "hdd",
						CrushRoot:     "default",
						FailureDomain: "host",
						ErasureCoded: &cephlcmv1alpha1.CephPoolErasureCodedSpec{
							CodingChunks: 2,
							DataChunks:   1,
						},
					},
					MetadataPool: cephlcmv1alpha1.CephPoolSpec{
						DeviceClass:   "hdd",
						CrushRoot:     "default",
						FailureDomain: "host",
						Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
							Size: 3,
						},
					},
				},
			},
		},
		Rgw: cephlcmv1alpha1.CephRGW{
			Name:    "rgw-store",
			Gateway: CephRgwBaseSpec.Gateway,
			Zone: &cephv1.ZoneSpec{
				Name: "secondary-zone1",
			},
		},
	}
	return *cd
}()

var MultisiteRgwWithSyncDaemon = func() cephlcmv1alpha1.CephDeployment {
	cd := CephDeployMultisiteRgw.DeepCopy()
	cd.Spec.ObjectStorage.Rgw.Gateway.SplitDaemonForMultisiteTrafficSync = true
	return *cd
}()

// spec fixtures

var HyperConvergeCephDeploy = &cephlcmv1alpha1.CephDeploymentHyperConverge{
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
}

var HyperConvergeForExtraSVC = &cephlcmv1alpha1.CephDeploymentHyperConverge{
	Tolerations: map[string]cephlcmv1alpha1.CephDeploymentToleration{
		"mds": {
			Rules: []v1.Toleration{
				{
					Key:      "test.kubernetes.io/testkey",
					Effect:   "Schedule",
					Operator: "Exists",
				},
			},
		},
		"rgw": {
			Rules: []v1.Toleration{
				{
					Key:      "rgw-toleration",
					Operator: "Exists",
				},
			},
		},
	},
	Resources: cephv1.ResourceSpec{
		"mds": v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceCPU: resource.MustParse("120m"),
			},
			Requests: v1.ResourceList{
				v1.ResourceCPU: resource.MustParse("10m"),
			},
		},
		"rgw": v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceCPU: resource.MustParse("50m"),
			},
			Requests: v1.ResourceList{
				v1.ResourceCPU: resource.MustParse("20m"),
			},
		},
	},
}

var CephDeployClientTest = cephlcmv1alpha1.CephClient{
	ClientSpec: cephlcmv1alpha1.ClientSpec{
		Name: "test",
		Caps: map[string]string{
			"osd": "custom-caps",
		},
	},
}

var CephDeployClientCinder = cephlcmv1alpha1.CephClient{
	ClientSpec: cephlcmv1alpha1.ClientSpec{
		Name: "cinder",
		Caps: map[string]string{
			"mon": "allow profile rbd",
			"osd": "profile rbd pool=volumes-hdd, profile rbd-read-only pool=images-hdd, profile rbd pool=backup-hdd",
		},
	},
}

var CephDeployClientGlance = cephlcmv1alpha1.CephClient{
	ClientSpec: cephlcmv1alpha1.ClientSpec{
		Name: "glance",
		Caps: map[string]string{
			"mon": "allow profile rbd",
			"osd": "profile rbd pool=images-hdd",
		},
	},
}

var CephDeployClientNova = cephlcmv1alpha1.CephClient{
	ClientSpec: cephlcmv1alpha1.ClientSpec{
		Name: "nova",
		Caps: map[string]string{
			"mon": "allow profile rbd",
			"osd": "profile rbd pool=vms-hdd, profile rbd pool=images-hdd, profile rbd pool=volumes-hdd",
		},
	},
}

var CephDeployClientManila = cephlcmv1alpha1.CephClient{
	ClientSpec: cephlcmv1alpha1.ClientSpec{
		Name: "manila",
		Caps: map[string]string{
			"mds": "allow rw",
			"mgr": "allow rw",
			"osd": "allow rw tag cephfs *=*",
			"mon": `allow r, allow command "auth del", allow command "auth caps", allow command "auth get", allow command "auth get-or-create"`,
		},
	},
}

var CephDeployPoolReplicated = cephlcmv1alpha1.CephPool{
	Name: "pool1",
	Role: "fake",
	StorageClassOpts: cephlcmv1alpha1.CephStorageClassSpec{
		Default: true,
	},
	CephPoolSpec: cephlcmv1alpha1.CephPoolSpec{
		DeviceClass:   "hdd",
		CrushRoot:     "default",
		FailureDomain: "host",
		Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
			Size: 3,
		},
	},
}

var CephDeployPoolErasureCoded = cephlcmv1alpha1.CephPool{
	Name: "pool1",
	Role: "fake",
	CephPoolSpec: cephlcmv1alpha1.CephPoolSpec{
		DeviceClass:   "hdd",
		CrushRoot:     "default",
		FailureDomain: "host",
		ErasureCoded: &cephlcmv1alpha1.CephPoolErasureCodedSpec{
			CodingChunks: 1,
			DataChunks:   2,
			Algorithm:    "fake",
		},
	},
}

var CephDeployPoolMirroring = cephlcmv1alpha1.CephPool{
	Name: "pool1",
	Role: "fake",
	CephPoolSpec: cephlcmv1alpha1.CephPoolSpec{
		DeviceClass:   "hdd",
		CrushRoot:     "default",
		FailureDomain: "host",
		Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
			Size: 3,
		},
		Mirroring: &cephlcmv1alpha1.CephPoolMirrorSpec{
			Mode: "pool",
		},
	},
}

func GetCephDeployPool(name string, role string) cephlcmv1alpha1.CephPool {
	return cephlcmv1alpha1.CephPool{
		Name: name,
		Role: role,
		CephPoolSpec: cephlcmv1alpha1.CephPoolSpec{
			DeviceClass:   "hdd",
			CrushRoot:     "default",
			FailureDomain: "host",
			Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
				Size: 3,
			},
		},
	}
}

var CephNodesOk = []cephlcmv1alpha1.CephDeploymentNode{
	{
		Node: cephv1.Node{
			Name: "node-1",
			Selection: cephv1.Selection{
				Devices: []cephv1.Device{
					{
						Name:   "sda",
						Config: map[string]string{"deviceClass": "hdd"},
					},
				},
			},
		},
		Roles: []string{"mon", "mgr"},
	},
	{
		Node: cephv1.Node{
			Name: "node-2",
			Selection: cephv1.Selection{
				Devices: []cephv1.Device{
					{
						Name:   "sda",
						Config: map[string]string{"osdsPerDevice": "1", "deviceClass": "hdd"},
					},
				},
			},
		},
		Roles: []string{"mon"},
	},
	{
		Node: cephv1.Node{
			Name:   "node-3",
			Config: map[string]string{"osdsPerDevice": "2"},
			Selection: cephv1.Selection{
				Devices: []cephv1.Device{
					{
						Name:   "sda",
						Config: map[string]string{"deviceClass": "hdd"},
					},
				},
			},
		},
		Roles: []string{"mon"},
	},
}

var CephNodesExtendedOk = []cephlcmv1alpha1.CephDeploymentNode{
	{
		Node: cephv1.Node{
			Name: "node-1",
			Selection: cephv1.Selection{
				Devices: []cephv1.Device{
					{
						Name:   "sda",
						Config: map[string]string{"deviceClass": "hdd"},
					},
					{
						Name:   "sdb",
						Config: map[string]string{"deviceClass": "hdd"},
					},
				},
			},
		},
		Roles: []string{"mon", "mgr", "mds"},
	},
	{
		Node: cephv1.Node{
			Name: "node-2",
			Selection: cephv1.Selection{
				Devices: []cephv1.Device{
					{
						Name:   "sda",
						Config: map[string]string{"osdsPerDevice": "1", "deviceClass": "hdd"},
					},
					{
						Name:   "sdb",
						Config: map[string]string{"osdsPerDevice": "2", "deviceClass": "hdd"},
					},
					{
						Name:   "sdc",
						Config: map[string]string{"metadataDevice": "sde", "deviceClass": "hdd"},
					},
				},
			},
		},
		Roles: []string{"mon"},
	},
	{
		Node: cephv1.Node{
			Name:   "node-3",
			Config: map[string]string{"osdsPerDevice": "2"},
			Selection: cephv1.Selection{
				Devices: []cephv1.Device{
					{
						Name:   "sda",
						Config: map[string]string{"deviceClass": "hdd"},
					},
				},
			},
		},
		Roles: []string{"mon"},
	},
}

var CephNodesExtendedInvalid = []cephlcmv1alpha1.CephDeploymentNode{
	{
		Node: cephv1.Node{
			Name: "node-1",
			Selection: cephv1.Selection{
				Devices: []cephv1.Device{
					{
						Name:   "sda",
						Config: map[string]string{"deviceClass": "hdd"},
					},
					{
						Name:   "sdb",
						Config: map[string]string{},
					},
				},
			},
			Config: map[string]string{"osdsPerDevice": "3.5"},
		},
		Crush: map[string]string{"datecenter": "fr"},
		Roles: []string{"mon"},
	},
	{
		Node: cephv1.Node{
			Name: "node-2",
			Selection: cephv1.Selection{
				Devices: []cephv1.Device{
					{
						Name:   "sda",
						Config: map[string]string{"osdsPerDevice": "1", "deviceClass": "hdd"},
					},
					{
						Name:   "sdb",
						Config: map[string]string{"osdsPerDevice": "2", "deviceClass": "hdd"},
					},
					{
						Name:   "sdc",
						Config: map[string]string{"metadataDevice": "sde", "deviceClass": "some-custom-class"},
					},
				},
			},
		},
		Roles: []string{"mon"},
	},
	{
		Node: cephv1.Node{
			Name:   "node-3",
			Config: map[string]string{"osdsPerDevice": "2"},
			Selection: cephv1.Selection{
				Devices: []cephv1.Device{
					{
						Name:   "sda",
						Config: map[string]string{"deviceClass": "unknown-class", "osdsPerDevice": "3.5"},
					},
				},
			},
		},
		Roles: []string{"mon"},
	},
	{
		Node: cephv1.Node{
			Name:   "node-4",
			Config: map[string]string{"osdsPerDevice": "2"},
			Selection: cephv1.Selection{
				Devices: []cephv1.Device{
					{
						Name:   "sda",
						Config: map[string]string{"deviceClass": "hdd"},
					},
				},
			},
		},
		Roles: []string{"mon"},
	},
	{
		Node: cephv1.Node{
			Name: "node-5",
			Selection: cephv1.Selection{
				UseAllDevices: &[]bool{true}[0],
			},
		},
	},
	{
		Node: cephv1.Node{
			Name: "node-6",
			Selection: cephv1.Selection{
				DeviceFilter: "sda",
			},
		},
	},
}

var CephFSNewOk = cephlcmv1alpha1.CephFS{
	Name: "test-cephfs",
	MetadataPool: cephlcmv1alpha1.CephPoolSpec{
		DeviceClass: "hdd",
		Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
			Size: 3,
		},
	},
	DataPools: []cephlcmv1alpha1.CephFSPool{
		{
			Name: "some-pool-name",
			CephPoolSpec: cephlcmv1alpha1.CephPoolSpec{
				DeviceClass: "hdd",
				Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
					Size: 3,
				},
			},
		},
	},
	MetadataServer: cephlcmv1alpha1.CephMetadataServer{
		ActiveCount:   1,
		ActiveStandby: true,
	},
}

var CephFSOkWithResources = func() cephlcmv1alpha1.CephFS {
	fs := CephFSNewOk.DeepCopy()
	fs.MetadataServer.Resources = &v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("100m"),
			v1.ResourceMemory: resource.MustParse("156Mi"),
		},
		Requests: v1.ResourceList{
			v1.ResourceMemory: resource.MustParse("28Mi"),
			v1.ResourceCPU:    resource.MustParse("10m"),
		},
	}
	return *fs
}()

var CephSharedFileSystemOk = &cephlcmv1alpha1.CephSharedFilesystem{
	CephFS: []cephlcmv1alpha1.CephFS{
		CephFSNewOk,
	},
}

var CephSharedFileSystemMultiple = &cephlcmv1alpha1.CephSharedFilesystem{
	CephFS: []cephlcmv1alpha1.CephFS{
		CephFSNewOk,
		func() cephlcmv1alpha1.CephFS {
			newCephFS := CephFSNewOk.DeepCopy()
			newCephFS.Name = "second-test-cephfs"
			return *newCephFS
		}(),
	},
}

var CephIngressConfig = cephlcmv1alpha1.CephDeploymentIngressConfig{
	TLSConfig: &cephlcmv1alpha1.CephDeploymentIngressTLSConfig{
		Domain: "test",
		TLSCerts: &cephlcmv1alpha1.CephDeploymentCert{
			Cacert:  "spec-cacert",
			TLSCert: "spec-tlscert",
			TLSKey:  "spec-tlskey",
		},
	},
	Annotations: map[string]string{
		"fake": "fake",
	},
	ControllerClassName: "fake-class-name",
}

var CephRgwBaseSpec = cephlcmv1alpha1.CephRGW{
	Name:                  "rgw-store",
	PreservePoolsOnDelete: false,
	DataPool: &cephlcmv1alpha1.CephPoolSpec{
		DeviceClass: "hdd",
		ErasureCoded: &cephlcmv1alpha1.CephPoolErasureCodedSpec{
			CodingChunks: 2,
			DataChunks:   1,
		},
	},
	MetadataPool: &cephlcmv1alpha1.CephPoolSpec{
		DeviceClass: "hdd",
		Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
			Size: 3,
		},
	},
	Gateway: cephlcmv1alpha1.CephRGWGateway{
		Instances:  2,
		Port:       80,
		SecurePort: 8443,
	},
}

var CephRgwSpecWithUsersBuckets = func() cephlcmv1alpha1.CephRGW {
	rgw := CephRgwBaseSpec.DeepCopy()
	rgw.ObjectUsers = []cephlcmv1alpha1.CephRGWUser{
		{Name: "fake-user-1"}, {Name: "fake-user-2"},
	}
	rgw.Buckets = []string{"fake-bucket-1", "fake-bucket-2"}
	return *rgw
}()

var RgwExternalSslEnabled = cephlcmv1alpha1.CephRGW{
	Name: "rgw-store",
	Gateway: cephlcmv1alpha1.CephRGWGateway{
		Instances:  2,
		Port:       80,
		SecurePort: 8443,
		ExternalRgwEndpoint: &cephv1.EndpointAddress{
			IP:       "127.0.0.1",
			Hostname: "fake-1",
		},
	},
}
