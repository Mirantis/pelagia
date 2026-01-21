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

package osdremove

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestRunCleanupJob(t *testing.T) {
	taskConfigForTest := taskConfig{task: unitinputs.CephOsdRemoveTaskProcessing, cephCluster: &unitinputs.CephClusterReady}
	tests := []struct {
		name             string
		taskConfig       taskConfig
		removeAllLVMs    bool
		osd              string
		host             string
		osdMapping       lcmv1alpha1.OsdMapping
		batchJob         *batch.Job
		apiError         string
		expectedError    string
		expectedBatchJob *batch.Job
	}{
		{
			name:          "job create - failed to get owner refs",
			taskConfig:    taskConfig{},
			expectedError: "failed to get CephOsdRemoveTask owner refs: failed to get GVK for object: expected pointer, but got nil",
		},
		{
			name:          "job create - cephcluster has no image",
			taskConfig:    taskConfig{task: unitinputs.CephOsdRemoveTaskProcessing, cephCluster: &unitinputs.CephClusterNotReady},
			expectedError: "failed to determine ceph cluster image, no current used image in status",
		},
		{
			name:       "job create - devices info has missed important params",
			taskConfig: taskConfigForTest,
			osd:        "2",
			host:       "custom",
			osdMapping: lcmv1alpha1.OsdMapping{
				HostDirectory: "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_69481cd1-38b1-42fd-ac07-06bf4d7c0e19",
				DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{"/dev/vdb": {}},
			},
			expectedError: "partition or device path missed in provided info",
		},
		{
			name:          "job create - failed to check presense",
			taskConfig:    taskConfigForTest,
			osd:           "20",
			host:          "node-1",
			apiError:      "get-jobs",
			osdMapping:    unitinputs.FullNodesRemoveMap.CleanupMap["node-1"].OsdMapping["20"],
			expectedError: "Retries (5/5) exceeded: failed to get-jobs",
		},
		{
			name:          "job create - found job, failed to wait job completion",
			taskConfig:    taskConfigForTest,
			osd:           "20",
			host:          "node-1",
			batchJob:      unitinputs.GetCleanupJobOnlyStatus("device-cleanup-job-node-1-20", "lcm-namespace", 1, 0, 0),
			osdMapping:    unitinputs.FullNodesRemoveMap.CleanupMap["node-1"].OsdMapping["20"],
			expectedError: "Retries (5/5) exceeded: waiting old job",
		},
		{
			name:          "job create - found job, failed to remove",
			taskConfig:    taskConfigForTest,
			osd:           "20",
			host:          "node-1",
			batchJob:      unitinputs.GetCleanupJobOnlyStatus("device-cleanup-job-node-1-20", "lcm-namespace", 0, 0, 1),
			apiError:      "delete-jobs",
			osdMapping:    unitinputs.FullNodesRemoveMap.CleanupMap["node-1"].OsdMapping["20"],
			expectedError: "Retries (5/5) exceeded: failed to delete-jobs",
		},
		{
			name:          "job create - found job, removed, create failed",
			taskConfig:    taskConfigForTest,
			osd:           "20",
			host:          "node-1",
			batchJob:      unitinputs.GetCleanupJobOnlyStatus("device-cleanup-job-node-1-20", "lcm-namespace", 0, 0, 1),
			apiError:      "create-jobs",
			osdMapping:    unitinputs.FullNodesRemoveMap.CleanupMap["node-1"].OsdMapping["20"],
			expectedError: "Retries (5/5) exceeded: failed to create-jobs",
		},
		{
			name:       "job created - full disk zap device and no partition destroy",
			taskConfig: taskConfigForTest,
			osd:        "20",
			host:       "node-1",
			osdMapping: unitinputs.DevNotInSpecRemoveMap.CleanupMap["node-1"].OsdMapping["20"],
			expectedBatchJob: unitinputs.GetCleanupJob("node-1", "20", "", map[string]string{
				"vdd": fmt.Sprintf(cleanupScriptTmpl, fmt.Sprintf(partitionCleanupScriptTmpl, "/dev/ceph-metadata/part-1", false,
					fmt.Sprintf(hostDirectoryCleanupScriptTmpl, "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_vbsgs3a3-sdcv-casq-sd11-asd12dasczsf"))),
				"vde": fmt.Sprintf(cleanupScriptTmpl,
					fmt.Sprintf(partitionCleanupScriptTmpl, "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf", true,
						fmt.Sprintf(diskCleanupScriptTmpl, "/dev/disk/by-path/pci-0000:00:0f.0", true,
							fmt.Sprintf(hostDirectoryCleanupScriptTmpl, "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_vbsgs3a3-sdcv-casq-sd11-asd12dasczsf")))),
			}),
		},
		{
			name:          "job created - full disk zap device and manual partition destroy",
			taskConfig:    taskConfigForTest,
			removeAllLVMs: true,
			osd:           "20",
			host:          "node-1",
			osdMapping:    unitinputs.DevNotInSpecRemoveMap.CleanupMap["node-1"].OsdMapping["20"],
			expectedBatchJob: unitinputs.GetCleanupJob("node-1", "20", "", map[string]string{
				"vdd": fmt.Sprintf(cleanupScriptTmpl, fmt.Sprintf(partitionCleanupScriptTmpl, "/dev/ceph-metadata/part-1", true,
					fmt.Sprintf(hostDirectoryCleanupScriptTmpl, "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_vbsgs3a3-sdcv-casq-sd11-asd12dasczsf"))),
				"vde": fmt.Sprintf(cleanupScriptTmpl,
					fmt.Sprintf(partitionCleanupScriptTmpl, "/dev/ceph-21312wds-sdfv-vs3f-scv3-sdfdsg23edaa/osd-block-vbsgs3a3-sdcv-casq-sd11-asd12dasczsf", true,
						fmt.Sprintf(diskCleanupScriptTmpl, "/dev/disk/by-path/pci-0000:00:0f.0", true,
							fmt.Sprintf(hostDirectoryCleanupScriptTmpl, "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_vbsgs3a3-sdcv-casq-sd11-asd12dasczsf")))),
			}),
		},
		{
			name:       "job created - device is unavailable, dm clean",
			taskConfig: taskConfigForTest,
			osd:        "0",
			host:       "node-2",
			osdMapping: unitinputs.FullNodesInfoFromOsdMeta["node-2"].OsdMapping["0"],
			expectedBatchJob: unitinputs.GetCleanupJob("node-2", "0", "", map[string]string{
				"vdb": fmt.Sprintf(cleanupScriptTmpl, fmt.Sprintf(dmSetupTableClean, "/dev/dm-0",
					fmt.Sprintf(hostDirectoryCleanupScriptTmpl, "/var/lib/rook/rook-ceph/8668f062-3faa-358a-85f3-f80fe6c1e306_69481cd1-38b1-42fd-ac07-06bf4d7c0e19"))),
			}),
		},
		{
			name:       "job created - cleanup for stray osd",
			taskConfig: taskConfigForTest,
			osd:        "0.06bf4d7c-9603-41a4-b250-284ecf3ecb2f.__stray",
			host:       "node-2",
			osdMapping: unitinputs.StrayOnlyOnNodeRemoveMap.CleanupMap["node-2"].OsdMapping["0.06bf4d7c-9603-41a4-b250-284ecf3ecb2f.__stray"],
			expectedBatchJob: unitinputs.GetCleanupJob("node-2", "0.06bf4d7c-9603-41a4-b250-284ecf3ecb2f.__stray", "device-cleanup-job-551f6c87c7fe8e774164b810f1b16a17",
				map[string]string{"vdc": fmt.Sprintf(cleanupScriptTmpl,
					fmt.Sprintf(partitionCleanupScriptTmpl, "/dev/ceph-c5628abe-ae41-4c3d-bdc6-ef86c54bf78c/osd-block-69481cd1-38b1-42fd-ac07-06bf4d7c0e19", true,
						fmt.Sprintf(diskCleanupScriptTmpl, "/dev/disk/by-path/pci-0000:00:0c.0", true,
							fmt.Sprintf(hostDirectoryCleanupScriptTmpl, "/var/lib/rook/rook-ceph/8668f062-0lsk-358a-1gt4-f80fe6c1e306_06bf4d7c-9603-41a4-b250-284ecf3ecb2f")))),
				}),
		},
		{
			name:       "job created - only partition destroy and no host dir cleanup",
			taskConfig: taskConfigForTest,
			osd:        "4",
			host:       "node-2",
			osdMapping: lcmv1alpha1.OsdMapping{
				DeviceMapping: map[string]lcmv1alpha1.DeviceInfo{
					"/dev/vdd": {
						Rotational: false,
						Path:       "/dev/disk/by-path/pci-0000:00:1e.0",
						Partition:  "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde",
						Type:       "block",
						Alive:      true,
					},
				},
			},
			expectedBatchJob: func() *batch.Job {
				newJob := unitinputs.GetCleanupJob("node-2", "4", "", map[string]string{
					"vdd": fmt.Sprintf(cleanupScriptTmpl,
						fmt.Sprintf(partitionCleanupScriptTmpl, "/dev/ceph-dada9f25-41b4-4c26-9a20-448ac01e1d06/osd-block-ad76cf53-5cb5-48fe-a39a-343734f5ccde", true, "")),
				})
				newJob.Spec.Template.Spec.Volumes = newJob.Spec.Template.Spec.Volumes[:2]
				newJob.Spec.Template.Spec.Containers[0].VolumeMounts = newJob.Spec.Template.Spec.Containers[0].VolumeMounts[:2]
				return newJob
			}(),
		},
	}
	oldRetryTimeout := commandRetryRunTimeout
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commandRetryRunTimeout = 0
			lcmConfigData := map[string]string{}
			if test.removeAllLVMs {
				lcmConfigData["TASK_ALLOW_REMOVE_MANUALLY_CREATED_LVMS"] = "true"
			}
			c := fakeCephReconcileConfig(&test.taskConfig, lcmConfigData)

			inputRes := map[string]runtime.Object{"jobs": &batch.JobList{}}
			if test.batchJob != nil {
				inputRes["jobs"] = &batch.JobList{Items: []batch.Job{*test.batchJob}}
			}
			apiErrors := map[string]error{}
			if test.apiError != "" {
				apiErrors[test.apiError] = errors.New("failed to " + test.apiError)
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.BatchV1(), "get", []string{"jobs"}, inputRes, apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.BatchV1(), "create", []string{"jobs"}, inputRes, apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.BatchV1(), "delete", []string{"jobs"}, inputRes, apiErrors)

			jobName, err := c.runCleanupJob(test.host, test.osd, test.osdMapping.HostDirectory, test.osdMapping.DeviceMapping)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				expectedjobName := ""
				if test.osd != "" && test.host != "" {
					expectedjobName = fmt.Sprintf("device-cleanup-job-%s-%s", test.host, test.osd)
				}
				assert.Equal(t, expectedjobName, jobName)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, test.expectedBatchJob)
				assert.Equal(t, test.expectedBatchJob.Name, jobName)
				newJob, err := c.api.Kubeclientset.BatchV1().Jobs(test.expectedBatchJob.Namespace).Get(c.context, test.expectedBatchJob.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.Equal(t, test.expectedBatchJob, newJob)
			}
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.BatchV1())
		})
	}
	commandRetryRunTimeout = oldRetryTimeout
}
