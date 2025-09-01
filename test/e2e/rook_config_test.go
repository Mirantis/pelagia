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
	"fmt"
	"strings"
	"testing"

	"github.com/go-yaml/yaml"
	"github.com/pkg/errors"

	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

func verifyCephAvailable(t *testing.T) []error {
	errs := []error{}

	f.Step(t, "Executing 'ceph -s' command")
	stdout, err := f.TF.ManagedCluster.RunCephToolsCommand("ceph -s")
	if err != nil {
		errMsg := "failed to get ceph status"
		errs = append(errs, errors.Wrap(err, errMsg))
	}
	t.Logf("Ceph status is: %v", stdout)

	f.Step(t, "Executing 'ceph health detail' and check it is HEALTH_OK")
	stdout, err = f.TF.ManagedCluster.RunCephToolsCommand("ceph health detail")
	if err != nil {
		errMsg := "failed to get ceph status"
		errs = append(errs, errors.Wrap(err, errMsg))
	}
	if !strings.Contains(stdout, "HEALTH_OK") {
		errs = append(errs, errors.Errorf("ceph health details expects HEALTH_OK but found: %v", stdout))
	}
	f.Step(t, "Test radosgw-admin CLI is available")
	_, err = f.TF.ManagedCluster.RunCephToolsCommand("radosgw-admin user list")
	if err != nil {
		errMsg := "failed to get RGW user list"
		errs = append(errs, errors.Wrap(err, errMsg))
	}
	if len(errs) > 0 {
		return errs
	}

	f.Step(t, "Create rbd image")
	poolToTest := "kubernetes-hdd"
	imageToTest := "rookconfigtest"
	_, err = f.TF.ManagedCluster.RunCephToolsCommand(fmt.Sprintf("rbd create %s/%s --size 10G", poolToTest, imageToTest))
	if err != nil {
		return []error{errors.Wrapf(err, "failed to create fake image to test rook config")}
	}
	defer func() {
		f.Step(t, "Cleaning rbd image")
		_, err = f.TF.ManagedCluster.RunCephToolsCommand(fmt.Sprintf("rbd rm %s/%s", poolToTest, imageToTest))
		if err != nil {
			t.Log(errors.Wrap(err, "failed to remove fake image to test rook config"))
		}
	}()
	f.Step(t, "Export created rbd image and check its size")
	_, err = f.TF.ManagedCluster.RunCephToolsCommand(fmt.Sprintf("rbd export %s/%s /tmp/file", poolToTest, imageToTest))
	if err != nil {
		return []error{errors.Wrap(err, "failed to export fake image to test rook config")}
	}
	defer func() {
		f.Step(t, "Cleaning image exported in test file")
		_, err = f.TF.ManagedCluster.RunCephToolsCommand("rm /tmp/file")
		if err != nil {
			t.Log(errors.Wrap(err, "failed to remove exported rbd image file"))
		}
	}()
	stdout, err = f.TF.ManagedCluster.RunCephToolsCommand("stat --format %s /tmp/file")
	if err != nil {
		return []error{errors.Wrap(err, "failed to get exported fake image size to test rook config")}
	}
	if strings.Trim(stdout, " \"\n") != "10737418240" {
		return []error{errors.Errorf("exported rbd image expected size 10737418240, but actual size is '%v'", strings.Trim(stdout, " \"\n"))}
	}
	return nil
}

func TestAddRookConfigOptions(t *testing.T) {
	t.Log("#### e2e test: add rookConfig options to a cluster")
	err := f.BaseSetup(t)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "get rookConfig for test scenario")
	testConfig := f.GetConfigForTestCase(t)
	if _, ok := testConfig["rookConfig"]; !ok {
		t.Fatal("Test config does not contain 'rookConfig' key")
	}

	var rookConfig map[string]string
	err = yaml.Unmarshal([]byte(testConfig["rookConfig"]), &rookConfig)
	if err != nil {
		t.Fatalf("unable to unmarshal rookConfig key from test config: %v", err)
	}

	f.Step(t, "update ceph spec with rookConfig section and wait for ready status")
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range rookConfig {
		if cd.Spec.RookConfig == nil {
			cd.Spec.RookConfig = map[string]string{}
		}
		if _, ok := cd.Spec.RookConfig[k]; !ok {
			cd.Spec.RookConfig[k] = v
		}
	}
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "verify ceph cluster is available after rookConfig applied")
	errs := verifyCephAvailable(t)
	if len(errs) > 0 {
		errCollect := []string{}
		for _, verifyErr := range errs {
			errCollect = append(errCollect, verifyErr.Error())
		}
		t.Fatalf("ceph cluster CLI has following issues:\n%v", strings.Join(errCollect, "\n"))
	}

	t.Logf("#### Test %s complete successfully", t.Name())
}

func TestRemoveRookConfigOptions(t *testing.T) {
	t.Log("#### e2e test: remove rookConfig options from a cluster")
	err := f.BaseSetup(t)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "get rookConfig for test scenario")
	testConfig := f.GetConfigForTestCase(t)
	if _, ok := testConfig["rookConfig"]; !ok {
		t.Fatal("Test config does not contain 'rookConfig' key")
	}

	var rookConfig map[string]string
	err = yaml.Unmarshal([]byte(testConfig["rookConfig"]), &rookConfig)
	if err != nil {
		t.Fatalf("unable to unmarshal rookConfig key from test config: %v", err)
	}

	f.Step(t, "remove desired rookConfig options from spec and wait for ready status")
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range rookConfig {
		if v, ok := cd.Spec.RookConfig[key]; ok && v == value {
			delete(cd.Spec.RookConfig, key)
		}
	}
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "verify ceph cluster is available after rookConfig options removed")
	errs := verifyCephAvailable(t)
	if len(errs) > 0 {
		errCollect := []string{}
		for _, verifyErr := range errs {
			errCollect = append(errCollect, verifyErr.Error())
		}
		t.Fatalf("ceph cluster CLI has following issues:\n%v", strings.Join(errCollect, "\n"))
	}

	t.Logf("#### Test %s complete successfully", t.Name())
}
