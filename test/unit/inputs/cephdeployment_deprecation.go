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
	"k8s.io/apimachinery/pkg/runtime"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

var CephDeploymentDeprecated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		OldDashboardEnabled: &[]bool{true}[0],
		OldDataDirHostPath:  "/var/lib/custom-path",
		OldHealthCheck: &cephlcmv1alpha1.CephClusterHealthCheckSpec{
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
		OldHyperConverge: &cephlcmv1alpha1.CephDeploymentHyperConverge{
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
				"mds": v1.ResourceRequirements{
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("100m"),
						v1.ResourceMemory: resource.MustParse("156Mi"),
					},
				},
				"rgw": {
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
							Key:      "test.kubernetes.io/testkey-mgr",
							Effect:   "Schedule",
							Operator: "Exists",
						},
					},
				},
				"mon": {
					Rules: []v1.Toleration{
						{
							Key:      "test.kubernetes.io/testkey-mon",
							Effect:   "Schedule",
							Operator: "Exists",
						},
					},
				},
				"osd": {
					Rules: []v1.Toleration{
						{
							Key:      "test.kubernetes.io/testkey-osd",
							Effect:   "Schedule",
							Operator: "Exists",
						},
					},
				},
				"mds": {
					Rules: []v1.Toleration{
						{
							Key:      "test.kubernetes.io/testkey-mds",
							Effect:   "Schedule",
							Operator: "Exists",
						},
					},
				},
				"rgw": {
					Rules: []v1.Toleration{
						{
							Key:      "test.kubernetes.io/testkey-rgw",
							Effect:   "Schedule",
							Operator: "Exists",
						},
					},
				},
			},
		},
		OldMgr: &cephlcmv1alpha1.Mgr{
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
		OldNetwork: &cephlcmv1alpha1.CephNetworkSpec{
			ClusterNet: "127.0.0.0/16",
			PublicNet:  "192.168.0.0/16",
		},
		OldPools: []cephlcmv1alpha1.CephPoolOld{
			{
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
						Size:            3,
						TargetSizeRatio: 0.1,
					},
					Mirroring: &cephlcmv1alpha1.CephPoolMirrorSpec{
						Mode: "peer",
					},
				},
			},
			{
				Name: "pool2",
				Role: "custom",
				CephPoolSpec: cephlcmv1alpha1.CephPoolSpec{
					DeviceClass:   "hdd",
					FailureDomain: "host",
					ErasureCoded: &cephlcmv1alpha1.CephPoolErasureCodedSpec{
						CodingChunks: 1,
						DataChunks:   2,
						Algorithm:    "custom",
					},
					Parameters: map[string]string{
						"custom-pool-param": "custom-pool-value",
					},
					EnableCrushUpdates: &[]bool{true}[0],
				},
			},
		},
		SharedFilesystem: &cephlcmv1alpha1.CephSharedFilesystem{
			OldCephFS: []cephlcmv1alpha1.CephFS{
				{
					Name: "test-cephfs",
					MetadataPool: cephlcmv1alpha1.CephPoolSpec{
						DeviceClass: "hdd",
						Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
							Size: 3,
						},
					},
					PreserveFilesystemOnDelete: false,
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
						{
							Name: "second-pool-name",
							CephPoolSpec: cephlcmv1alpha1.CephPoolSpec{
								DeviceClass: "hdd",
								ErasureCoded: &cephlcmv1alpha1.CephPoolErasureCodedSpec{
									CodingChunks: 1,
									DataChunks:   2,
								},
							},
						},
					},
					MetadataServer: cephlcmv1alpha1.CephMetadataServer{
						ActiveCount:   1,
						ActiveStandby: true,
						HealthCheck: &cephlcmv1alpha1.CephMdsHealthCheck{
							LivenessProbe: &cephv1.ProbeSpec{Disabled: true},
							StartupProbe:  &cephv1.ProbeSpec{Disabled: true},
						},
					},
				},
			},
		},
		ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
			OldRgw: &cephlcmv1alpha1.CephRGW{
				Name: "rgw-store",
				ObjectUsers: []cephlcmv1alpha1.CephRGWUser{
					{
						Name:         "custom-user",
						DisplayName:  "custom-user-for-app",
						Capabilities: &cephv1.ObjectUserCapSpec{User: "read"},
						Quotas:       &cephv1.ObjectUserQuotaSpec{MaxBuckets: &[]int{1}[0]},
					},
					{
						Name: "base-user",
					},
				},
				Buckets:               []string{"bucket-1", "bucket-2"},
				PreservePoolsOnDelete: false,
				DataPool: &cephlcmv1alpha1.CephPoolSpec{
					DeviceClass: "hdd",
					ErasureCoded: &cephlcmv1alpha1.CephPoolErasureCodedSpec{
						CodingChunks: 1,
						DataChunks:   2,
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
				SSLCert: &cephlcmv1alpha1.CephDeploymentCert{
					Cacert:  "fake",
					TLSCert: "fake",
					TLSKey:  "fake",
				},
			},
		},
		Nodes: CephNodesOk,
	},
}

var CephDeploymentMultisiteDeprecated = func() cephlcmv1alpha1.CephDeployment {
	cd := BaseCephDeployment.DeepCopy()
	cd.Spec.ObjectStorage = &cephlcmv1alpha1.CephObjectStorage{
		OldMultiSite: &cephlcmv1alpha1.CephMultiSite{
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
							CodingChunks: 1,
							DataChunks:   2,
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
		OldRgw: &cephlcmv1alpha1.CephRGW{
			Name: "rgw-store",
			Gateway: cephlcmv1alpha1.CephRGWGateway{
				Instances:  2,
				Port:       80,
				SecurePort: 8443,
			},
			Zone: &cephv1.ZoneSpec{
				Name: "zone1",
			},
		},
	}
	return *cd
}()

var CephDeploymentSpecClusterJSON = `{"dashboard":{"enabled":true},"dataDirHostPath":"/var/lib/custom-path","healthCheck":{"daemonHealth":{"status":{"disabled":true},"mon":{"timeout":"60s"},"osd":{"timeout":"60s"}},"livenessProbe":{"osd":{"probe":{"timeoutSeconds":10,"failureThreshold":10}}},"startupProbe":{"osd":{"probe":{"timeoutSeconds":5,"failureThreshold":5}}}},"mgr":{"modules":[{"name":"balancer","enabled":true,"settings":{"balancerMode":"upmap"}},{"name":"fake","enabled":true}]},"network":{"addressRanges":{"cluster":["127.0.0.0/16"],"public":["192.168.0.0/16"]}},"placement":{"all":{"tolerations":[{"key":"test.kubernetes.io/testkey","operator":"Exists","effect":"Schedule"}]},"mgr":{"tolerations":[{"key":"test.kubernetes.io/testkey-mgr","operator":"Exists","effect":"Schedule"}]},"mon":{"tolerations":[{"key":"test.kubernetes.io/testkey-mon","operator":"Exists","effect":"Schedule"}]},"osd":{"tolerations":[{"key":"test.kubernetes.io/testkey-osd","operator":"Exists","effect":"Schedule"}]}},"resources":{"osd-nvme":{"limits":{"cpu":"100m","memory":"156Mi"},"requests":{"cpu":"10m","memory":"28Mi"}}}}`
var CephPoolSpec1MigratedJSON = `{"replicated":{"size":3,"targetSizeRatio":0.1},"failureDomain":"host","crushRoot":"default","deviceClass":"hdd","mirroring":{"mode":"peer"}}`
var CephPoolSpec2MigratedJSON = `{"failureDomain":"host","deviceClass":"hdd","erasureCoded":{"codingChunks":1,"dataChunks":2,"algorithm":"custom"},"parameters":{"custom-pool-param":"custom-pool-value"},"enableCrushUpdates":true}`
var CephFsSpecMigratedJSON = `{"dataPools":[{"name":"some-pool-name","replicated":{"size":3},"deviceClass":"hdd"},{"name":"second-pool-name","deviceClass":"hdd","erasureCoded":{"codingChunks":1,"dataChunks":2}}],"metadataPool":{"replicated":{"size":3},"deviceClass":"hdd"},"metadataServer":{"activeCount":1,"activeStandby":true,"livenessProbe":{"disabled":true},"placement":{"tolerations":[{"key":"test.kubernetes.io/testkey-mds","operator":"Exists","effect":"Schedule"}]},"resources":{"limits":{"cpu":"100m","memory":"156Mi"}},"startupProbe":{"disabled":true}},"preserveFilesystemOnDelete":false}`
var CephDeploymentRgwJSON = `{"dataPool":{"deviceClass":"hdd","erasureCoded":{"codingChunks":1,"dataChunks":2}},"gateway":{"instances":2,"placement":{"tolerations":[{"key":"test.kubernetes.io/testkey-rgw","operator":"Exists","effect":"Schedule"}]},"port":80,"resources":{"requests":{"cpu":"10m","memory":"28Mi"}},"securePort":8443,"sslCertificateRef":"rgw-ssl-certificate"},"metadataPool":{"replicated":{"size":3},"deviceClass":"hdd"},"preservePoolsOnDelete":false}`

var CephDeploymentMigrated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Cluster: &cephlcmv1alpha1.CephCluster{
			RawExtension: runtime.RawExtension{
				Raw: []byte(CephDeploymentSpecClusterJSON),
			},
		},
		BlockStorage: &cephlcmv1alpha1.CephBlockStorage{
			Pools: []cephlcmv1alpha1.CephPool{
				{
					Name: "pool1",
					Role: "fake",
					StorageClassOpts: cephlcmv1alpha1.CephStorageClassSpec{
						Default: true,
					},
					PoolSpec: runtime.RawExtension{
						Raw: []byte(CephPoolSpec1MigratedJSON),
					},
				},
				{
					Name: "pool2",
					Role: "custom",
					PoolSpec: runtime.RawExtension{
						Raw: []byte(CephPoolSpec2MigratedJSON),
					},
				},
			},
		},
		SharedFilesystem: &cephlcmv1alpha1.CephSharedFilesystem{
			Filesystems: []cephlcmv1alpha1.CephFilesystem{
				{
					Name: "test-cephfs",
					FsSpec: runtime.RawExtension{
						Raw: []byte(CephFsSpecMigratedJSON),
					},
				},
			},
		},
		ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
			Rgws: []cephlcmv1alpha1.CephObjectStore{
				{
					Name: "rgw-store",
					Spec: runtime.RawExtension{
						Raw: []byte(CephDeploymentRgwJSON),
					},
				},
			},
			Users: []cephlcmv1alpha1.CephObjectStoreUser{
				{
					Name: "custom-user",
					Spec: runtime.RawExtension{
						Raw: []byte(`{"capabilities":{"user":"read"},"displayName":"custom-user-for-app","quotas":{"maxBuckets":1},"store":"rgw-store"}`),
					},
				},
				{
					Name: "base-user",
					Spec: runtime.RawExtension{
						Raw: []byte(`{"store":"rgw-store"}`),
					},
				},
			},
		},
		Nodes: CephNodesOk,
	},
}

var CephDeploymentZoneJSON = `{"dataPool":{"failureDomain":"host","deviceClass":"hdd","erasureCoded":{"codingChunks":1,"dataChunks":2}},"metadataPool":{"replicated":{"size":3},"failureDomain":"host","deviceClass":"hdd"},"zoneGroup":"zonegroup1"}`
var CephDeploymentRgwMultisiteJSON = `{"gateway":{"instances":2,"port":80,"securePort":8443},"zone":{"name":"zone1"}}`

var CephDeploymentMultisiteMigrated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Cluster: BaseCephDeployment.Spec.Cluster.DeepCopy(),
		ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
			Rgws: []cephlcmv1alpha1.CephObjectStore{
				{
					Name: "rgw-store",
					Spec: runtime.RawExtension{
						Raw: []byte(CephDeploymentRgwMultisiteJSON),
					},
				},
			},
			Realms: []cephlcmv1alpha1.CephObjectRealm{
				{
					Name: "realm1",
					Spec: runtime.RawExtension{
						Raw: []byte(`{"defaultRealm":false}`),
					},
				},
			},
			Zonegroups: []cephlcmv1alpha1.CephObjectZonegroup{
				{
					Name: "zonegroup1",
					Spec: runtime.RawExtension{
						Raw: []byte(`{"realm":"realm1"}`),
					},
				},
			},
			Zones: []cephlcmv1alpha1.CephObjectZone{
				{
					Name: "zone1",
					Spec: runtime.RawExtension{
						Raw: []byte(CephDeploymentZoneJSON),
					},
				},
			},
		},
		Nodes: CephNodesOk,
	},
}

var CephDeploymentMultusDeprecated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		OldNetwork: &cephlcmv1alpha1.CephNetworkSpec{
			Provider:   "multus",
			ClusterNet: "127.0.0.0/16",
			PublicNet:  "192.168.0.0/16",
			Selector: map[cephv1.CephNetworkType]string{
				cephv1.CephNetworkPublic:  "192.168.0.0/16",
				cephv1.CephNetworkCluster: "127.0.0.0/16",
			},
		},
	},
}

var CephDeploymentSpecClusterMultusJSON = `{"network":{"addressRanges":{"cluster":["127.0.0.0/16"],"public":["192.168.0.0/16"]},"provider":"multus","selectors":{"cluster":"127.0.0.0/16","public":"192.168.0.0/16"}}}`

var CephDeploymentMultusMigrated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Cluster: &cephlcmv1alpha1.CephCluster{
			RawExtension: runtime.RawExtension{
				Raw: []byte(CephDeploymentSpecClusterMultusJSON),
			},
		},
	},
}

var CephDeployExternalDeprecated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		OldNetwork: &cephlcmv1alpha1.CephNetworkSpec{
			ClusterNet: "127.0.0.0/32",
			PublicNet:  "127.0.0.0/32",
		},
		OldExternal: &[]bool{true}[0],
		ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
			OldRgw: &cephlcmv1alpha1.CephRGW{
				Name: "external-rgw",
				Gateway: cephlcmv1alpha1.CephRGWGateway{
					Port:       8080,
					SecurePort: 8443,
					ExternalRgwEndpoint: &cephv1.EndpointAddress{
						IP:       "127.0.0.1",
						Hostname: "fake-1",
					},
				},
			},
		},
	},
}

var CephDeploymentSpecClusterExternalJSON = `{"external":{"enable":true}}`

var CephDeployExternalMigrated = cephlcmv1alpha1.CephDeployment{
	ObjectMeta: LcmObjectMeta,
	Spec: cephlcmv1alpha1.CephDeploymentSpec{
		Cluster: &cephlcmv1alpha1.CephCluster{
			RawExtension: runtime.RawExtension{
				Raw: []byte(CephDeploymentSpecClusterExternalJSON),
			},
		},
		ObjectStorage: &cephlcmv1alpha1.CephObjectStorage{
			Rgws: []cephlcmv1alpha1.CephObjectStore{
				{
					Name: "external-rgw",
					Spec: runtime.RawExtension{
						Raw: []byte(`{"gateway":{"externalRgwEndpoints":[{"ip":"127.0.0.1","hostname":"fake-1"}],"port":8080,"securePort":8443}}`),
					},
				},
			},
		},
	},
}
