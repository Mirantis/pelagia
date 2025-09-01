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
	"testing"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"

	cephlcmv1alpha "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

func TestUpdateStorageNodes(t *testing.T) {
	t.Log("e2e test: ceph extend storage osd(s) on nodes cluster")
	err := f.BaseSetup(t)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("read provided test update config")
	testConfig := f.GetConfigForTestCase(t)
	var extendNodes, reduceNodes, updateNodes, dropNodes map[string]cephv1.Node
	if config, ok := testConfig["extendNodes"]; ok {
		extendNodes, err = f.ReadNodeStorageConfig(config)
		if err != nil {
			t.Fatal(err)
		}
	}
	if config, ok := testConfig["reduceNodes"]; ok {
		reduceNodes, err = f.ReadNodeStorageConfig(config)
		if err != nil {
			t.Fatal(err)
		}
	}
	if config, ok := testConfig["updateNodes"]; ok {
		updateNodes, err = f.ReadNodeStorageConfig(config)
		if err != nil {
			t.Fatal(err)
		}
	}
	if config, ok := testConfig["dropNodes"]; ok {
		dropNodes, err = f.ReadNodeStorageConfig(config)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(extendNodes) == 0 && len(updateNodes) == 0 && len(dropNodes) == 0 && len(reduceNodes) == 0 {
		t.Fatal("either one test config value should be provided: extendNodes, reduceNodes, updateNodes, dropNodes")
	}
	waitReadiness := true
	if v, ok := testConfig["waitReadiness"]; ok && v != "yes" {
		waitReadiness = false
	}

	t.Log("update nodes in CephDeployment")
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	newSpecNodes := []cephlcmv1alpha.CephDeploymentNode{}
	for _, cephNode := range cd.Spec.Nodes {
		if _, ok := dropNodes[cephNode.Name]; ok {
			continue
		}
		if updatedNode, ok := updateNodes[cephNode.Name]; ok {
			newNode := cephNode.DeepCopy()
			newNode.Config = updatedNode.Config
			newNode.Devices = updatedNode.Devices
			newNode.DeviceFilter = updatedNode.DeviceFilter
			newNode.DevicePathFilter = updatedNode.DevicePathFilter
			newSpecNodes = append(newSpecNodes, *newNode)
			// delete from update, if any new nodes
			delete(updateNodes, cephNode.Name)
			continue
		}
		if extendNode, ok := extendNodes[cephNode.Name]; ok {
			if len(cephNode.Devices) > 0 {
				newNode := cephNode.DeepCopy()
				newNode.Devices = append(newNode.Devices, extendNode.Devices...)
				newSpecNodes = append(newSpecNodes, *newNode)
			}
			continue
		}
		if reduceNode, ok := reduceNodes[cephNode.Name]; ok {
			if len(cephNode.Devices) > 0 {
				newNode := cephNode.DeepCopy()
				newDevices := []cephv1.Device{}
				for _, dev := range cephNode.Devices {
					keep := true
					for _, devToRemove := range reduceNode.Devices {
						if dev.Name == devToRemove.Name && dev.Name != "" || dev.FullPath != "" && dev.FullPath == devToRemove.FullPath {
							keep = false
							break
						}
					}
					if keep {
						newDevices = append(newDevices, dev)
					}
				}
				newNode.Devices = newDevices
				newSpecNodes = append(newSpecNodes, *newNode)
			}
			continue
		}
		newSpecNodes = append(newSpecNodes, cephNode)
	}
	for nodeNameToUpdate, newNodeConf := range updateNodes {
		newOsdNode := cephlcmv1alpha.CephDeploymentNode{
			Node: cephv1.Node{
				Name:   nodeNameToUpdate,
				Config: newNodeConf.Config,
				Selection: cephv1.Selection{
					Devices:          newNodeConf.Devices,
					DeviceFilter:     newNodeConf.DeviceFilter,
					DevicePathFilter: newNodeConf.DevicePathFilter,
				},
			},
			Roles: make([]string, 0),
		}
		newSpecNodes = append(newSpecNodes, newOsdNode)
	}
	cd.Spec.Nodes = newSpecNodes

	err = f.UpdateCephDeploymentSpec(cd, waitReadiness)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("e2e test: ceph osd(s) update completed sucessfully")
}

func prepareStray(osdID string) error {
	f.TF.Log.Info().Msgf("# Creating dummy osd in crush with id '%s' and uuid 'ef5ab081-8f51-4926-a5ac-67797ac98da9'", osdID)
	stdOut, err := f.TF.ManagedCluster.RunCephToolsCommand("ceph osd new ef5ab081-8f51-4926-a5ac-67797ac98da9 " + osdID)
	f.TF.Log.Info().Msg(stdOut)
	return err
}

func TestCephOsdRemoveTask(t *testing.T) {
	t.Log("e2e test: ceph osd remove task to remove osd from cluster")
	err := f.BaseSetup(t)
	if err != nil {
		t.Fatal(err)
	}
	testConfig := f.GetConfigForTestCase(t)
	tasksStr, present := testConfig["tasks"]
	if !present {
		t.Fatal("Test config does not contain 'tasks' configuration")
	}
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("# Prepare ceph osd remove task")
	tasks, err := f.GenerateOsdRemoveTasks(tasksStr)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) == 0 {
		t.Fatal("No valid tasks provided in test config")
	}
	if id, ok := testConfig["strayPrepare"]; ok {
		err = prepareStray(id)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Log("# Run and wait ceph osd remove tasks")
	for _, task := range tasks {
		f.TF.Log.Info().Msgf("# run ceph osd remove task %s/%s", task.Namespace, task.Name)
		err := f.TF.ManagedCluster.CreateAndWaitOsdRemoveTask(task)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Log("# Wait for overall Ready status")
	err = f.WaitForStatusReady(cd.Name)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("e2e test: ceph osd remove tasks completed sucessfully")
}
