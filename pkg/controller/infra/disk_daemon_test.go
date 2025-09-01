/*
Copyright 2025 The Mirantis Authors.

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

package infra

import (
	"testing"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestEnsureDiskDaemon(t *testing.T) {
	config := infraConfig{
		namespace:       unitinputs.LcmObjectMeta.Namespace,
		cephImage:       unitinputs.ReefCephClusterReady.Status.CephVersion.Image,
		controllerImage: "some-registry/lcm-controller:v1",
	}
	tests := []struct {
		name              string
		infraConfig       infraConfig
		lcmConfigData     map[string]string
		inputResources    map[string]runtime.Object
		apiErrors         map[string]error
		expectedResources map[string]runtime.Object
		expectedError     string
	}{
		{
			name: "nothing to do for external",
			infraConfig: infraConfig{
				externalCeph: true,
			},
		},
		{
			name:        "current ceph image is unknown",
			infraConfig: infraConfig{},
		},
		{
			name:        "failed to get disk daemon",
			infraConfig: config,
			inputResources: map[string]runtime.Object{
				"daemonsets": unitinputs.DaemonSetListEmpty,
			},
			apiErrors: map[string]error{
				"get-daemonsets-pelagia-disk-daemon": errors.New("failed to get daemonset"),
			},
			expectedError: "failed to check disk-daemon daemonset 'lcm-namespace/pelagia-disk-daemon': failed to get daemonset",
		},
		{
			name:        "create disk daemon",
			infraConfig: config,
			inputResources: map[string]runtime.Object{
				"daemonsets": unitinputs.DaemonSetListEmpty.DeepCopy(),
			},
			expectedResources: map[string]runtime.Object{
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{unitinputs.DiskDaemonDaemonset},
				},
			},
		},
		{
			name:        "create disk daemon failed",
			infraConfig: config,
			inputResources: map[string]runtime.Object{
				"daemonsets": unitinputs.DaemonSetListEmpty.DeepCopy(),
			},
			apiErrors: map[string]error{
				"create-daemonsets-pelagia-disk-daemon": errors.New("failed to create daemonset"),
			},
			expectedError: "failed to create disk-daemon daemonset 'lcm-namespace/pelagia-disk-daemon': failed to create daemonset",
		},
		{
			name:        "update disk daemon failed",
			infraConfig: config,
			inputResources: map[string]runtime.Object{
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{*unitinputs.DiskDaemonDaemonsetWithOsdTolerations.DeepCopy()},
				},
			},
			apiErrors: map[string]error{
				"update-daemonsets-pelagia-disk-daemon": errors.New("failed to update daemonset"),
			},
			expectedError: "failed to update disk-daemon daemonset 'lcm-namespace/pelagia-disk-daemon': failed to update daemonset",
		},
		{
			name:        "update disk daemon",
			infraConfig: config,
			inputResources: map[string]runtime.Object{
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{*unitinputs.DiskDaemonDaemonsetWithOsdTolerations.DeepCopy()},
				},
			},
			expectedResources: map[string]runtime.Object{
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{unitinputs.DiskDaemonDaemonset},
				},
			},
		},
		{
			name:        "update disk daemon meta only",
			infraConfig: config,
			inputResources: map[string]runtime.Object{
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{
						func() appsv1.DaemonSet {
							ds := unitinputs.DiskDaemonDaemonset.DeepCopy()
							ds.OwnerReferences = nil
							ds.Labels = map[string]string{}
							return *ds
						}(),
					},
				},
			},
			expectedResources: map[string]runtime.Object{
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{unitinputs.DiskDaemonDaemonset},
				},
			},
		},
		{
			name:        "nothing to do with disk daemon",
			infraConfig: config,
			inputResources: map[string]runtime.Object{
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{*unitinputs.DiskDaemonDaemonset.DeepCopy()},
				},
			},
		},
		{
			name:        "disk daemon has different port and placement label",
			infraConfig: config,
			lcmConfigData: map[string]string{
				"DISK_DAEMON_API_PORT":                 "3333",
				"DISK_DAEMON_PLACEMENT_NODES_SELECTOR": "custom-disk-daemon-label=true",
			},
			inputResources: map[string]runtime.Object{
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{*unitinputs.DiskDaemonDaemonset.DeepCopy()},
				},
			},
			expectedResources: map[string]runtime.Object{
				"daemonsets": &appsv1.DaemonSetList{
					Items: []appsv1.DaemonSet{
						func() appsv1.DaemonSet {
							ds := unitinputs.DiskDaemonDaemonset.DeepCopy()
							ds.Spec.Template.Spec.NodeSelector = map[string]string{"custom-disk-daemon-label": "true"}
							ds.Spec.Template.Spec.Containers[0].Args = []string{"/usr/local/bin/pelagia-disk-daemon", "--daemon", "--port", "3333"}
							ds.Spec.Template.Spec.Containers[0].LivenessProbe.Exec.Command = []string{
								"/usr/local/bin/pelagia-disk-daemon", "--api-check", "--port", "3333"}
							ds.Spec.Template.Spec.Containers[0].ReadinessProbe.Exec.Command = []string{
								"/usr/local/bin/pelagia-disk-daemon", "--api-check", "--port", "3333"}
							return *ds
						}(),
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeReconcileInfraConfig(&test.infraConfig, test.lcmConfigData)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"daemonsets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "create", []string{"daemonsets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "update", []string{"daemonsets"}, test.inputResources, test.apiErrors)
			test.expectedResources = faketestclients.PrepareExpectedResources(test.inputResources, test.expectedResources)

			err := c.ensureDiskDaemon()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedResources, test.inputResources)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
		})
	}
}

func TestGenerateDiskDaemon(t *testing.T) {
	tests := []struct {
		name              string
		infraConfig       infraConfig
		expectedDaemonSet *appsv1.DaemonSet
	}{
		{
			name: "generate daemonset",
			infraConfig: infraConfig{
				namespace:       "lcm-namespace",
				cephImage:       unitinputs.ReefCephClusterReady.Status.CephVersion.Image,
				controllerImage: "some-registry/lcm-controller:v1",
			},
			expectedDaemonSet: &unitinputs.DiskDaemonDaemonset,
		},
		{
			name: "generate daemonset with osd tolerations",
			infraConfig: infraConfig{
				namespace:       "lcm-namespace",
				cephImage:       unitinputs.ReefCephClusterReady.Status.CephVersion.Image,
				controllerImage: "some-registry/lcm-controller:v1",
				osdPlacement: cephv1.Placement{
					Tolerations: []corev1.Toleration{
						{
							Key:      "test.kubernetes.io/testkey",
							Effect:   "Schedule",
							Operator: "Exists",
						},
					},
				},
			},
			expectedDaemonSet: unitinputs.DiskDaemonDaemonsetWithOsdTolerations,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeReconcileInfraConfig(&test.infraConfig, nil)

			daemonSet := c.generateDiskDaemon()
			assert.Equal(t, test.expectedDaemonSet, daemonSet)
		})
	}
}
