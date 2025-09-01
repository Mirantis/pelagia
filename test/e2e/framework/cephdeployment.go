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

package framework

import (
	"context"
	"reflect"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *ManagedConfig) FindCephDeployment() (*cephlcmv1alpha1.CephDeployment, error) {
	cephdpls, err := c.ListCephDeployment()
	if err != nil {
		return nil, err
	}
	if len(cephdpls) != 1 {
		return nil, errors.Errorf("wrong number of CephDeployments in cluster (namespace '%s'): expected = 1, found = %d", c.LcmNamespace, len(cephdpls))
	}
	cephdpl := cephdpls[0]
	return &cephdpl, nil
}

func (c *ManagedConfig) ListCephDeployment() ([]cephlcmv1alpha1.CephDeployment, error) {
	cephDeploymentList, err := c.CephDplClient.List(c.Context, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list CephDeployments in namespace '%s'", c.LcmNamespace)
	}
	return cephDeploymentList.Items, nil
}

func (c *ManagedConfig) CreateCephDeployment(cephdpl *cephlcmv1alpha1.CephDeployment) error {
	_, err := c.CephDplClient.Create(c.Context, cephdpl, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to create CephDeployment %s/%s", cephdpl.Namespace, cephdpl.Name)
	}
	return nil
}

func (c *ManagedConfig) GetCephDeployment(cdName string) (*cephlcmv1alpha1.CephDeployment, error) {
	cephdpl, err := c.CephDplClient.Get(c.Context, cdName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get '%s/%s' CephDeployment", c.LcmNamespace, cdName)
	}
	return cephdpl, nil
}

func (c *ManagedConfig) UpdateCephDeploymentSpec(cephDpl *cephlcmv1alpha1.CephDeployment) (bool, error) {
	TF.Log.Info().Msgf("Trying to update CephDeployment '%s/%s'", cephDpl.Namespace, cephDpl.Name)
	updated := false
	err := wait.PollUntilContextTimeout(c.Context, 10*time.Second, 15*time.Minute, true, func(_ context.Context) (bool, error) {
		realCephDpl, getErr := c.GetCephDeployment(cephDpl.Name)
		if getErr != nil {
			TF.Log.Error().Err(getErr).Msg("")
			return false, nil
		}
		TF.PreviousClusterState.CephDeployment = realCephDpl.DeepCopy()
		if !reflect.DeepEqual(realCephDpl.Spec, cephDpl.Spec) {
			TF.Log.Info().Msg("spec changed, updating CephDeployment resource")
			lcmcommon.ShowObjectDiff(TF.Log, realCephDpl.Spec, cephDpl.Spec)
			realCephDpl.Spec = cephDpl.Spec
			_, updateErr := c.CephDplClient.Update(context.Background(), realCephDpl, metav1.UpdateOptions{})
			if updateErr != nil {
				TF.Log.Error().Err(updateErr).Msg("")
				return false, nil
			}
			updated = true
		} else {
			TF.Log.Info().Msgf("Updating CephDeployment '%s/%s' aborted, no changes", cephDpl.Namespace, cephDpl.Name)
		}
		return true, nil
	})
	if err != nil {
		return false, errors.Wrapf(err, "failed to update CephDeployment %s/%s", cephDpl.Namespace, cephDpl.Name)
	}
	if updated {
		// sleep some time to give time start new reconciling for rook and miracephhealth
		time.Sleep(30 * time.Second)
	}
	return updated, nil
}

func (c *ManagedConfig) WaitForCephDeploymentReady(cdName string) error {
	TF.Log.Info().Msgf("Waiting CephDeployment %s/%s health state", c.LcmNamespace, cdName)
	err := wait.PollUntilContextTimeout(c.Context, 15*time.Second, 30*time.Minute, true, func(_ context.Context) (bool, error) {
		cephDpl, err := c.GetCephDeployment(cdName)
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to get CephDeployment '%s/%s'", c.LcmNamespace, cdName)
			return false, nil
		}
		if cephDpl.Status.Phase == cephlcmv1alpha1.PhaseReady && cephDpl.Status.Validation.LastValidatedGeneration == cephDpl.Generation {
			return true, nil
		}
		TF.Log.Info().Msgf("CephDeployment '%s/%s' spec is not ready yet", cephDpl.Namespace, cephDpl.Name)
		return false, nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to wait CephDeployment '%s/%s' ready", c.LcmNamespace, cdName)
	}
	return nil
}
