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
	csiopapi "github.com/ceph/ceph-csi-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmcommon "github.com/Mirantis/pelagia/v3/pkg/common"
)

var CsiDriversEmpty = &csiopapi.DriverList{}
var CsiDriversRook = &csiopapi.DriverList{
	Items: []csiopapi.Driver{DriverCephFSDefault, DriverRBDDefault},
}

var OperatorConfigsEmpty = &csiopapi.OperatorConfigList{}
var OperatorConfigsRook = &csiopapi.OperatorConfigList{
	Items: []csiopapi.OperatorConfig{OperatorConfigDefault},
}

var CephConnectionsEmpty = &csiopapi.CephConnectionList{}
var CephConnectionsRook = &csiopapi.CephConnectionList{
	Items: []csiopapi.CephConnection{CephConnection},
}

var CephConnection = csiopapi.CephConnection{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rook-ceph",
		Namespace: "rook-ceph",
	},
}

var ClientProfilesEmpty = &csiopapi.ClientProfileList{}
var ClientProfilesRook = &csiopapi.ClientProfileList{
	Items: []csiopapi.ClientProfile{ClientProfile},
}

var ClientProfile = csiopapi.ClientProfile{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rook-ceph",
		Namespace: "rook-ceph",
	},
}

func GetCsiDriver(driverName, clusterName string) csiopapi.Driver {
	return csiopapi.Driver{
		ObjectMeta: metav1.ObjectMeta{
			Name:            driverName,
			Namespace:       "rook-ceph",
			ResourceVersion: "1",
		},
		Spec: csiopapi.DriverSpec{
			ClusterName: &clusterName,
		},
	}
}

func GetOperatorConfig(configName, clusterName string) *csiopapi.OperatorConfig {
	return &csiopapi.OperatorConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:            configName,
			Namespace:       "rook-ceph",
			ResourceVersion: "1",
		},
		Spec: csiopapi.OperatorConfigSpec{
			DriverSpecDefaults: &csiopapi.DriverSpec{
				ClusterName: &clusterName,
			},
		},
	}
}

var OperatorConfigDefault = csiopapi.OperatorConfig{
	ObjectMeta: metav1.ObjectMeta{
		Name:            "ceph-csi-operator-config",
		Namespace:       "rook-ceph",
		ResourceVersion: "1",
		Labels: map[string]string{
			"app.kubernetes.io/created-by": "pelagia-deployment-controller",
			"app.kubernetes.io/managed-by": "pelagia-deployment-controller",
			"app.kubernetes.io/part-of":    "ceph.pelagia.lcm",
		},
	},
	Spec: csiopapi.OperatorConfigSpec{
		DriverSpecDefaults: &csiopapi.DriverSpec{
			ImageSet:         &corev1.LocalObjectReference{Name: "rook-csi-operator-image-set-configmap"},
			ClusterName:      &LcmObjectMeta.Name,
			GenerateOMapInfo: lcmcommon.PtrTo(false),
			FsGroupPolicy:    storagev1.FileFSGroupPolicy,
			NodePlugin: &csiopapi.NodePluginSpec{
				PodCommonSpec: csiopapi.PodCommonSpec{
					PrioritylClassName: lcmcommon.PtrTo(""),
					Affinity:           &corev1.Affinity{},
				},
				Resources:              csiopapi.NodePluginResourcesSpec{},
				KubeletDirPath:         "/var/lib/kubelet",
				EnableSeLinuxHostMount: lcmcommon.PtrTo(true),
			},
			ControllerPlugin: &csiopapi.ControllerPluginSpec{
				PodCommonSpec: csiopapi.PodCommonSpec{
					PrioritylClassName: lcmcommon.PtrTo(""),
					Affinity:           &corev1.Affinity{},
				},
				Replicas:  lcmcommon.PtrTo(int32(2)),
				Resources: csiopapi.ControllerPluginResourcesSpec{},
			},
			DeployCsiAddons:  lcmcommon.PtrTo(false),
			CephFsClientType: csiopapi.KernelCephFsClient,
		},
	},
}

var DriverRBDDefault = csiopapi.Driver{
	ObjectMeta: metav1.ObjectMeta{
		Name:            "rook-ceph.rbd.csi.ceph.com",
		Namespace:       "rook-ceph",
		ResourceVersion: "1",
		Labels: map[string]string{
			"app.kubernetes.io/created-by": "pelagia-deployment-controller",
			"app.kubernetes.io/managed-by": "pelagia-deployment-controller",
			"app.kubernetes.io/part-of":    "ceph.pelagia.lcm",
		},
	},
	Spec: csiopapi.DriverSpec{},
}

var DriverCephFSDefault = csiopapi.Driver{
	ObjectMeta: metav1.ObjectMeta{
		Name:            "rook-ceph.cephfs.csi.ceph.com",
		Namespace:       "rook-ceph",
		ResourceVersion: "1",
		Labels: map[string]string{
			"app.kubernetes.io/created-by": "pelagia-deployment-controller",
			"app.kubernetes.io/managed-by": "pelagia-deployment-controller",
			"app.kubernetes.io/part-of":    "ceph.pelagia.lcm",
		},
	},
	Spec: csiopapi.DriverSpec{
		SnapshotPolicy: csiopapi.VolumeGroupSnapshotPolicy,
	},
}

var DriverNFSDefault = csiopapi.Driver{
	ObjectMeta: metav1.ObjectMeta{
		Name:            "rook-ceph.nfs.csi.ceph.com",
		Namespace:       "rook-ceph",
		ResourceVersion: "1",
		Labels: map[string]string{
			"app.kubernetes.io/created-by": "pelagia-deployment-controller",
			"app.kubernetes.io/managed-by": "pelagia-deployment-controller",
			"app.kubernetes.io/part-of":    "ceph.pelagia.lcm",
		},
	},
	Spec: csiopapi.DriverSpec{},
}
