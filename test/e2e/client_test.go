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
	"testing"
	"time"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

func TestCreateCephClient(t *testing.T) {
	t.Log("#### e2e test: create custom ceph client")
	defer f.SetupTeardown(t)()

	f.Step(t, "Update spec with new cephclient")
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	testClient := cephlcmv1alpha1.CephClient{
		ClientSpec: cephlcmv1alpha1.ClientSpec{
			Name: "test-e2e-client",
			Caps: map[string]string{"mon": "allow r, allow command \"osd blacklist\""},
		},
	}
	if len(cd.Spec.Clients) > 0 {
		cd.Spec.Clients = append(cd.Spec.Clients, testClient)
	} else {
		cd.Spec.Clients = []cephlcmv1alpha1.CephClient{testClient}
	}
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Wait for CephClient becomes ready")
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 10*time.Second, 5*time.Minute, true, func(_ context.Context) (bool, error) {
		cephclient, err := f.TF.ManagedCluster.GetCephClient("test-e2e-client")
		if err != nil {
			if k8serrors.IsNotFound(err) {
				// wait function returns 2 parameters: bool and error. bool indicates "is wait successfully finished", error is a error.
				// if wait function return false, nil then it will repeat this code after 10*time.Second sleep.
				// if wait function return true, nil then wait finished successfully and wait function ended
				// if wait function returns error then it will be interrupted immediately with error
				// if wait function spend 5*time.Minute w/o return true, nil then it will return timeout error
				return false, nil
			}
			f.TF.Log.Error().Err(err).Msg("failed to get cephclient test-e2e-client")
			return false, nil
		}
		return cephclient.Status.Phase == cephv1.ConditionReady, nil
	})
	if err != nil {
		t.Fatalf("cephclient ready status wait failed: %v", err)
	}

	f.Step(t, "Verify cephclient has an access to ceph cluster")
	_, err = f.TF.ManagedCluster.RunCephToolsCommand("ceph auth export client.test-e2e-client -o /etc/ceph/ceph.client.test-e2e-client.keyring")
	if err != nil {
		errMsg := "failed to export client.test-e2e-client keyring"
		t.Fatal(errors.Wrap(err, errMsg))
	}
	// here -n client.test-e2e-client means that our cephclient requesting ceph cluster status (-n, --name CLIENT_NAME)
	_, err = f.TF.ManagedCluster.RunCephToolsCommand("ceph status -n client.test-e2e-client")
	if err != nil {
		errMsg := "test-e2e-client cephclient failed to get ceph cluster status"
		t.Fatal(errors.Wrap(err, errMsg))
	}
	_, err = f.TF.ManagedCluster.RunCephToolsCommand("rm /etc/ceph/ceph.client.test-e2e-client.keyring")
	if err != nil {
		errMsg := "failed to remove test-e2e-client keyring file from rook-ceph-tools pod"
		t.Fatal(errors.Wrap(err, errMsg))
	}
	t.Logf("#### Test %s complete sucessfully", t.Name())
}
