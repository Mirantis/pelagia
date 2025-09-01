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
	bktv1alpha1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//
// CEPHOBJECTSTORE SECTION
//

var CephObjectStoreListEmpty = cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{}}
var CephObjectStoreListExternal = cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{CephObjectStoreExternalReady}}

var CephObjectStoreReady = cephv1.CephObjectStore{
	ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "rgw-store"},
	Spec: cephv1.ObjectStoreSpec{
		Gateway: cephv1.GatewaySpec{
			Instances:   2,
			CaBundleRef: "rgw-ssl-certificate",
		},
	},
	Status: &cephv1.ObjectStoreStatus{Phase: cephv1.ConditionReady},
}

var CephObjectStoreSyncReady = cephv1.CephObjectStore{
	ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "rgw-store-sync"},
	Spec: cephv1.ObjectStoreSpec{
		Gateway: cephv1.GatewaySpec{
			Instances:                   1,
			DisableMultisiteSyncTraffic: false,
		},
	},
	Status: &cephv1.ObjectStoreStatus{Phase: cephv1.ConditionReady},
}

var CephObjectStoreExternalReady = cephv1.CephObjectStore{
	ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "rgw-store-external"},
	Spec: cephv1.ObjectStoreSpec{
		Gateway: cephv1.GatewaySpec{
			ExternalRgwEndpoints: []cephv1.EndpointAddress{
				{
					IP:       "127.0.0.1",
					Hostname: "external-rgw-endpoint",
				},
			},
		},
	},
	Status: &cephv1.ObjectStoreStatus{
		Phase: cephv1.ConditionReady,
		Info: map[string]string{
			"endpont":        "http://127.0.0.1:80",
			"secureEndpoint": "https://127.0.0.1:8443",
		},
	},
}

var CephObjectStoreListReady = cephv1.CephObjectStoreList{
	Items: []cephv1.CephObjectStore{CephObjectStoreReady},
}

var CephObjectStoresMultisiteSyncDaemonPhaseReady = cephv1.CephObjectStoreList{
	Items: []cephv1.CephObjectStore{
		func() cephv1.CephObjectStore {
			rgw := CephObjectStoreReady.DeepCopy()
			rgw.Spec.Gateway.DisableMultisiteSyncTraffic = true
			return *rgw
		}(),
		CephObjectStoreSyncReady,
	},
}

var CephObjectStoresMultisiteSyncDaemonPhaseNotReady = cephv1.CephObjectStoreList{
	Items: []cephv1.CephObjectStore{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "rgw-store"},
			Spec: cephv1.ObjectStoreSpec{
				Gateway: cephv1.GatewaySpec{
					DisableMultisiteSyncTraffic: true,
					Instances:                   2,
				},
			},
			Status: &cephv1.ObjectStoreStatus{Phase: cephv1.ConditionFailure},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "rgw-store-sync"},
			Spec:       CephObjectStoreSyncReady.Spec,
		},
	},
}

var CephObjectStoreBase = &cephv1.CephObjectStore{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rgw-store",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.ObjectStoreSpec{
		DefaultRealm:          true,
		PreservePoolsOnDelete: false,
		DataPool: cephv1.PoolSpec{
			DeviceClass: "hdd",
			ErasureCoded: cephv1.ErasureCodedSpec{
				CodingChunks: 2,
				DataChunks:   1,
			},
		},
		MetadataPool: cephv1.PoolSpec{
			DeviceClass: "hdd",
			Replicated: cephv1.ReplicatedSpec{
				Size: 3,
			},
		},
		Gateway: cephv1.GatewaySpec{
			Annotations: map[string]string{
				"cephdeployment.lcm.mirantis.com/config-global-updated":                 "some-time",
				"cephdeployment.lcm.mirantis.com/ssl-cert-generated":                    "some-time",
				"cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-updated": "some-time",
			},
			SSLCertificateRef: "rgw-ssl-certificate",
			CaBundleRef:       "rgw-ssl-certificate",
			Instances:         2,
			Port:              80,
			SecurePort:        8443,
			Placement: cephv1.Placement{
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
				Tolerations: []corev1.Toleration{
					{
						Key:      "ceph_role_mon",
						Operator: "Exists",
					},
				},
			},
		},
	},
}

var CephObjectStoreWithZone = &cephv1.CephObjectStore{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rgw-store",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.ObjectStoreSpec{
		PreservePoolsOnDelete: false,
		Zone: cephv1.ZoneSpec{
			Name: "zone1",
		},
		Gateway: cephv1.GatewaySpec{
			Annotations: map[string]string{
				"cephdeployment.lcm.mirantis.com/config-global-updated":                 "some-time",
				"cephdeployment.lcm.mirantis.com/ssl-cert-generated":                    "some-time",
				"cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-updated": "some-time",
			},
			SSLCertificateRef: "rgw-ssl-certificate",
			CaBundleRef:       "rgw-ssl-certificate",
			Instances:         2,
			Port:              80,
			SecurePort:        8443,
			Placement: cephv1.Placement{
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
				Tolerations: []corev1.Toleration{
					{
						Key:      "ceph_role_mon",
						Operator: "Exists",
					},
				},
			},
		},
	},
}

var CephObjectStoreWithSyncDaemon = func() *cephv1.CephObjectStore {
	store := CephObjectStoreWithZone.DeepCopy()
	store.Name = "rgw-store-sync"
	store.Spec.Zone.Name = "secondary-zone1"
	delete(store.Spec.Gateway.Annotations, "cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.a-updated")
	store.Spec.Gateway.Annotations["cephdeployment.lcm.mirantis.com/config-client.rgw.rgw.store.sync.a-updated"] = "some-time-sync"
	store.Spec.Gateway.DisableMultisiteSyncTraffic = false
	store.Spec.Gateway.Instances = 1
	store.Spec.Gateway.SecurePort = 0
	store.Spec.Gateway.Port = 8380
	store.Spec.Gateway.SSLCertificateRef = ""
	return store
}()

var CephObjectStoreExternal = &cephv1.CephObjectStore{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rgw-store",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.ObjectStoreSpec{
		Gateway: cephv1.GatewaySpec{
			Annotations: map[string]string{
				"cephdeployment.lcm.mirantis.com/ssl-cert-generated": "some-time",
			},
			Port:              80,
			SecurePort:        8443,
			SSLCertificateRef: "rgw-ssl-certificate",
			CaBundleRef:       "rgw-ssl-certificate",
			ExternalRgwEndpoints: []cephv1.EndpointAddress{
				{
					IP:       "127.0.0.1",
					Hostname: "fake-1",
				},
			},
		},
	},
}

var CephObjectStoreBaseReady = func() *cephv1.CephObjectStore {
	store := CephObjectStoreBase.DeepCopy()
	store.Status = &cephv1.ObjectStoreStatus{
		Phase: "Ready",
	}
	return store
}()

var CephObjectStoreBaseListReady = cephv1.CephObjectStoreList{
	Items: []cephv1.CephObjectStore{*CephObjectStoreBaseReady},
}

var CephObjectStoreMultisiteSyncList = cephv1.CephObjectStoreList{
	Items: []cephv1.CephObjectStore{
		func() cephv1.CephObjectStore {
			rgw := CephObjectStoreWithZone.DeepCopy()
			rgw.Spec.Zone.Name = "secondary-zone1"
			rgw.Spec.Gateway.DisableMultisiteSyncTraffic = true
			return *rgw
		}(),
		*CephObjectStoreWithSyncDaemon,
	},
}

//
// CEPHOBJECTSTOREUSER SECTION
//

var CephObjectStoreUserListEmpty = cephv1.CephObjectStoreUserList{Items: []cephv1.CephObjectStoreUser{}}

var CephObjectStoreUserListReady = cephv1.CephObjectStoreUserList{
	Items: []cephv1.CephObjectStoreUser{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "rgw-user-1"},
			Status:     &cephv1.ObjectStoreUserStatus{Phase: "Ready"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "rgw-user-2"},
			Status:     &cephv1.ObjectStoreUserStatus{Phase: "Ready"},
		},
	},
}

var CephObjectStoreUserListNotReady = cephv1.CephObjectStoreUserList{
	Items: []cephv1.CephObjectStoreUser{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "rgw-user-1"},
			Status:     &cephv1.ObjectStoreUserStatus{Phase: "Failed"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "rgw-user-2"},
		},
	},
}

var RgwUserBase = GetCephRgwUser("test-user", "rook-ceph", "rgw-store")

var RgwUserWithCapsAndQuotas = cephv1.CephObjectStoreUser{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-user",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.ObjectStoreUserSpec{
		Store:       "rgw-store",
		DisplayName: "test-user",
		Capabilities: &cephv1.ObjectUserCapSpec{
			User: "*",
		},
		Quotas: &cephv1.ObjectUserQuotaSpec{
			MaxBuckets: &[]int{1}[0],
		},
	},
}

var RgwCeilometerUser = cephv1.CephObjectStoreUser{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rgw-ceilometer",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.ObjectStoreUserSpec{
		Store:       "rgw-store",
		DisplayName: "rgw-ceilometer",
		Capabilities: &cephv1.ObjectUserCapSpec{
			User:     "read",
			Bucket:   "read",
			MetaData: "read",
			Usage:    "read",
		},
	},
}

var CephObjectStoreUserListMetrics = cephv1.CephObjectStoreUserList{
	Items: []cephv1.CephObjectStoreUser{*RgwUserWithStatus(RgwCeilometerUser, "Ready")},
}

func RgwUserWithStatus(user cephv1.CephObjectStoreUser, phase string) *cephv1.CephObjectStoreUser {
	rgwUser := user.DeepCopy()
	rgwUser.Status = &cephv1.ObjectStoreUserStatus{
		Phase: phase,
		Info: map[string]string{
			"secretName": "rgw-metrics-user-secret",
		},
	}
	return rgwUser
}

func GetCephRgwUser(name, namespace, rgwName string) cephv1.CephObjectStoreUser {
	return cephv1.CephObjectStoreUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cephv1.ObjectStoreUserSpec{
			Store:       rgwName,
			DisplayName: name,
		},
	}
}

var CephRgwUsersList = cephv1.CephObjectStoreUserList{
	Items: []cephv1.CephObjectStoreUser{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-user-1",
				Namespace: "rook-ceph",
			},
			Spec: cephv1.ObjectStoreUserSpec{
				Store:       "rgw-store",
				DisplayName: "fake-user-1",
			},
			Status: &cephv1.ObjectStoreUserStatus{
				Phase: "Ready",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-user-2",
				Namespace: "rook-ceph",
			},
			Spec: cephv1.ObjectStoreUserSpec{
				Store:       "rgw-store",
				DisplayName: "fake-user-2",
			},
			Status: &cephv1.ObjectStoreUserStatus{
				Phase: "Ready",
			},
		},
	},
}

//
// OBJECTBUCKETCLAIM SECTION
//

var ObjectBucketClaimListEmpty = bktv1alpha1.ObjectBucketClaimList{Items: []bktv1alpha1.ObjectBucketClaim{}}

var ObjectBucketClaimListReady = bktv1alpha1.ObjectBucketClaimList{
	Items: []bktv1alpha1.ObjectBucketClaim{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "bucket-1"},
			Status:     bktv1alpha1.ObjectBucketClaimStatus{Phase: bktv1alpha1.ObjectBucketClaimStatusPhaseBound},
		},
	},
}

var ObjectBucketClaimListNotReady = bktv1alpha1.ObjectBucketClaimList{
	Items: []bktv1alpha1.ObjectBucketClaim{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "bucket-1"},
			Status:     bktv1alpha1.ObjectBucketClaimStatus{Phase: bktv1alpha1.ObjectBucketClaimStatusPhaseFailed},
		},
	},
}

func GetSimpleBucket(name string) *bktv1alpha1.ObjectBucketClaim {
	return &bktv1alpha1.ObjectBucketClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "rook-ceph",
		},
		Spec: bktv1alpha1.ObjectBucketClaimSpec{
			GenerateBucketName: name,
			StorageClassName:   "rgw-storage-class",
		},
	}
}

var CephRgwBucketsList = bktv1alpha1.ObjectBucketClaimList{
	Items: []bktv1alpha1.ObjectBucketClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-bucket-1",
				Namespace: "rook-ceph",
			},
			Spec: bktv1alpha1.ObjectBucketClaimSpec{
				GenerateBucketName: "fake-bucket-1",
				StorageClassName:   "rgw-storage-class",
			},
			Status: bktv1alpha1.ObjectBucketClaimStatus{
				Phase: bktv1alpha1.ObjectBucketClaimStatusPhaseBound,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-bucket-2",
				Namespace: "rook-ceph",
			},
			Spec: bktv1alpha1.ObjectBucketClaimSpec{
				GenerateBucketName: "fake-bucket-2",
				StorageClassName:   "rgw-storage-class",
			},
			Status: bktv1alpha1.ObjectBucketClaimStatus{
				Phase: bktv1alpha1.ObjectBucketClaimStatusPhaseBound,
			},
		},
	},
}

//
// CEPHOBJECTREALM SECTION
//

var CephObjectRealmListEmpty = cephv1.CephObjectRealmList{Items: []cephv1.CephObjectRealm{}}

var CephObjectRealmListReady = cephv1.CephObjectRealmList{
	Items: []cephv1.CephObjectRealm{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "realm-1"},
			Status:     &cephv1.Status{Phase: "Ready"},
		},
	},
}

var CephObjectRealmListNotReady = cephv1.CephObjectRealmList{
	Items: []cephv1.CephObjectRealm{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "realm-1"},
			Status:     &cephv1.Status{Phase: "Failed"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "realm-2"},
		},
	},
}

var RgwMultisiteMasterRealm1 = cephv1.CephObjectRealm{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "realm1",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.ObjectRealmSpec{
		DefaultRealm: true,
	},
}

var RgwMultisiteMasterPullRealm1 = cephv1.CephObjectRealm{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "realm1",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.ObjectRealmSpec{
		DefaultRealm: true,
		Pull: cephv1.PullSpec{
			Endpoint: "http://10.10.0.1",
		},
	},
}

//
// CEPHOBJECTZONEGROUP SECTION
//

var CephObjectZoneGroupListEmpty = cephv1.CephObjectZoneGroupList{Items: []cephv1.CephObjectZoneGroup{}}

var CephObjectZoneGroupListReady = cephv1.CephObjectZoneGroupList{
	Items: []cephv1.CephObjectZoneGroup{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "zonegroup-1"},
			Status:     &cephv1.Status{Phase: "Ready"},
		},
	},
}

var CephObjectZoneGroupListNotReady = cephv1.CephObjectZoneGroupList{
	Items: []cephv1.CephObjectZoneGroup{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "zonegroup-1"},
			Status:     &cephv1.Status{Phase: "Failed"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "zonegroup-2"},
		},
	},
}

var RgwMultisiteMasterZoneGroup1 = cephv1.CephObjectZoneGroup{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "zonegroup1",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.ObjectZoneGroupSpec{
		Realm: "realm1",
	},
}

//
// CEPHOBJECTZONE SECTION
//

var CephObjectZoneListEmpty = cephv1.CephObjectZoneList{Items: []cephv1.CephObjectZone{}}

var CephObjectZoneListReady = cephv1.CephObjectZoneList{
	Items: []cephv1.CephObjectZone{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "zone-1"},
			Spec:       cephv1.ObjectZoneSpec{ZoneGroup: "zonegroup-1"},
			Status:     &cephv1.Status{Phase: "Ready"},
		},
	},
}

var CephObjectZoneListNotReady = cephv1.CephObjectZoneList{
	Items: []cephv1.CephObjectZone{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "zone-1"},
			Spec:       cephv1.ObjectZoneSpec{ZoneGroup: "zonegroup-1"},
			Status:     &cephv1.Status{Phase: "Failed"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "zone-2"},
			Spec:       cephv1.ObjectZoneSpec{ZoneGroup: "zonegroup-2"},
		},
	},
}

var RgwMultisiteMasterZone1 = cephv1.CephObjectZone{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "zone1",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.ObjectZoneSpec{
		ZoneGroup: "zonegroup1",
		DataPool: cephv1.PoolSpec{
			DeviceClass:   "hdd",
			FailureDomain: "host",
			ErasureCoded: cephv1.ErasureCodedSpec{
				CodingChunks: 2,
				DataChunks:   1,
			},
		},
		MetadataPool: cephv1.PoolSpec{
			DeviceClass:   "hdd",
			FailureDomain: "host",
			Replicated: cephv1.ReplicatedSpec{
				Size: 3,
			},
		},
	},
}

var RgwMultisiteSecondaryZone1 = cephv1.CephObjectZone{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "secondary-zone1",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.ObjectZoneSpec{
		ZoneGroup: "zonegroup1",
		DataPool: cephv1.PoolSpec{
			DeviceClass:   "hdd",
			FailureDomain: "host",
			ErasureCoded: cephv1.ErasureCodedSpec{
				CodingChunks: 2,
				DataChunks:   1,
			},
		},
		MetadataPool: cephv1.PoolSpec{
			DeviceClass:   "hdd",
			FailureDomain: "host",
			Replicated: cephv1.ReplicatedSpec{
				Size: 3,
			},
		},
	},
}
