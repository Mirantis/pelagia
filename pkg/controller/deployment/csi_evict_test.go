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

package deployment

import (
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	fakestorage "k8s.io/client-go/kubernetes/typed/storage/v1/fake"
	gotesting "k8s.io/client-go/testing"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestDetachCSIVolumes(t *testing.T) {
	tests := []struct {
		name                         string
		nodeName                     string
		podList                      v1.PodList
		volumeAttachmentList         storagev1.VolumeAttachmentList
		mountCommandOutput           map[int]string
		actions                      map[string]map[int]string
		isTimeout                    bool
		expectedUmount               map[int]string
		expectedVolumeAttachmentName string
		expectedError                error
	}{
		{
			name:     "no csi pods, skipped",
			podList:  v1.PodList{Items: []v1.Pod{}},
			nodeName: "node-1",
			actions: map[string]map[int]string{
				"list pods": {1: "return"},
			},
		},
		{
			name:     "csi pod found on different node, skipped",
			podList:  v1.PodList{Items: []v1.Pod{unitinputs.CsiRbdPod}},
			nodeName: "node-2",
			actions: map[string]map[int]string{
				"list pods": {1: "return"},
			},
		},
		{
			name:     "csi pod list error, failed",
			podList:  v1.PodList{Items: []v1.Pod{unitinputs.CsiRbdPod}},
			nodeName: "node-1",
			actions: map[string]map[int]string{
				"list pods": {1: "error"},
			},
			isTimeout:     true,
			expectedError: errors.New("timeout to detach volumes from node 'node-1'"),
		},
		{
			name:                 "umount rbd volumes, success",
			podList:              v1.PodList{Items: []v1.Pod{unitinputs.CsiRbdPod}},
			volumeAttachmentList: storagev1.VolumeAttachmentList{Items: []storagev1.VolumeAttachment{}},
			nodeName:             "node-1",
			mountCommandOutput: map[int]string{
				2: "/dev/rbd0 /dev/dm0\n/dev/rbd1 /dev/dm1",
				3: "/dev/rbd0 /dev/dm0\n/dev/rbd1 /dev/dm1",
				4: "",
			},
			actions: map[string]map[int]string{
				"list pods": {1: "return"},
				"mount": {
					1: "error",
					2: "return",
					3: "return",
					4: "return",
				},
				"umount": {
					1: "return",
					2: "error",
					3: "return",
					4: "return",
				},
				"list volumeattachments": {
					1: "return",
				},
			},
			expectedUmount: map[int]string{
				1: "umount /dev/rbd0",
				2: "umount /dev/rbd1",
				3: "umount /dev/rbd0",
				4: "umount /dev/rbd1",
			},
		},
		{
			name:                 "umount rbd volumes timeout, failed",
			isTimeout:            true,
			podList:              v1.PodList{Items: []v1.Pod{unitinputs.CsiRbdPod}},
			volumeAttachmentList: storagev1.VolumeAttachmentList{Items: []storagev1.VolumeAttachment{}},
			nodeName:             "node-1",
			actions: map[string]map[int]string{
				"list pods": {1: "return"},
				"mount":     {1: "error"},
			},
			expectedError: errors.New("timeout to detach volumes from node 'node-1'"),
		},
		{
			name:    "delete volumeattachments, success",
			podList: v1.PodList{Items: []v1.Pod{unitinputs.CsiRbdPod}},
			volumeAttachmentList: storagev1.VolumeAttachmentList{
				Items: []storagev1.VolumeAttachment{
					unitinputs.CephVolumeAttachment,
					unitinputs.NonCephVolumeAttachment,
				},
			},
			nodeName: "node-1",
			mountCommandOutput: map[int]string{
				1: "",
				2: "",
			},
			actions: map[string]map[int]string{
				"list pods": {
					1: "return",
					2: "return",
				},
				"mount": {
					1: "return",
					2: "return",
				},
				"delete volumeattachments": {
					1: "error",
					2: "return",
					3: "not found",
				},
				"list volumeattachments": {
					1: "error",
					2: "return",
				},
			},
			expectedVolumeAttachmentName: "ceph-volumeattachment",
		},
		{
			name:      "delete volumeattachments timeout, failed",
			isTimeout: true,
			podList:   v1.PodList{Items: []v1.Pod{unitinputs.CsiRbdPod}},
			volumeAttachmentList: storagev1.VolumeAttachmentList{
				Items: []storagev1.VolumeAttachment{
					unitinputs.CephVolumeAttachment,
					unitinputs.NonCephVolumeAttachment,
				},
			},
			nodeName: "node-1",
			mountCommandOutput: map[int]string{
				1: "",
			},
			actions: map[string]map[int]string{
				"list pods":                {1: "return"},
				"mount":                    {1: "return"},
				"list volumeattachments":   {1: "return"},
				"delete volumeattachments": {1: "error"},
			},
			expectedVolumeAttachmentName: "ceph-volumeattachment",
			expectedError:                errors.New("timeout to detach volumes from node 'node-1'"),
		},
	}
	actionsCount := map[string]int{}
	oldDetachCSIVolumesTimeout := detachCSIVolumesTimeout
	oldVerifyRBDVolumesMountsTimeout := verifyRBDVolumesMountsTimeout
	oldVerifyRBDVolumeAttachmentsTimeout := verifyRBDVolumeAttachmentsTimeout
	oldCSIPollInterval := csiPollInterval
	oldCmdRunFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			csiPollInterval = 300 * time.Millisecond
			if test.isTimeout {
				detachCSIVolumesTimeout = 200 * time.Millisecond
				verifyRBDVolumesMountsTimeout = 200 * time.Millisecond
				verifyRBDVolumeAttachmentsTimeout = 200 * time.Millisecond
			} else {
				detachCSIVolumesTimeout = 5 * time.Second
				verifyRBDVolumesMountsTimeout = 5 * time.Second
				verifyRBDVolumeAttachmentsTimeout = 5 * time.Second
			}

			actionsCount = map[string]int{
				"list pods":                0,
				"mount":                    0,
				"umount":                   0,
				"list volumeattachments":   0,
				"delete volumeattachments": 0,
			}

			c.api.Kubeclientset.CoreV1().(*fakecorev1.FakeCoreV1).AddReactor("list", "pods", func(_ gotesting.Action) (handled bool, ret runtime.Object, err error) {
				actionsCount["list pods"]++
				switch test.actions["list pods"][actionsCount["list pods"]] {
				case "return":
					return true, test.podList.DeepCopy(), nil
				case "error":
					return true, nil, errors.New("list pods failed")
				}
				return true, nil, errors.Errorf("unexpected list pods call, count = %v", actionsCount["list pods"])
			})
			c.api.Kubeclientset.StorageV1().(*fakestorage.FakeStorageV1).AddReactor("list", "volumeattachments", func(_ gotesting.Action) (handled bool, ret runtime.Object, err error) {
				actionsCount["list volumeattachments"]++
				switch test.actions["list volumeattachments"][actionsCount["list volumeattachments"]] {
				case "return":
					return true, test.volumeAttachmentList.DeepCopy(), nil
				case "error":
					return true, nil, errors.New("list volumeattachments failed")
				}
				return true, nil, errors.Errorf("unexpected list volumeattachments call, count = %v", actionsCount["list volumeattachments"])
			})
			c.api.Kubeclientset.StorageV1().(*fakestorage.FakeStorageV1).AddReactor("delete", "volumeattachments", func(action gotesting.Action) (handled bool, ret runtime.Object, err error) {
				if test.expectedVolumeAttachmentName != "" {
					actual := action.(gotesting.DeleteActionImpl).Name
					assert.Equal(t, test.expectedVolumeAttachmentName, actual)
				}
				actionsCount["delete volumeattachments"]++
				switch test.actions["delete volumeattachments"][actionsCount["delete volumeattachments"]] {
				case "return":
					return true, nil, nil
				case "error":
					return true, nil, errors.New("delete volumeattachments failed")
				case "not found":
					return true, nil, apierrors.NewNotFound(schema.GroupResource{Group: "storagev1", Resource: "volumeattachments"}, "fake")
				}
				return true, nil, errors.Errorf("unexpected delete volumeattachments call, count = %v", actionsCount["delete volumeattachments"])
			})
			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if e.Command == "mount" {
					actionsCount["mount"]++
					switch test.actions["mount"][actionsCount["mount"]] {
					case "return":
						return test.mountCommandOutput[actionsCount["mount"]], "", nil
					case "error":
						return "", "stderr", errors.New("mount command failed")
					}
					return "", "stderr", errors.Errorf("unexpected mount command call, count = %v", actionsCount["mount"])
				} else if strings.HasPrefix(e.Command, "umount") {
					actionsCount["umount"]++
					if expected, ok := test.expectedUmount[actionsCount["umount"]]; ok {
						assert.Equal(t, expected, e.Command)
					}
					switch test.actions["umount"][actionsCount["umount"]] {
					case "return":
						return "", "", nil
					case "error":
						return "", "stderr", errors.New("umount command failed")
					}
					return "", "stderr", errors.Errorf("unexpected umount command call, count = %v", actionsCount["umount"])
				}
				return "", "", errors.New("unexpected run command call")
			}
			err := c.detachCSIVolumes(test.nodeName)
			if test.expectedError != nil {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), test.expectedError.Error())
			} else {
				assert.Nil(t, err)
			}
			for k, v := range actionsCount {
				assert.Equal(t, len(test.actions[k]), v)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.StorageV1())
		})
	}
	detachCSIVolumesTimeout = oldDetachCSIVolumesTimeout
	verifyRBDVolumesMountsTimeout = oldVerifyRBDVolumesMountsTimeout
	verifyRBDVolumeAttachmentsTimeout = oldVerifyRBDVolumeAttachmentsTimeout
	csiPollInterval = oldCSIPollInterval
	lcmcommon.RunPodCommandWithValidation = oldCmdRunFunc
}

func TestEnsureDaemonsetLabels(t *testing.T) {
	tests := []struct {
		name               string
		nodeList           v1.NodeList
		podList            map[int]v1.PodList
		nodeGet            map[int]*v1.Node
		isTimeout          bool
		nodeListError      error
		actions            map[string]map[int]string
		expectedNodeUpdate map[int]*v1.Node
		expectedPodDelete  map[int]string
	}{
		{
			name:     "no lcm-drained, csi labels exist, skipped",
			nodeList: v1.NodeList{Items: []v1.Node{unitinputs.DaemonSetLabelsNode}},
		},
		{
			name:          "failed to list nodes, skipped",
			nodeList:      v1.NodeList{Items: []v1.Node{unitinputs.DaemonSetLabelsNode}},
			nodeListError: errors.New("failed to list nodes"),
		},
		{
			name:     "no lcm-drained, no csi labels, update node with csi label",
			nodeList: v1.NodeList{Items: []v1.Node{unitinputs.EmptyLabelsNode}},
			nodeGet: map[int]*v1.Node{
				1: unitinputs.EmptyLabelsNode.DeepCopy(),
			},
			actions: map[string]map[int]string{
				"get node":    {1: "return"},
				"update node": {1: "return"},
			},
			expectedNodeUpdate: map[int]*v1.Node{
				1: unitinputs.DaemonSetLabelsNode.DeepCopy(),
			},
		},
		{
			name:     "no lcm-drained, no csi labels, get node failed",
			nodeList: v1.NodeList{Items: []v1.Node{unitinputs.EmptyLabelsNode}},
			actions: map[string]map[int]string{
				"get node": {1: "error"},
			},
		},
		{
			name:     "no lcm-drained, no csi labels, update node failed",
			nodeList: v1.NodeList{Items: []v1.Node{unitinputs.EmptyLabelsNode}},
			nodeGet: map[int]*v1.Node{
				1: unitinputs.EmptyLabelsNode.DeepCopy(),
			},
			actions: map[string]map[int]string{
				"get node":    {1: "return"},
				"update node": {1: "error"},
			},
			expectedNodeUpdate: map[int]*v1.Node{
				1: unitinputs.DaemonSetLabelsNode.DeepCopy(),
			},
		},
		{
			name:     "lcm-drained found, no csi labels, csi-drained annotation, skipped",
			nodeList: v1.NodeList{Items: []v1.Node{unitinputs.CsiDrainedAnnotationsNode}},
			podList: map[int]v1.PodList{
				1: {Items: []v1.Pod{}},
				2: {Items: []v1.Pod{}},
			},
			nodeGet: map[int]*v1.Node{
				1: unitinputs.CsiDrainedAnnotationsNode.DeepCopy(),
			},
			actions: map[string]map[int]string{
				"get node": {1: "return"},
				"list pods": {
					1: "return",
					2: "return",
				},
			},
		},
		{
			name:     "lcm-drained found, csi labels, detach csi volumes failed",
			nodeList: v1.NodeList{Items: []v1.Node{unitinputs.LcmDrainedAnnotationsNode}},
			actions: map[string]map[int]string{
				"list pods": {
					1: "error",
				},
			},
		},
		{
			name:     "lcm-drained found, csi labels, get node to delete label failed",
			nodeList: v1.NodeList{Items: []v1.Node{unitinputs.LcmDrainedAnnotationsNode}},
			podList: map[int]v1.PodList{
				1: {Items: []v1.Pod{}},
			},
			actions: map[string]map[int]string{
				"list pods": {
					1: "return",
				},
				"get node": {
					1: "error",
				},
			},
		},
		{
			name:     "lcm-drained found, csi labels, update node to delete label failed",
			nodeList: v1.NodeList{Items: []v1.Node{unitinputs.LcmDrainedAnnotationsNode}},
			nodeGet: map[int]*v1.Node{
				1: unitinputs.LcmDrainedAnnotationsNode.DeepCopy(),
			},
			podList: map[int]v1.PodList{
				1: {Items: []v1.Pod{}},
			},
			actions: map[string]map[int]string{
				"list pods": {
					1: "return",
				},
				"get node": {
					1: "return",
				},
				"update node": {
					1: "error",
				},
			},
			expectedNodeUpdate: map[int]*v1.Node{
				1: unitinputs.LcmDrainedNoCsiLabelNode.DeepCopy(),
			},
		},
		{
			name:     "lcm-drained found, csi labels, list pod on verify evicted failed",
			nodeList: v1.NodeList{Items: []v1.Node{unitinputs.LcmDrainedAnnotationsNode}},
			nodeGet: map[int]*v1.Node{
				1: unitinputs.LcmDrainedAnnotationsNode.DeepCopy(),
			},
			podList: map[int]v1.PodList{
				1: {Items: []v1.Pod{}},
			},
			actions: map[string]map[int]string{
				"list pods": {
					1: "return",
					2: "error",
				},
				"get node": {
					1: "return",
				},
				"update node": {
					1: "return",
				},
			},
			expectedNodeUpdate: map[int]*v1.Node{
				1: unitinputs.LcmDrainedNoCsiLabelNode.DeepCopy(),
			},
		},
		{
			name:      "lcm-drained found, csi labels, delete pod on verify evicted failed",
			isTimeout: true,
			nodeList:  v1.NodeList{Items: []v1.Node{unitinputs.LcmDrainedAnnotationsNode}},
			podList: map[int]v1.PodList{
				1: {Items: []v1.Pod{}},
				2: {Items: []v1.Pod{unitinputs.CsiRbdPod}},
			},
			nodeGet: map[int]*v1.Node{
				1: unitinputs.LcmDrainedAnnotationsNode.DeepCopy(),
			},
			actions: map[string]map[int]string{
				"list pods": {
					1: "return",
					2: "return",
				},
				"get node": {
					1: "return",
				},
				"update node": {
					1: "return",
				},
				"delete pod": {
					1: "error",
				},
			},
			expectedNodeUpdate: map[int]*v1.Node{
				1: unitinputs.LcmDrainedNoCsiLabelNode.DeepCopy(),
			},
			expectedPodDelete: map[int]string{
				1: "csi-rbdplugin",
			},
		},
		{
			name:      "lcm-drained found, csi labels, wait for daemonset pods failed",
			isTimeout: true,
			nodeList:  v1.NodeList{Items: []v1.Node{unitinputs.LcmDrainedAnnotationsNode}},
			podList: map[int]v1.PodList{
				1: {Items: []v1.Pod{}},
				2: {Items: []v1.Pod{unitinputs.CsiRbdPod}},
			},
			nodeGet: map[int]*v1.Node{
				1: unitinputs.LcmDrainedAnnotationsNode.DeepCopy(),
			},
			actions: map[string]map[int]string{
				"list pods": {
					1: "return",
					2: "return",
					3: "error",
				},
				"get node": {
					1: "return",
				},
				"update node": {
					1: "return",
				},
				"delete pod": {
					1: "not found",
				},
			},
			expectedNodeUpdate: map[int]*v1.Node{
				1: unitinputs.LcmDrainedNoCsiLabelNode.DeepCopy(),
			},
			expectedPodDelete: map[int]string{
				1: "csi-rbdplugin",
			},
		},
		{
			name:      "lcm-drained found, csi labels, get node to set annotation failed",
			isTimeout: true,
			nodeList:  v1.NodeList{Items: []v1.Node{unitinputs.LcmDrainedAnnotationsNode}},
			podList: map[int]v1.PodList{
				1: {Items: []v1.Pod{}},
				2: {Items: []v1.Pod{}},
				3: {Items: []v1.Pod{}},
			},
			nodeGet: map[int]*v1.Node{
				1: unitinputs.LcmDrainedAnnotationsNode.DeepCopy(),
			},
			actions: map[string]map[int]string{
				"list pods": {
					1: "return",
					2: "return",
					3: "return",
				},
				"get node": {
					1: "return",
					2: "error",
				},
				"update node": {
					1: "return",
				},
			},
			expectedNodeUpdate: map[int]*v1.Node{
				1: unitinputs.LcmDrainedNoCsiLabelNode.DeepCopy(),
			},
			expectedPodDelete: map[int]string{
				1: "csi-rbdplugin",
			},
		},
		{
			name:      "lcm-drained found, csi labels, update node to set annotation failed",
			isTimeout: true,
			nodeList:  v1.NodeList{Items: []v1.Node{unitinputs.LcmDrainedAnnotationsNode}},
			podList: map[int]v1.PodList{
				1: {Items: []v1.Pod{}},
				2: {Items: []v1.Pod{}},
				3: {Items: []v1.Pod{}},
			},
			nodeGet: map[int]*v1.Node{
				1: unitinputs.LcmDrainedAnnotationsNode.DeepCopy(),
				2: unitinputs.LcmDrainedNoCsiLabelNode.DeepCopy(),
			},
			actions: map[string]map[int]string{
				"list pods": {
					1: "return",
					2: "return",
					3: "return",
				},
				"get node": {
					1: "return",
					2: "return",
				},
				"update node": {
					1: "return",
					2: "error",
				},
			},
			expectedNodeUpdate: map[int]*v1.Node{
				1: unitinputs.LcmDrainedNoCsiLabelNode.DeepCopy(),
				2: unitinputs.CsiDrainedAnnotationsNode.DeepCopy(),
			},
			expectedPodDelete: map[int]string{
				1: "csi-rbdplugin",
			},
		},
		{
			name:     "lcm-drained found, csi labels, success",
			nodeList: v1.NodeList{Items: []v1.Node{unitinputs.LcmDrainedAnnotationsNode}},
			podList: map[int]v1.PodList{
				1: {Items: []v1.Pod{}},
				2: {Items: []v1.Pod{unitinputs.CsiRbdPod}},
				3: {Items: []v1.Pod{unitinputs.CsiRbdPod}},
				4: {Items: []v1.Pod{}},
			},
			nodeGet: map[int]*v1.Node{
				1: unitinputs.LcmDrainedAnnotationsNode.DeepCopy(),
				2: unitinputs.LcmDrainedNoCsiLabelNode.DeepCopy(),
			},
			actions: map[string]map[int]string{
				"list pods": {
					1: "return",
					2: "return",
					3: "return",
					4: "return",
				},
				"get node": {
					1: "return",
					2: "return",
				},
				"update node": {
					1: "return",
					2: "return",
				},
				"delete pod": {
					1: "return",
					2: "error",
					3: "not found",
				},
			},
			expectedNodeUpdate: map[int]*v1.Node{
				1: unitinputs.LcmDrainedNoCsiLabelNode.DeepCopy(),
				2: unitinputs.CsiDrainedAnnotationsNode.DeepCopy(),
			},
			expectedPodDelete: map[int]string{
				1: "csi-rbdplugin",
				2: "csi-rbdplugin",
				3: "csi-rbdplugin",
			},
		},
	}
	oldVerifyCSIPodEvictedTimeout := verifyCSIPodEvictedTimeout
	oldWaitForDaemonsetsPodsTimeout := waitForDaemonsetsPodsTimeout
	oldWaitForDaemonsetsPodsInterval := waitForDaemonsetsPodsInterval
	oldCSIPollInterval := csiPollInterval
	oldDetachCSIVolumesTimeout := detachCSIVolumesTimeout
	actionsCount := map[string]int{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: unitinputs.BaseCephDeployment.DeepCopy()}, nil)
			waitForDaemonsetsPodsInterval = 300 * time.Millisecond
			csiPollInterval = 300 * time.Millisecond
			detachCSIVolumesTimeout = 200 * time.Millisecond
			if test.isTimeout {
				verifyCSIPodEvictedTimeout = 200 * time.Millisecond
				waitForDaemonsetsPodsTimeout = 200 * time.Millisecond
			} else {
				verifyCSIPodEvictedTimeout = 5 * time.Second
				waitForDaemonsetsPodsTimeout = 5 * time.Second
			}

			actionsCount = map[string]int{
				"get node":    0,
				"update node": 0,
				"list pods":   0,
				"delete pod":  0,
			}

			c.api.Kubeclientset.CoreV1().(*fakecorev1.FakeCoreV1).AddReactor("list", "nodes", func(_ gotesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, test.nodeList.DeepCopy(), test.nodeListError
			})
			c.api.Kubeclientset.CoreV1().(*fakecorev1.FakeCoreV1).AddReactor("list", "pods", func(_ gotesting.Action) (handled bool, ret runtime.Object, err error) {
				actionsCount["list pods"]++
				switch test.actions["list pods"][actionsCount["list pods"]] {
				case "return":
					list := test.podList[actionsCount["list pods"]]
					return true, list.DeepCopy(), nil
				case "error":
					return true, nil, errors.New("list pods failed")
				}
				return true, nil, errors.Errorf("unexpected list pods call, count = %v", actionsCount["list pods"])
			})
			c.api.Kubeclientset.CoreV1().(*fakecorev1.FakeCoreV1).AddReactor("delete", "pods", func(action gotesting.Action) (handled bool, ret runtime.Object, err error) {
				actionsCount["delete pod"]++
				if expected, ok := test.expectedPodDelete[actionsCount["delete pod"]]; ok {
					actual := action.(gotesting.DeleteActionImpl).Name
					assert.Equal(t, expected, actual)
				}
				switch test.actions["delete pod"][actionsCount["delete pod"]] {
				case "return":
					return true, nil, nil
				case "error":
					return true, nil, errors.New("delete pod failed")
				case "not found":
					return true, nil, apierrors.NewNotFound(schema.GroupResource{Group: "v1", Resource: "pods"}, "fake")
				}
				return true, nil, errors.Errorf("unexpected delete pod call, count = %v", actionsCount["delete pod"])
			})
			c.api.Kubeclientset.CoreV1().(*fakecorev1.FakeCoreV1).AddReactor("get", "nodes", func(_ gotesting.Action) (handled bool, ret runtime.Object, err error) {
				actionsCount["get node"]++
				switch test.actions["get node"][actionsCount["get node"]] {
				case "return":
					return true, test.nodeGet[actionsCount["get node"]].DeepCopy(), nil
				case "error":
					return true, nil, errors.New("get node failed")
				}
				return true, nil, errors.Errorf("unexpected get node call, count = %v", actionsCount["get node"])
			})
			c.api.Kubeclientset.CoreV1().(*fakecorev1.FakeCoreV1).AddReactor("update", "nodes", func(action gotesting.Action) (handled bool, ret runtime.Object, err error) {
				actionsCount["update node"]++
				if expected, ok := test.expectedNodeUpdate[actionsCount["update node"]]; ok {
					actual := action.(gotesting.UpdateActionImpl).Object.(*v1.Node)
					assert.Equal(t, expected, actual)
				}
				switch test.actions["update node"][actionsCount["update node"]] {
				case "return":
					return true, nil, nil
				case "error":
					return true, nil, errors.New("update node failed")
				}
				return true, nil, errors.Errorf("unexpected update node call, count = %v", actionsCount["update node"])
			})
			c.ensureDaemonsetLabels()
			for k, v := range actionsCount {
				assert.Equal(t, len(test.actions[k]), v)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
	verifyCSIPodEvictedTimeout = oldVerifyCSIPodEvictedTimeout
	waitForDaemonsetsPodsTimeout = oldWaitForDaemonsetsPodsTimeout
	waitForDaemonsetsPodsInterval = oldWaitForDaemonsetsPodsInterval
	csiPollInterval = oldCSIPollInterval
	detachCSIVolumesTimeout = oldDetachCSIVolumesTimeout
}
