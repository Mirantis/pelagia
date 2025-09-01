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
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var CephRBDMirror = cephv1.CephRBDMirror{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "cephcluster",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.RBDMirroringSpec{
		Count: 1,
		Peers: cephv1.MirroringPeerSpec{
			SecretNames: []string{"rbd-mirror-token-mirror1-pool-1", "rbd-mirror-token-mirror1-pool-2"},
		},
	},
}

func CephRBDMirrorWithStatus(mirror cephv1.CephRBDMirror, phase string) *cephv1.CephRBDMirror {
	rbdMirror := mirror.DeepCopy()
	rbdMirror.Status = &cephv1.Status{
		Phase: phase,
	}
	return rbdMirror
}

var CephRBDMirrorUpdatedReady = cephv1.CephRBDMirror{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "cephcluster",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.RBDMirroringSpec{
		Count: 2,
		Peers: cephv1.MirroringPeerSpec{
			SecretNames: []string{"rbd-mirror-token-mirror1-pool-1", "rbd-mirror-token-mirror1-pool-2"},
		},
	},
	Status: &cephv1.Status{Phase: "Ready"},
}

var CephRBDMirrorsEmpty = cephv1.CephRBDMirrorList{
	Items: []cephv1.CephRBDMirror{},
}

var CephRBDMirrorsList = cephv1.CephRBDMirrorList{
	Items: []cephv1.CephRBDMirror{CephRBDMirror},
}

var CephRBDMirrorsListReady = cephv1.CephRBDMirrorList{
	Items: []cephv1.CephRBDMirror{
		func() cephv1.CephRBDMirror {
			mirror := CephRBDMirrorUpdatedReady.DeepCopy()
			mirror.Spec.Count = 1
			return *mirror
		}(),
	},
}
