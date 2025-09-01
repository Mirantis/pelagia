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

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/wait"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

func TestE2eConfig(t *testing.T) {
	defer f.SetupTeardown(t)()

	config := f.GetConfigForTestCase(t)
	assert.Equal(t, map[string]string{"fake": "test"}, config)
	t.Logf("Test successfully passed")
}

func TestGetCluster(t *testing.T) {
	t.Log("#### e2e test: test get ceph cluster")
	defer f.SetupTeardown(t)()

	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(cd)
	t.Logf("CephDeployment found: %v\n", string(data))

	cephCluster, err := f.TF.ManagedCluster.GetCephCluster(cd.Name)
	if err != nil {
		t.Fatal(err)
	}
	data, _ = json.Marshal(cephCluster)
	t.Logf("CephCluster found: %v\n", string(data))
	pods, err := f.TF.ManagedCluster.ListPods(f.TF.ManagedCluster.LcmConfig.RookNamespace, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Next pods in namespace rook-ceph found:\n")
	for _, pod := range pods {
		t.Logf("   pod.name = %s\n", pod.Name)
	}

	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 10*time.Second, 5*time.Minute, true, func(_ context.Context) (bool, error) {
		nodes, err := f.TF.ManagedCluster.ListNodes()
		if err != nil {
			t.Fatal(err)
		}
		foundMonRoles := true
		t.Logf("Next nodes found:\n")
		for _, node := range nodes {
			t.Logf("   node.name = %s\n", node.Name)
			for label := range node.Labels {
				if label == "ceph_role_mon" {
					enabled, err := strconv.ParseBool(node.Labels[label])
					t.Logf("       found label = %s equals %v\n", label, enabled)
					if err != nil {
						t.Log("unable to parse node ceph label", "node.name", node.Name)
						foundMonRoles = false
					}
					foundMonRoles = foundMonRoles && enabled
				}
			}
		}
		return foundMonRoles, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Test successfully passed")
}

func TestPrometheusMetrics(t *testing.T) {
	t.Log("#### e2e test: verify prometheus module is enabled and metrics are available")
	defer f.SetupTeardown(t)()

	f.Step(t, "Check prometheus module is enabled")
	stdout, err := f.TF.ManagedCluster.RunCephToolsCommand("ceph mgr module ls -f json")
	if err != nil {
		errMsg := "failed to list mgr modules"
		t.Fatalf("%v", errors.Wrap(err, errMsg))
	}

	mgrModules := lcmcommon.MgrModuleLs{}
	err = json.Unmarshal([]byte(stdout), &mgrModules)
	if err != nil {
		t.Fatalf("%v", errors.Wrap(err, "failed to parse output for 'ceph mgr module ls -f json' command"))
	}

	if !lcmcommon.Contains(append(mgrModules.AlwaysOn, mgrModules.Enabled...), "prometheus") {
		t.Fatal("Ceph mgr module 'prometheus' is not enabled in the cluster")
	}

	f.Step(t, "Obtain Ceph exposed metrics")
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 15*time.Minute, true, func(_ context.Context) (done bool, err error) {
		t.Logf("trying to get mgr metrics")
		stdout, err := f.TF.ManagedCluster.RunCephToolsCommand("curl --silent http://rook-ceph-mgr.rook-ceph.svc:9283/metrics")
		if err != nil {
			errMsg := "failed to get metrics"
			t.Logf("curl mgr metrics failed: %v", errMsg)
			return false, nil
		}
		metrics := []string{"ceph_osd_up{ceph_daemon=", "ceph_mon_quorum_status{ceph_daemon=", "ceph_mgr_status{ceph_daemon=", "ceph_mgr_module_status{name=\"prometheus\"} 1.0"}

		for _, metric := range metrics {
			if !strings.Contains(stdout, metric) {
				t.Logf("%v metric prefix is not found in exposed ones", metric)
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("failed to get metrics: %v", err)
	}

	t.Logf("Test %v successfully passed", t.Name())
}

func TestPGAutoscaler(t *testing.T) {
	t.Log("#### e2e test: verify pg_autoscaler is available")
	defer f.SetupTeardown(t)()

	f.Step(t, "Check pg_autoscaler module is enabled")
	stdout, err := f.TF.ManagedCluster.RunCephToolsCommand("ceph mgr module ls -f json")
	if err != nil {
		errMsg := "failed to list mgr modules"
		t.Fatalf("%v", errors.Wrap(err, errMsg))
	}

	mgrModules := lcmcommon.MgrModuleLs{}
	err = json.Unmarshal([]byte(stdout), &mgrModules)
	if err != nil {
		t.Fatalf("%v", errors.Wrap(err, "failed to parse output for 'ceph mgr module ls -f json' command"))
	}

	if !lcmcommon.Contains(append(mgrModules.AlwaysOn, mgrModules.Enabled...), "pg_autoscaler") {
		t.Fatal("Ceph mgr module 'pg_autoscaler' is not enabled in the cluster")
	}

	f.Step(t, "Verify autoscale status is available")
	stdout, err = f.TF.ManagedCluster.RunCephToolsCommand("ceph osd pool autoscale-status")
	if err != nil {
		errMsg := "failed to get autoscale-status"
		t.Fatalf("%v", errors.Wrap(err, errMsg))
	}

	if strings.Trim(stdout, " \n") == "" {
		t.Fatal("Autoscale status is empty")
	}

	t.Logf("Test %v successfully passed", t.Name())
}

func TestValidationFailure(t *testing.T) {
	t.Log("#### e2e test: test ceph cluster spec validation failures")
	defer f.SetupTeardown(t)()

	f.Step(t, "get current spec and generation")
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	curGeneration := cd.Status.Validation.LastValidatedGeneration

	f.Step(t, "generate incorrect cluster spec")
	// network validation
	cd.Spec.Network.PublicNet = ""
	cd.Spec.Network.ClusterNet = "0.0.0.0/0"
	// pools validation
	poolName := "test-pool-invalid-" + fmt.Sprintf("%d", time.Now().Unix())
	cd.Spec.Pools = append(cd.Spec.Pools, cephlcmv1alpha1.CephPool{
		Name: poolName,
		StorageClassOpts: cephlcmv1alpha1.CephStorageClassSpec{
			ReclaimPolicy: "Fake",
		},
	})
	// nodes validation
	monCnt := 0
	for _, nodeSpec := range cd.Spec.Nodes {
		if lcmcommon.Contains(nodeSpec.Roles, "mon") {
			monCnt++
		}
	}
	nodeCnt := 0
	nodeNameToCheck := ""
	deviceNameToCheck := ""
	for idx, nodeSpec := range cd.Spec.Nodes {
		if nodeCnt == 0 && !lcmcommon.Contains(nodeSpec.Roles, "mon") {
			newNodeSpec := nodeSpec.DeepCopy()
			newNodeSpec.Roles = append(newNodeSpec.Roles, "mon")
			cd.Spec.Nodes[idx] = *newNodeSpec
			monCnt++
			nodeCnt++
			continue
		}
		if nodeCnt == 1 && len(nodeSpec.Devices) > 0 {
			nodeNameToCheck = nodeSpec.Name
			newNodeSpec := nodeSpec.DeepCopy()
			if newNodeSpec.Crush == nil {
				newNodeSpec.Crush = map[string]string{"datcentr": "fake"}
			} else {
				newNodeSpec.Crush["datcentr"] = "fake"
			}
			newNodeSpec.Devices[0].Config["osdsPerDevice"] = "fake"
			deviceNameToCheck = newNodeSpec.Devices[0].Name
			cd.Spec.Nodes[idx] = *newNodeSpec
			nodeCnt++
			break
		}
	}
	if nodeCnt != 2 {
		t.Skip("failed to update cephdeployment nodes section incorrectly - not all required nodes items found")
	}
	// rgw validation
	if cd.Spec.ObjectStorage != nil {
		if cd.Spec.ObjectStorage.Rgw.DataPool.Replicated != nil {
			cd.Spec.ObjectStorage.Rgw.DataPool.ErasureCoded = &cephlcmv1alpha1.CephPoolErasureCodedSpec{}
		} else if cd.Spec.ObjectStorage.Rgw.DataPool.ErasureCoded != nil {
			cd.Spec.ObjectStorage.Rgw.DataPool.Replicated = &cephlcmv1alpha1.CephPoolReplicatedSpec{}
		}
		cd.Spec.ObjectStorage.Rgw.MetadataPool.Replicated = nil
		cd.Spec.ObjectStorage.Rgw.MetadataPool.ErasureCoded = nil
	}
	// cephfs validation
	cd.Spec.SharedFilesystem = &cephlcmv1alpha1.CephSharedFilesystem{
		CephFS: []cephlcmv1alpha1.CephFS{
			{
				Name:         "fake",
				MetadataPool: cephlcmv1alpha1.CephPoolSpec{},
				DataPools: []cephlcmv1alpha1.CephFSPool{
					{
						Name: "fake-datapool-1",
						CephPoolSpec: cephlcmv1alpha1.CephPoolSpec{
							DeviceClass: "hdd",
							ErasureCoded: &cephlcmv1alpha1.CephPoolErasureCodedSpec{
								CodingChunks: 2,
								DataChunks:   1,
							},
						},
					},
					{
						Name: "fake-datapool-2",
						CephPoolSpec: cephlcmv1alpha1.CephPoolSpec{
							DeviceClass: "hdd",
						},
					},
				},
			},
		},
	}

	f.Step(t, "update cephdeployment with incorrectly generated spec")
	err = f.UpdateCephDeploymentSpec(cd, false)
	if err != nil {
		t.Fatal(err)
	}
	result := ""
	messages := []string{}
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 15*time.Minute, true, func(_ context.Context) (done bool, err error) {
		cdCur, err := f.TF.ManagedCluster.GetCephDeployment(cd.Name)
		if err != nil {
			f.TF.Log.Error().Err(err).Msg("")
			return false, nil
		}
		newGeneration := cdCur.Status.Validation.LastValidatedGeneration
		result = string(cdCur.Status.Validation.Result)
		messages = cdCur.Status.Validation.Messages
		return newGeneration > curGeneration, nil
	})
	if err != nil {
		t.Fatalf("failed to wait for new generation of validation result: %v", err)
	}

	t.Logf("Validation result is '%s' with messages:\n%v", result, strings.Join(messages, "\n   "))
	if result != "Failed" {
		t.Fatalf("validation result expected is 'Failed', actual is '%s'", result)
	}
	expectedMsg := []string{
		fmt.Sprintf("CephDeployment pool %s has no deviceClass specified (valid options are: [hdd nvme ssd])", poolName),
		fmt.Sprintf("CephDeployment pool %s spec should contain either replicated or erasureCoded spec", poolName),
		fmt.Sprintf("CephDeployment pool %s spec contains invalid reclaimPolicy 'Fake', valid are: %v", poolName, []string{"Retain", "Delete"}),
		fmt.Sprintf("CephDeployment node spec for node '%s' contains invalid crush topology key 'datcentr'. Valid are: chassis, datacenter, pdu, rack, region, room, row, zone", nodeNameToCheck),
		fmt.Sprintf("failed to parse config parameter 'osdsPerDevice' for device '%s' from node '%s'", deviceNameToCheck, nodeNameToCheck),
		fmt.Sprintf("CephDeployment monitors (roles 'mon') count %d is even, but should be odd for a healthy quorum", monCnt),
		"network clusterNet parameter contains prohibited 0.0.0.0 range",
		"network publicNet parameter is empty",
		"metadataPool for CephFS rook-ceph/fake must use replication only",
		"metadataPool for CephFS rook-ceph/fake has no deviceClass specified (valid options are: [hdd nvme ssd])",
		"dataPool fake-datapool-1 will be used as default for CephFS rook-ceph/fake and must use replication only",
		"dataPool fake-datapool-2 for CephFS rook-ceph/fake has no neither replication or erasureCoded sections specified",
	}
	if cd.Spec.ObjectStorage != nil {
		expectedMsg = append(expectedMsg, "ObjectStorage section is incorrect: rgw metadata pool must be only replicated,rgw data pool must have only one pool type specified")
	}
	for _, expected := range expectedMsg {
		found := false
		for _, actual := range messages {
			if strings.Contains(actual, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("failed to find '%s' message in validation error messages", expected)
		}
	}

	t.Logf("Test successfully passed")
}

func TestAllOsdRestarted(t *testing.T) {
	t.Logf("e2e test: check all osd are restarted")
	defer f.SetupTeardown(t)()

	osdRestartReason := fmt.Sprintf("unit test osd restart time-%d", time.Now().Unix())
	annotation := "cephdeployment.lcm.mirantis.com/restart-osd-reason"

	f.Step(t, "Update ceph spec with osdRestartReason")
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	if cd.Spec.ExtraOpts == nil {
		cd.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{}
	}
	cd.Spec.ExtraOpts.OsdRestartReason = osdRestartReason
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "check osd deployments with annotation")
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 15*time.Second, 15*time.Minute, true, func(_ context.Context) (bool, error) {
		deploys, err := f.TF.ManagedCluster.ListDeployments(f.TF.ManagedCluster.LcmConfig.RookNamespace, "app=rook-ceph-osd")
		if err != nil {
			t.Logf("failed to get osd deployments: %v", err)
			return false, err
		}
		if len(deploys.Items) == 0 {
			t.Log("failed to find osd deployments (no deployments with label 'app=rook-ceph-osd' in 'rook-ceph' namespace")
			return false, err
		}
		notUpdatedDeploys := 0
		for _, deploy := range deploys.Items {
			if value, ok := deploy.Annotations[annotation]; ok {
				if value == osdRestartReason {
					continue
				}
				t.Logf("deployment '%s/%s' has unexpected annotation set '%s=%s', expected value '%s', waiting...",
					deploy.Namespace, deploy.Name, annotation, value, osdRestartReason)
			} else {
				t.Logf("deployment '%s/%s' has no annotation '%s', waiting...", deploy.Namespace, deploy.Name, annotation)
			}
			notUpdatedDeploys++
		}
		if notUpdatedDeploys == 0 {
			return true, nil
		}
		t.Logf("not all deployments has restart annotation")
		return false, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("All osd are restarted, test successfully passed")
}
