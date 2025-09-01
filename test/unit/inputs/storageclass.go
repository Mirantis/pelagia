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
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var BaseStorageClassDefault = storagev1.StorageClass{
	ObjectMeta: metav1.ObjectMeta{
		Name: "pool1-hdd",
		Labels: map[string]string{
			"rook-ceph-storage-class": "true",
		},
		Annotations: map[string]string{"storageclass.kubernetes.io/is-default-class": "true"},
	},
	Provisioner: "rook-ceph.rbd.csi.ceph.com",
	Parameters: map[string]string{
		"clusterID":     "rook-ceph",
		"pool":          "pool1-hdd",
		"imageFormat":   "2",
		"imageFeatures": "layering",
		"csi.storage.k8s.io/provisioner-secret-name":      "rook-csi-rbd-provisioner",
		"csi.storage.k8s.io/provisioner-secret-namespace": "rook-ceph",
		"csi.storage.k8s.io/node-stage-secret-name":       "rook-csi-rbd-node",
		"csi.storage.k8s.io/node-stage-secret-namespace":  "rook-ceph",
	},
}

var ExternalStorageClassDefault = storagev1.StorageClass{
	ObjectMeta: metav1.ObjectMeta{
		Name: "pool1-hdd",
		Labels: map[string]string{
			"rook-ceph-storage-class": "true",
		},
		Annotations: map[string]string{"storageclass.kubernetes.io/is-default-class": "true"},
	},
	Provisioner:          "rook-ceph.rbd.csi.ceph.com",
	AllowVolumeExpansion: &TrueVarForPointer,
	Parameters: map[string]string{
		"clusterID":     "rook-ceph",
		"pool":          "pool1-hdd",
		"imageFormat":   "2",
		"imageFeatures": "layering",
		"csi.storage.k8s.io/provisioner-secret-name":            "rook-csi-rbd-provisioner",
		"csi.storage.k8s.io/provisioner-secret-namespace":       "rook-ceph",
		"csi.storage.k8s.io/node-stage-secret-name":             "rook-csi-rbd-node",
		"csi.storage.k8s.io/node-stage-secret-namespace":        "rook-ceph",
		"csi.storage.k8s.io/controller-expand-secret-name":      "rook-csi-rbd-provisioner",
		"csi.storage.k8s.io/controller-expand-secret-namespace": "rook-ceph",
		"csi.storage.k8s.io/fstype":                             "ext4",
	},
}

func GetNamedStorageClass(pool string, external bool) *storagev1.StorageClass {
	var sc *storagev1.StorageClass
	if external {
		sc = ExternalStorageClassDefault.DeepCopy()
	} else {
		sc = BaseStorageClassDefault.DeepCopy()
	}
	sc.Name = pool
	sc.Parameters["pool"] = pool
	sc.Annotations["storageclass.kubernetes.io/is-default-class"] = "false"
	return sc
}

var TrueVarForPointer = true
var DeleteReclaimPolicyForPointer = corev1.PersistentVolumeReclaimDelete

var CephFSStorageClass = storagev1.StorageClass{
	ObjectMeta: metav1.ObjectMeta{
		Name: "test-cephfs-some-pool-name",
		Labels: map[string]string{
			"rook-ceph-storage-class":                     "true",
			"rook-ceph-storage-class-keep-on-spec-remove": "false",
		},
	},
	Provisioner: "rook-ceph.cephfs.csi.ceph.com",
	Parameters: map[string]string{
		"clusterID": "rook-ceph",
		"pool":      "test-cephfs-some-pool-name",
		"fsName":    "test-cephfs",
		"csi.storage.k8s.io/provisioner-secret-name":            "rook-csi-cephfs-provisioner",
		"csi.storage.k8s.io/provisioner-secret-namespace":       "rook-ceph",
		"csi.storage.k8s.io/node-stage-secret-name":             "rook-csi-cephfs-node",
		"csi.storage.k8s.io/node-stage-secret-namespace":        "rook-ceph",
		"csi.storage.k8s.io/controller-expand-secret-name":      "rook-csi-cephfs-provisioner",
		"csi.storage.k8s.io/controller-expand-secret-namespace": "rook-ceph",
	},
	AllowVolumeExpansion: &TrueVarForPointer,
	ReclaimPolicy:        &DeleteReclaimPolicyForPointer,
}

var RgwStorageClass = storagev1.StorageClass{
	ObjectMeta: metav1.ObjectMeta{
		Name: "rgw-storage-class",
	},
	Provisioner: "rook-ceph.ceph.rook.io/bucket",
	Parameters: map[string]string{
		"objectStoreName":      "rgw-store",
		"objectStoreNamespace": "rook-ceph",
		"region":               "rgw-store",
	},
}

var StorageClassesListEmpty = storagev1.StorageClassList{
	Items: []storagev1.StorageClass{},
}

var StorageClassesList = storagev1.StorageClassList{
	Items: []storagev1.StorageClass{BaseStorageClassDefault},
}

var PersistentVolumeClaimListEmpty = corev1.PersistentVolumeClaimList{
	Items: []corev1.PersistentVolumeClaim{},
}

var PersistentVolumeListEmpty = corev1.PersistentVolumeList{
	Items: []corev1.PersistentVolume{},
}

var PersistentVolumeClaimList = corev1.PersistentVolumeClaimList{
	Items: []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pvc"},
			Spec: corev1.PersistentVolumeClaimSpec{
				StorageClassName: &StorageClassesList.Items[0].Name,
			},
			Status: corev1.PersistentVolumeClaimStatus{
				Phase: corev1.ClaimPending,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pvc-2"},
			Spec: corev1.PersistentVolumeClaimSpec{
				StorageClassName: &StorageClassesList.Items[0].Name,
			},
			Status: corev1.PersistentVolumeClaimStatus{
				Phase: corev1.ClaimBound,
			},
		},
	},
}

var PersistentVolumeList = corev1.PersistentVolumeList{
	Items: []corev1.PersistentVolume{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec: corev1.PersistentVolumeSpec{
				StorageClassName: "pool1-hdd",
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumePending,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "test-2"},
			Spec: corev1.PersistentVolumeSpec{
				StorageClassName: "pool1-hdd",
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeBound,
			},
		},
	},
}

var CephVolumeAttachment = storagev1.VolumeAttachment{
	ObjectMeta: metav1.ObjectMeta{
		Name: "ceph-volumeattachment",
	},
	Spec: storagev1.VolumeAttachmentSpec{
		Attacher: "rook-ceph.rbd.csi.ceph.com",
		NodeName: "node-1",
	},
}

var NonCephVolumeAttachment = storagev1.VolumeAttachment{
	ObjectMeta: metav1.ObjectMeta{
		Name: "non-ceph-volumeattachment",
	},
	Spec: storagev1.VolumeAttachmentSpec{
		Attacher: "another-attacher",
		NodeName: "node-1",
	},
}
