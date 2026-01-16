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

var CephBlockPoolListEmpty = cephv1.CephBlockPoolList{Items: []cephv1.CephBlockPool{}}

var CephBlockPoolListReady = cephv1.CephBlockPoolList{
	Items: []cephv1.CephBlockPool{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pool1", Namespace: RookNamespace},
			Status:     &cephv1.CephBlockPoolStatus{Phase: cephv1.ConditionReady},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pool2", Namespace: RookNamespace},
			Status:     &cephv1.CephBlockPoolStatus{Phase: cephv1.ConditionReady},
		},
	},
}

var CephBlockPoolListNotReady = cephv1.CephBlockPoolList{
	Items: []cephv1.CephBlockPool{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pool1", Namespace: RookNamespace},
			Status:     &cephv1.CephBlockPoolStatus{Phase: cephv1.ConditionFailure},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pool2", Namespace: RookNamespace},
		},
	},
}

var CephBlockPoolReplicated = cephv1.CephBlockPool{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pool1-hdd",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.NamedBlockPoolSpec{
		PoolSpec: cephv1.PoolSpec{
			DeviceClass:   "hdd",
			CrushRoot:     "default",
			FailureDomain: "host",
			Replicated: cephv1.ReplicatedSpec{
				Size: 3,
			},
		},
	},
}

var CephBlockPoolReplicatedMirroring = cephv1.CephBlockPool{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pool1-hdd",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.NamedBlockPoolSpec{
		PoolSpec: cephv1.PoolSpec{
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
	},
}

var CephBlockPoolErasureCoded = cephv1.CephBlockPool{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pool1-hdd",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.NamedBlockPoolSpec{
		PoolSpec: cephv1.PoolSpec{
			DeviceClass:   "hdd",
			CrushRoot:     "default",
			FailureDomain: "host",
			ErasureCoded: cephv1.ErasureCodedSpec{
				CodingChunks: 1,
				DataChunks:   2,
				Algorithm:    "fake",
			},
		},
	},
}

var CephBlockPoolListBaseReady = cephv1.CephBlockPoolList{
	Items: []cephv1.CephBlockPool{GetCephBlockPoolWithStatus(CephBlockPoolReplicated, true)},
}

var CephBlockPoolListBaseNotReady = cephv1.CephBlockPoolList{
	Items: []cephv1.CephBlockPool{GetCephBlockPoolWithStatus(CephBlockPoolReplicated, false)},
}

func GetCephBlockPoolWithStatus(pool cephv1.CephBlockPool, ready bool) cephv1.CephBlockPool {
	newPool := pool.DeepCopy()
	newPool.Status = &cephv1.CephBlockPoolStatus{Phase: cephv1.ConditionProgressing}
	if ready {
		newPool.Status.Phase = cephv1.ConditionReady
	}
	return *newPool
}

func GetOpenstackPool(name string, ready bool, targetRatio float64) cephv1.CephBlockPool {
	pool := GetCephBlockPoolWithStatus(CephBlockPoolReplicated, ready)
	pool.Name = name
	if targetRatio != 0 {
		pool.Spec.Replicated.TargetSizeRatio = targetRatio
	}
	return pool
}

var OpenstackCephBlockPoolsList = cephv1.CephBlockPoolList{
	Items: []cephv1.CephBlockPool{
		GetOpenstackPool("vms-hdd", false, 0.2), GetOpenstackPool("volumes-hdd", false, 0.4), GetOpenstackPool("images-hdd", false, 0.1), GetOpenstackPool("backup-hdd", false, 0.1),
	},
}

var OpenstackCephBlockPoolsListReady = cephv1.CephBlockPoolList{
	Items: []cephv1.CephBlockPool{
		GetOpenstackPool("vms-hdd", true, 0.2), GetOpenstackPool("volumes-hdd", true, 0.4), GetOpenstackPool("images-hdd", true, 0.1), GetOpenstackPool("backup-hdd", true, 0.1),
	},
}

var BuiltinMgrPool = &cephv1.CephBlockPool{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "builtin-mgr",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.NamedBlockPoolSpec{
		Name: ".mgr",
		PoolSpec: cephv1.PoolSpec{
			EnableCrushUpdates: &TrueVarForPointer,
			DeviceClass:        "hdd",
			Replicated: cephv1.ReplicatedSpec{
				Size: 3,
			},
			FailureDomain: "host",
			CrushRoot:     "default",
		},
	},
}

var BuiltinRgwRootPool = &cephv1.CephBlockPool{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "builtin-rgw-root",
		Namespace: "rook-ceph",
	},
	Spec: cephv1.NamedBlockPoolSpec{
		Name: ".rgw.root",
		PoolSpec: cephv1.PoolSpec{
			EnableCrushUpdates: &TrueVarForPointer,
			DeviceClass:        "hdd",
			Replicated: cephv1.ReplicatedSpec{
				Size: 3,
			},
		},
	},
}
