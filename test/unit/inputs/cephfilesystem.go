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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var CephFilesystemListEmpty = cephv1.CephFilesystemList{Items: []cephv1.CephFilesystem{}}

var CephFilesystemNoActiveStandbyReady = cephv1.CephFilesystem{
	ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "cephfs-1"},
	Spec: cephv1.FilesystemSpec{
		MetadataServer: cephv1.MetadataServerSpec{
			ActiveCount: 1,
		},
	},
	Status: &cephv1.CephFilesystemStatus{Phase: cephv1.ConditionReady},
}

var CephFilesystemActiveStandbyReady = cephv1.CephFilesystem{
	ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "cephfs-2"},
	Spec: cephv1.FilesystemSpec{
		MetadataServer: cephv1.MetadataServerSpec{
			ActiveCount:   1,
			ActiveStandby: true,
		},
	},
	Status: &cephv1.CephFilesystemStatus{Phase: cephv1.ConditionReady},
}

var TestCephFs = cephv1.CephFilesystem{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "test-cephfs",
	},
	Spec: cephv1.FilesystemSpec{
		MetadataPool: cephv1.NamedPoolSpec{
			PoolSpec: cephv1.PoolSpec{
				DeviceClass: "hdd",
				Replicated: cephv1.ReplicatedSpec{
					Size: 3,
				},
			},
		},
		DataPools: []cephv1.NamedPoolSpec{
			{
				Name: "some-pool-name",
				PoolSpec: cephv1.PoolSpec{
					DeviceClass: "hdd",
					Replicated: cephv1.ReplicatedSpec{
						Size: 3,
					},
				},
			},
		},
		MetadataServer: cephv1.MetadataServerSpec{
			Annotations: map[string]string{
				"cephdeployment.lcm.mirantis.com/config-global-updated": "some-time",
				"cephdeployment.lcm.mirantis.com/config-mds-updated":    "some-time",
			},
			ActiveCount:   1,
			ActiveStandby: true,
			Placement: cephv1.Placement{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "ceph_role_mds",
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
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "rook_file_system",
										Operator: "In",
										Values: []string{
											"test-cephfs",
										},
									},
								},
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      "ceph_role_mds",
						Operator: "Exists",
					},
				},
			},
			LivenessProbe: &cephv1.ProbeSpec{
				Probe: &corev1.Probe{
					TimeoutSeconds:   5,
					FailureThreshold: 5,
				},
			},
		},
	},
}

var TestCephFsWithTolerationsAndResources = func() cephv1.CephFilesystem {
	fs := TestCephFs.DeepCopy()
	fs.Spec.MetadataServer.Annotations["cephdeployment.lcm.mirantis.com/config-mds.test-cephfs-updated"] = "some-time"
	fs.Spec.MetadataServer.Placement.Tolerations = append(fs.Spec.MetadataServer.Placement.Tolerations, corev1.Toleration{
		Key:      "test.kubernetes.io/testkey",
		Effect:   "Schedule",
		Operator: "Exists",
	})
	fs.Spec.MetadataServer.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("156Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("28Mi"),
			corev1.ResourceCPU:    resource.MustParse("10m"),
		},
	}
	return *fs
}()

func GetCephFsWithStatus(condition cephv1.ConditionType) *cephv1.CephFilesystem {
	cephFs := TestCephFs.DeepCopy()
	cephFs.Status = &cephv1.CephFilesystemStatus{Phase: condition}
	return cephFs
}

var CephFSList = &cephv1.CephFilesystemList{
	Items: []cephv1.CephFilesystem{TestCephFs},
}

var CephFSListReady = &cephv1.CephFilesystemList{
	Items: []cephv1.CephFilesystem{*GetCephFsWithStatus(cephv1.ConditionReady)},
}

var CephFilesystemListSingleReady = cephv1.CephFilesystemList{
	Items: []cephv1.CephFilesystem{CephFilesystemNoActiveStandbyReady},
}

var CephFilesystemListMultipleReady = cephv1.CephFilesystemList{
	Items: []cephv1.CephFilesystem{CephFilesystemNoActiveStandbyReady, CephFilesystemActiveStandbyReady},
}

var CephFilesystemListMultipleNotReady = cephv1.CephFilesystemList{
	Items: []cephv1.CephFilesystem{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "cephfs-1"},
			Spec:       CephFilesystemNoActiveStandbyReady.Spec,
			Status:     &cephv1.CephFilesystemStatus{Phase: cephv1.ConditionFailure},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: RookNamespace, Name: "cephfs-2"},
			Spec:       CephFilesystemActiveStandbyReady.Spec,
		},
	},
}
