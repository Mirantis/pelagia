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
	"testing"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

func TestAddCustomDeviceClass(t *testing.T) {
	t.Log("#### e2e test: add custom device class")

	err := f.BaseSetup(t)
	if err != nil {
		t.Fatal(err)
	}

	testConfig := f.GetConfigForTestCase(t)
	if _, ok := testConfig["deviceClass"]; !ok {
		t.Fatal("test config does not contain 'deviceClass' key")
	}

	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	if cd.Spec.ExtraOpts == nil {
		cd.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{}
	}
	cd.Spec.ExtraOpts.CustomDeviceClasses = append(cd.Spec.ExtraOpts.CustomDeviceClasses, testConfig["deviceClass"])
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Test %v successfully passed", t.Name())
}

func TestVerifyCustomDeviceClass(t *testing.T) {
	t.Log("#### e2e test: create pool on custom device class and verify it is separate from main storage")
	defer f.SetupTeardown(t)()

	poolName := fmt.Sprintf("test-pool-custom-class-%d", time.Now().Unix())
	imageName := fmt.Sprintf("test-image-custom-class-%d", time.Now().Unix())

	testConfig := f.GetConfigForTestCase(t)
	if _, ok := testConfig["deviceClass"]; !ok {
		t.Fatal("test config does not contain 'deviceClass' key")
	}

	f.Step(t, "Create Pool on custom device class")
	newPool := f.GetNewPool(poolName, true, false, 2, "", "", testConfig["deviceClass"])
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	cd.Spec.Pools = append(cd.Spec.Pools, newPool)
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Obtain current usage by device class")
	var cephDetails lcmcommon.CephDetails
	stdout, err := f.TF.ManagedCluster.RunCephToolsCommand("ceph df --format json")
	if err != nil {
		errMsg := "failed to get 'ceph df --format json' output"
		t.Fatalf("%s: %v", errMsg, err)
	}
	err = json.Unmarshal([]byte(stdout), &cephDetails)
	if err != nil {
		t.Fatalf("failed to parse 'ceph df --format json' output: %v", err)
	}

	for class, stats := range cephDetails.StatsByClass {
		t.Logf("Used bytes size for '%s' device class is: %v", class, stats.UsedBytes)
	}

	f.Step(t, "Create rbd image in pool '%s'", poolName)
	_, err = f.TF.ManagedCluster.RunCephToolsCommand(fmt.Sprintf("rbd create %s/%s --size 157M --thick-provision", poolName, imageName))
	if err != nil {
		errMsg := "failed to create rbd image to test pool with custom device class"
		t.Fatalf("%s: %v", errMsg, err)
	}

	defer func() {
		t.Log("Cleaning rbd image")
		_, err = f.TF.ManagedCluster.RunCephToolsCommand(fmt.Sprintf("rbd rm %s/%s", poolName, imageName))
		if err != nil {
			t.Fatalf("failed to remove rbd image from test pool with custom device class: %v", err)
		}
	}()

	f.Step(t, "Verify only '%s' class used stats changed", testConfig["deviceClass"])
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 15*time.Second, 3*time.Minute, true, func(_ context.Context) (done bool, err error) {
		t.Log("Attempting compare used storage with new rbd image on custom device class")
		var newCephDetails lcmcommon.CephDetails
		stdout, err = f.TF.ManagedCluster.RunCephToolsCommand("ceph df --format json")
		if err != nil {
			errMsg := "failed to get 'ceph df --format json' output"
			t.Log(errors.Wrap(err, errMsg))
			return false, nil
		}
		err = json.Unmarshal([]byte(stdout), &newCephDetails)
		if err != nil {
			t.Logf("failed to parse 'ceph df --format json' output: %v", err)
			return false, nil
		}
		for class, stats := range cephDetails.StatsByClass {
			t.Logf("Used bytes size for '%s' device class is: %v", class, stats.UsedBytes)
		}

		// 157MB in bytes
		expectedIncrease := uint64(164626432)
		// 50MB in bytes
		acceptableDelta := uint64(52428800)
		for class, stats := range newCephDetails.StatsByClass {
			if class == testConfig["deviceClass"] {
				if stats.UsedBytes < cephDetails.StatsByClass[class].UsedBytes+expectedIncrease {
					t.Logf("class '%s' stats didn't change to expected 157MB increase: expected=%v, actual=%v",
						class, cephDetails.StatsByClass[class].UsedBytes+expectedIncrease, stats.UsedBytes)
					return false, nil
				}
				t.Logf("class '%s' used stats expectedly changed to 157MB increase: %v", class, stats.UsedBytes)
			} else {
				if stats.UsedBytes > cephDetails.StatsByClass[class].UsedBytes+acceptableDelta ||
					stats.UsedBytes < cephDetails.StatsByClass[class].UsedBytes-acceptableDelta {
					t.Logf("class '%s' stats unexpectedly changed more than acceptable delta: expected=%v, actual=%v",
						class, cephDetails.StatsByClass[class].UsedBytes, stats.UsedBytes)
					return false, nil
				}
				t.Logf("class '%s' used stats expectedly not changed: %v", class, stats.UsedBytes)
			}
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("failed to wait for used bytes changes: %v", err)
	}

	t.Logf("Test %v successfully passed", t.Name())
}

func TestRemoveCustomDeviceClass(t *testing.T) {
	t.Log("#### e2e test: remove custom device class")

	err := f.BaseSetup(t)
	if err != nil {
		t.Fatal(err)
	}

	testConfig := f.GetConfigForTestCase(t)
	if _, ok := testConfig["deviceClass"]; !ok {
		t.Fatal("test config does not contain 'deviceClass' key")
	}

	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	if cd.Spec.ExtraOpts != nil && len(cd.Spec.ExtraOpts.CustomDeviceClasses) > 0 {
		newCustomDeviceClasses := make([]string, 0)
		for idx, class := range cd.Spec.ExtraOpts.CustomDeviceClasses {
			if class == testConfig["deviceClass"] {
				newCustomDeviceClasses = append(
					cd.Spec.ExtraOpts.CustomDeviceClasses[:idx],
					cd.Spec.ExtraOpts.CustomDeviceClasses[idx+1:]...)
				break
			}
		}
		cd.Spec.ExtraOpts.CustomDeviceClasses = newCustomDeviceClasses
		err = f.UpdateCephDeploymentSpec(cd, true)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("Test %v successfully passed", t.Name())
}
