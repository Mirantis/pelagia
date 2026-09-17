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
	vsapi "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var VSCListEmpty = &vsapi.VolumeSnapshotClassList{Items: []vsapi.VolumeSnapshotClass{}}
var VSCListPresent = &vsapi.VolumeSnapshotClassList{
	Items: []vsapi.VolumeSnapshotClass{CsiCephFSPluginVSC, CsiRBDPluginVSC},
}

var CsiRBDPluginVSC = vsapi.VolumeSnapshotClass{
	ObjectMeta: metav1.ObjectMeta{
		Name: "csi-rbdplugin-snapclass",
		Labels: map[string]string{
			"app.kubernetes.io/created-by": "pelagia-deployment-controller",
			"app.kubernetes.io/managed-by": "pelagia-deployment-controller",
			"app.kubernetes.io/part-of":    "ceph.pelagia.lcm",
		},
		ResourceVersion: "1",
	},
	Driver: "rook-ceph.rbd.csi.ceph.com",
	Parameters: map[string]string{
		"clusterID": "rook-ceph",
		"csi.storage.k8s.io/snapshotter-secret-name":      "rook-csi-rbd-provisioner",
		"csi.storage.k8s.io/snapshotter-secret-namespace": "rook-ceph",
	},
	DeletionPolicy: vsapi.VolumeSnapshotContentDelete,
}

var CsiCephFSPluginVSC = vsapi.VolumeSnapshotClass{
	ObjectMeta: metav1.ObjectMeta{
		Name: "csi-cephfsplugin-snapclass",
		Labels: map[string]string{
			"app.kubernetes.io/created-by": "pelagia-deployment-controller",
			"app.kubernetes.io/managed-by": "pelagia-deployment-controller",
			"app.kubernetes.io/part-of":    "ceph.pelagia.lcm",
		},
		ResourceVersion: "1",
	},
	Driver: "rook-ceph.cephfs.csi.ceph.com",
	Parameters: map[string]string{
		"clusterID": "rook-ceph",
		"csi.storage.k8s.io/snapshotter-secret-name":      "rook-csi-cephfs-provisioner",
		"csi.storage.k8s.io/snapshotter-secret-namespace": "rook-ceph",
	},
	DeletionPolicy: vsapi.VolumeSnapshotContentDelete,
}
