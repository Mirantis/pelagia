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

package lcmcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
)

func TestRunPodCommandWithValidation(t *testing.T) {
	kubeClient := faketestclients.GetFakeKubeclient()
	fakeRestConfig := &rest.Config{}
	tests := []struct {
		name           string
		execConfig     ExecConfig
		podList        *v1.PodList
		expectedError  string
		expectedStdout string
		expectedStdErr string
	}{
		{
			name:          "no command provided",
			execConfig:    ExecConfig{Kubeclient: kubeClient},
			expectedError: "command is not specified",
		},
		{
			name: "config create failed",
			execConfig: ExecConfig{
				Kubeclient: kubeClient,
				Command:    "sleep",
			},
			expectedError: "failed to get rest config: unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined",
		},
		{
			name: "failed to get pods",
			execConfig: ExecConfig{
				Kubeclient: kubeClient,
				Command:    "sleep",
				Config:     fakeRestConfig,
			},
			expectedError: "failed to find pod to run command: failed to get pods list: failed to list pods",
		},
		{
			name: "no pods found at all",
			execConfig: ExecConfig{
				Kubeclient: kubeClient,
				Command:    "sleep",
				Config:     fakeRestConfig,
				Labels:     []string{"app=test"},
			},
			podList:       &v1.PodList{},
			expectedError: "failed to find pod to run command: no pods found matching criteria (label(s): 'app=test') in namespace 'default'",
		},
		{
			name: "no running pods found",
			execConfig: ExecConfig{
				Kubeclient: kubeClient,
				Command:    "sleep",
				Config:     fakeRestConfig,
			},
			podList: &v1.PodList{
				Items: []v1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "test-pod"},
						Status:     v1.PodStatus{},
					},
				},
			},
			expectedError: "failed to find pod to run command: no running ready pod matching criteria (no labels specified) in namespace 'default'",
		},
		{
			name: "no containers in pod found",
			execConfig: ExecConfig{
				Kubeclient: kubeClient,
				Command:    "sleep",
				Config:     fakeRestConfig,
				Nodename:   "test-node",
			},
			podList: &v1.PodList{
				Items: []v1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "test-pod"},
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{Name: "test"},
							},
						},
						Status: v1.PodStatus{
							Phase: v1.PodRunning,
							Conditions: []v1.PodCondition{
								{
									Type:   v1.PodReady,
									Status: v1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			expectedError: "failed to find pod to run command: no ready container 'test' found for pod matching criteria (no labels specified, field: 'spec.nodeName=test-node') in namespace 'default'",
		},
		{
			name: "run ok",
			execConfig: ExecConfig{
				Kubeclient: kubeClient,
				Namespace:  "lcm-test",
				Command:    "sleep",
				Config:     fakeRestConfig,
				Labels:     []string{"app=test"},
			},
			podList: &v1.PodList{
				Items: []v1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-pod",
							Namespace: "lcm-test",
							Labels:    map[string]string{"app": "test"},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{Name: "test"},
							},
						},
						Status: v1.PodStatus{
							Phase: v1.PodRunning,
							Conditions: []v1.PodCondition{
								{
									Type:   v1.PodReady,
									Status: v1.ConditionTrue,
								},
							},
							ContainerStatuses: []v1.ContainerStatus{
								{
									Ready: true,
									Name:  "test",
								},
							},
						},
					},
				},
			},
			expectedStdout: "stdout",
			expectedStdErr: "stderr",
		},
	}
	oldFunc := RunPodCommand
	RunPodCommand = func(_ ExecConfig) (string, string, error) {
		return "stdout", "stderr", nil
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := map[string]runtime.Object{}
			if test.podList != nil {
				res["pods"] = test.podList
			}
			faketestclients.FakeReaction(test.execConfig.Kubeclient.CoreV1(), "list", []string{"pods"}, res, nil)

			stdOut, stdErr, err := RunPodCommandWithValidation(test.execConfig)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expectedStdout, stdOut)
			assert.Equal(t, test.expectedStdErr, stdErr)

			faketestclients.CleanupFakeClientReactions(test.execConfig.Kubeclient.CoreV1())
		})
	}
	RunPodCommand = oldFunc
}
