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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

var BaseCephDeployment = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Cluster: &cephlcmv1alpha1.CephCluster{
			RawExtension: runtime.RawExtension{
				Raw: ConvertStructToRaw(
					cephv1.ClusterSpec{
						Network: cephv1.NetworkSpec{
							AddressRanges: &cephv1.AddressRangesSpec{
								Public:  []cephv1.CIDR{cephv1.CIDR("192.168.0.0/16")},
								Cluster: []cephv1.CIDR{cephv1.CIDR("127.0.0.0/16")},
							},
						},
					},
				),
			},
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
	cd.Spec.Cluster.Raw = ConvertStructToRaw(
		cephv1.ClusterSpec{
			Network: cephv1.NetworkSpec{
				Provider: "multus",
				AddressRanges: &cephv1.AddressRangesSpec{
					Public:  []cephv1.CIDR{cephv1.CIDR("192.168.0.0/16")},
					Cluster: []cephv1.CIDR{cephv1.CIDR("127.0.0.0/16")},
				},
				Selectors: map[cephv1.CephNetworkType]string{
					cephv1.CephNetworkPublic:  "192.168.0.0/16",
					cephv1.CephNetworkCluster: "127.0.0.0/16",
				},
			},
		},
	)
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
		Rgws: []cephlcmv1alpha1.CephObjectStore{CephRgwBaseSpec},
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
	cd.Spec.BlockStorage = &cephlcmv1alpha1.CephBlockStorage{
		Pools: []cephlcmv1alpha1.CephPool{CephDeployPoolReplicated},
	}
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
		Cluster: BaseCephDeployment.Spec.Cluster.DeepCopy(),
		BlockStorage: &cephlcmv1alpha1.CephBlockStorage{
			Pools: []cephlcmv1alpha1.CephPool{CephDeployPoolReplicated},
		},
		Clients: []cephlcmv1alpha1.CephClient{CephDeployClientTest},
		Nodes:   CephNodesExtendedOk,
		ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
			Rgws: []cephlcmv1alpha1.CephObjectStore{CephRgwBaseSpec},
			Users: []cephlcmv1alpha1.CephObjectStoreUser{
				{
					Name: "fake-user-1",
					Spec: runtime.RawExtension{
						Raw: ConvertStructToRaw(
							cephv1.ObjectStoreUserSpec{
								Store: "rgw-store",
							},
						),
					},
				},
				{
					Name: "fake-user-2",
					Spec: runtime.RawExtension{
						Raw: ConvertStructToRaw(
							cephv1.ObjectStoreUserSpec{
								Store: "rgw-store",
							},
						),
					},
				},
			},
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
	cd.Spec.ObjectStorage.Rgws[0].ServedByIngress = true
	return *cd
}()

var CephDeployNonMoskForSecret = func() cephlcmv1alpha1.CephDeployment {
	cd := CephDeployNonMosk.DeepCopy()
	cd.Spec.ObjectStorage.Users = []cephlcmv1alpha1.CephObjectStoreUser{{Name: "test-user"}}
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
		Cluster: BaseCephDeployment.Spec.Cluster.DeepCopy(),
		BlockStorage: &cephlcmv1alpha1.CephBlockStorage{
			Pools: []cephlcmv1alpha1.CephPool{
				CephDeployPoolReplicated,
				GetCephDeployPool("vms", "vms"),
				GetCephDeployPool("volumes", "volumes"),
				GetCephDeployPool("images", "images"),
				GetCephDeployPool("backup", "backup"),
			},
		},
		Nodes: CephNodesExtendedOk,
		ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
			Rgws: []cephlcmv1alpha1.CephObjectStore{
				func() cephlcmv1alpha1.CephObjectStore {
					rgw := CephRgwBaseSpec.DeepCopy()
					rgw.UsedByRockoon = true
					return *rgw
				}(),
			},
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
		Cluster: &cephlcmv1alpha1.CephCluster{
			RawExtension: runtime.RawExtension{
				Raw: ConvertStructToRaw(
					cephv1.ClusterSpec{
						External: cephv1.ExternalSpec{Enable: true},
					},
				),
			},
		},
		BlockStorage: &cephlcmv1alpha1.CephBlockStorage{
			Pools: []cephlcmv1alpha1.CephPool{CephDeployPoolReplicated},
		},
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
		Rgws: []cephlcmv1alpha1.CephObjectStore{CephRgwExternal},
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
		Realms: []cephlcmv1alpha1.CephObjectRealm{
			{
				Name: "realm1",
			},
		},
		Zonegroups: []cephlcmv1alpha1.CephObjectZonegroup{
			{
				Name: "zonegroup1",
				Spec: runtime.RawExtension{
					Raw: []byte(`{"realm": "realm1"}`),
				},
			},
		},
		Zones: []cephlcmv1alpha1.CephObjectZone{
			{
				Name: "zone1",
				Spec: runtime.RawExtension{
					Raw: ConvertStructToRaw(
						cephv1.ObjectZoneSpec{
							ZoneGroup: "zonegroup1",
							MetadataPool: cephv1.PoolSpec{
								DeviceClass:   "hdd",
								FailureDomain: "host",
								Replicated:    cephv1.ReplicatedSpec{Size: 3},
							},
							DataPool: cephv1.PoolSpec{
								DeviceClass:   "hdd",
								FailureDomain: "host",
								ErasureCoded: cephv1.ErasureCodedSpec{
									CodingChunks: 2,
									DataChunks:   1,
								},
							},
						},
					),
				},
			},
		},
		Rgws: []cephlcmv1alpha1.CephObjectStore{
			{
				Name: "rgw-store",
				Spec: runtime.RawExtension{
					Raw: ConvertStructToRaw(
						cephv1.ObjectStoreSpec{
							Gateway: cephv1.GatewaySpec{
								Instances:  2,
								Port:       80,
								SecurePort: 8443,
							},
							Zone: cephv1.ZoneSpec{
								Name: "zone1",
							},
						},
					),
				},
			},
		},
	}
	return *cd
}()

var CephDeployMultisiteRgw = func() cephlcmv1alpha1.CephDeployment {
	cd := BaseCephDeployment.DeepCopy()
	cd.Spec.ObjectStorage = &cephlcmv1alpha1.CephObjectStorage{
		Realms: []cephlcmv1alpha1.CephObjectRealm{
			{
				Name: "realm1",
				Spec: runtime.RawExtension{
					Raw: []byte(`{"pull": {"endpoint": "http://10.10.0.1"}}`),
				},
			},
		},
		Zonegroups: []cephlcmv1alpha1.CephObjectZonegroup{
			{
				Name: "zonegroup1",
				Spec: runtime.RawExtension{
					Raw: []byte(`{"realm": "realm1"}`),
				},
			},
		},
		Zones: []cephlcmv1alpha1.CephObjectZone{
			{
				Name: "secondary-zone1",
				Spec: runtime.RawExtension{
					Raw: ConvertStructToRaw(
						cephv1.ObjectZoneSpec{
							ZoneGroup: "zonegroup1",
							MetadataPool: cephv1.PoolSpec{
								DeviceClass:   "hdd",
								CrushRoot:     "default",
								FailureDomain: "host",
								Replicated:    cephv1.ReplicatedSpec{Size: 3},
							},
							DataPool: cephv1.PoolSpec{
								DeviceClass:   "hdd",
								CrushRoot:     "default",
								FailureDomain: "host",
								ErasureCoded: cephv1.ErasureCodedSpec{
									CodingChunks: 1,
									DataChunks:   2,
								},
							},
						},
					),
				},
			},
		},
		Rgws: []cephlcmv1alpha1.CephObjectStore{
			{
				Name: "rgw-store",
				Spec: runtime.RawExtension{
					Raw: ConvertStructToRaw(
						cephv1.ObjectStoreSpec{
							Gateway: cephv1.GatewaySpec{
								Instances:  2,
								Port:       80,
								SecurePort: 8443,
							},
							Zone: cephv1.ZoneSpec{
								Name: "secondary-zone1",
							},
						},
					),
				},
			},
		},
	}
	return *cd
}()

var MultisiteRgwWithSyncDaemon = func() cephlcmv1alpha1.CephDeployment {
	cd := CephDeployMultisiteRgw.DeepCopy()
	cd.Spec.ObjectStorage.Rgws = []cephlcmv1alpha1.CephObjectStore{
		{
			Name: "rgw-store",
			Spec: runtime.RawExtension{
				Raw: ConvertStructToRaw(cephv1.ObjectStoreSpec{
					Gateway: cephv1.GatewaySpec{
						Instances:                   2,
						Port:                        80,
						SecurePort:                  8443,
						DisableMultisiteSyncTraffic: true,
					},
					Zone: cephv1.ZoneSpec{
						Name: "secondary-zone1",
					},
				}),
			},
		},
		{
			Name:             "rgw-store-sync",
			AuxiliaryService: true,
			Spec: runtime.RawExtension{
				Raw: ConvertStructToRaw(cephv1.ObjectStoreSpec{
					Gateway: cephv1.GatewaySpec{
						Port:                        8340,
						Instances:                   1,
						CaBundleRef:                 "multisite-rgw-secret",
						DisableMultisiteSyncTraffic: false,
					},
					Zone: cephv1.ZoneSpec{
						Name: "secondary-zone1",
					},
				}),
			},
		},
	}
	return *cd
}()

// spec fixtures

var CephDeployClientTest = cephlcmv1alpha1.CephClient{
	RawExtension: runtime.RawExtension{
		Raw: ConvertStructToRaw(
			cephv1.ClientSpec{
				Name: "test",
				Caps: map[string]string{
					"osd": "custom-caps",
				},
			},
		),
	},
}

var CephDeployClientCinder = cephlcmv1alpha1.CephClient{
	RawExtension: runtime.RawExtension{
		Raw: ConvertStructToRaw(
			cephv1.ClientSpec{
				Name: "cinder",
				Caps: map[string]string{
					"mon": "allow profile rbd",
					"osd": "profile rbd pool=volumes-hdd, profile rbd-read-only pool=images-hdd, profile rbd pool=backup-hdd",
				},
			},
		),
	},
}

var CephDeployClientGlance = cephlcmv1alpha1.CephClient{
	RawExtension: runtime.RawExtension{
		Raw: ConvertStructToRaw(
			cephv1.ClientSpec{
				Name: "glance",
				Caps: map[string]string{
					"mon": "allow profile rbd",
					"osd": "profile rbd pool=images-hdd",
				},
			},
		),
	},
}

var CephDeployClientNova = cephlcmv1alpha1.CephClient{
	RawExtension: runtime.RawExtension{
		Raw: ConvertStructToRaw(
			cephv1.ClientSpec{
				Name: "nova",
				Caps: map[string]string{
					"mon": "allow profile rbd",
					"osd": "profile rbd pool=vms-hdd, profile rbd pool=images-hdd, profile rbd pool=volumes-hdd",
				},
			},
		),
	},
}

var CephDeployClientManila = cephlcmv1alpha1.CephClient{
	RawExtension: runtime.RawExtension{
		Raw: ConvertStructToRaw(
			cephv1.ClientSpec{
				Name: "manila",
				Caps: map[string]string{
					"mds": "allow rw",
					"mgr": "allow rw",
					"osd": "allow rw tag cephfs *=*",
					"mon": `allow r, allow command "auth del", allow command "auth caps", allow command "auth get", allow command "auth get-or-create"`,
				},
			},
		),
	},
}

var CephDeployPoolReplicated = cephlcmv1alpha1.CephPool{
	Name: "pool1",
	Role: "fake",
	StorageClassOpts: cephlcmv1alpha1.CephStorageClassSpec{
		Default: true,
	},
	PoolSpec: runtime.RawExtension{
		Raw: ConvertStructToRaw(
			cephv1.PoolSpec{
				DeviceClass:   "hdd",
				CrushRoot:     "default",
				FailureDomain: "host",
				Replicated: cephv1.ReplicatedSpec{
					Size: 3,
				},
			},
		),
	},
}

var CephDeployPoolErasureCoded = cephlcmv1alpha1.CephPool{
	Name: "pool1",
	Role: "fake",
	PoolSpec: runtime.RawExtension{
		Raw: ConvertStructToRaw(
			cephv1.PoolSpec{
				DeviceClass:   "hdd",
				CrushRoot:     "default",
				FailureDomain: "host",
				ErasureCoded: cephv1.ErasureCodedSpec{
					CodingChunks: 1,
					DataChunks:   2,
					Algorithm:    "fake",
				},
			},
		),
	},
}

var CephDeployPoolMirroring = cephlcmv1alpha1.CephPool{
	Name: "pool1",
	Role: "fake",
	PoolSpec: runtime.RawExtension{
		Raw: ConvertStructToRaw(
			cephv1.PoolSpec{
				DeviceClass:   "hdd",
				CrushRoot:     "default",
				FailureDomain: "host",
				Replicated: cephv1.ReplicatedSpec{
					Size: 3,
				},
				Mirroring: cephv1.MirroringSpec{
					Enabled: true,
					Mode:    "pool",
				},
			},
		),
	},
}

func GetCephDeployPool(name string, role string) cephlcmv1alpha1.CephPool {
	return cephlcmv1alpha1.CephPool{
		Name: name,
		Role: role,
		PoolSpec: runtime.RawExtension{
			Raw: ConvertStructToRaw(
				cephv1.PoolSpec{
					DeviceClass:   "hdd",
					CrushRoot:     "default",
					FailureDomain: "host",
					Replicated: cephv1.ReplicatedSpec{
						Size: 3,
					},
				},
			),
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

var CephFSNewOk = cephlcmv1alpha1.CephFilesystem{
	Name: "test-cephfs",
	FsSpec: runtime.RawExtension{
		Raw: ConvertStructToRaw(
			cephv1.FilesystemSpec{
				MetadataPool: cephv1.NamedPoolSpec{
					PoolSpec: cephv1.PoolSpec{
						DeviceClass: "hdd",
						Replicated:  cephv1.ReplicatedSpec{Size: 3},
					},
				},
				DataPools: []cephv1.NamedPoolSpec{
					{
						Name: "some-pool-name",
						PoolSpec: cephv1.PoolSpec{
							DeviceClass: "hdd",
							Replicated:  cephv1.ReplicatedSpec{Size: 3},
						},
					},
				},
				MetadataServer: cephv1.MetadataServerSpec{
					ActiveCount:   1,
					ActiveStandby: true,
				},
			},
		),
	},
}

var CephSharedFileSystemOk = &cephlcmv1alpha1.CephSharedFilesystem{
	Filesystems: []cephlcmv1alpha1.CephFilesystem{CephFSNewOk},
}

var CephSharedFileSystemMultiple = &cephlcmv1alpha1.CephSharedFilesystem{
	Filesystems: []cephlcmv1alpha1.CephFilesystem{
		CephFSNewOk,
		func() cephlcmv1alpha1.CephFilesystem {
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

var CephRgwBaseSpec = cephlcmv1alpha1.CephObjectStore{
	Name: "rgw-store",
	Spec: runtime.RawExtension{
		Raw: ConvertStructToRaw(
			cephv1.ObjectStoreSpec{
				PreservePoolsOnDelete: false,
				DataPool: cephv1.PoolSpec{
					DeviceClass: "hdd",
					ErasureCoded: cephv1.ErasureCodedSpec{
						CodingChunks: 1,
						DataChunks:   2,
					},
				},
				MetadataPool: cephv1.PoolSpec{
					DeviceClass: "hdd",
					Replicated:  cephv1.ReplicatedSpec{Size: 3},
				},
				Gateway: cephv1.GatewaySpec{
					Instances:  2,
					Port:       80,
					SecurePort: 8443,
				},
			},
		),
	},
}

var CephRgwExternal = cephlcmv1alpha1.CephObjectStore{
	Name: "rgw-store",
	Spec: runtime.RawExtension{
		Raw: ConvertStructToRaw(
			cephv1.ObjectStoreSpec{
				Gateway: cephv1.GatewaySpec{
					Port: 8080,
					ExternalRgwEndpoints: []cephv1.EndpointAddress{
						{
							IP:       "127.0.0.1",
							Hostname: "fake-1",
						},
					},
				},
			},
		),
	},
}
