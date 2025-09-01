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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var PodListEmpty = &corev1.PodList{}
var ToolBoxPodList = &corev1.PodList{
	Items: []corev1.Pod{
		GetReadySimplePod("pelagia-ceph-toolbox", RookNamespace, map[string]string{"app": "pelagia-ceph-toolbox"}),
	},
}
var ToolBoxAndDiskDaemonPodsList = &corev1.PodList{
	Items: []corev1.Pod{
		GetReadySimplePod("pelagia-disk-daemon", RookNamespace, map[string]string{"app": "pelagia-disk-daemon"}),
		GetReadySimplePod("pelagia-ceph-toolbox", LcmObjectMeta.Namespace, map[string]string{"app": "pelagia-ceph-toolbox"}),
	},
}

func GetReadySimplePod(name, namespace string, labels map[string]string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: name},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Ready: true,
					Name:  name,
				},
			},
		},
	}
}

var CsiRbdPod = corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "csi-rbdplugin",
		Labels: map[string]string{
			"app": "csi-rbdplugin",
		},
	},
	Spec: corev1.PodSpec{
		NodeName: "node-1",
	},
}
