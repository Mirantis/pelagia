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
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

func (c *ManagedConfig) GetCephDeploymentSecret(name string) (*cephlcmv1alpha1.CephDeploymentSecret, error) {
	cephDplSecret, err := c.CephDplSecretClient.Get(c.Context, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get CephDeploymentSecret '%s/%s'", c.LcmNamespace, name)
	}
	return cephDplSecret, nil
}

func (c *ManagedConfig) GetCephDeploymentHealth(clusterName string) (*cephlcmv1alpha1.CephDeploymentHealth, error) {
	cephHealth, err := c.CephHealthClient.Get(c.Context, clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get CephDeploymentHealth '%s/%s'", c.LcmNamespace, clusterName)
	}
	return cephHealth, nil
}

func (c *ManagedConfig) WaitForCephDeploymentHealthReady(clusterName string) error {
	err := wait.PollUntilContextTimeout(c.Context, 10*time.Second, 30*time.Minute, true, func(_ context.Context) (bool, error) {
		cephHealth, err := c.GetCephDeploymentHealth(clusterName)
		if err != nil {
			TF.Log.Error().Err(err).Msg("")
			return false, nil
		}
		if cephHealth.Status.State != cephlcmv1alpha1.HealthStateOk {
			TF.Log.Error().Msgf("Waiting CephDeploymentHealth %s/%s becomes ready: %v", cephHealth.Namespace, cephHealth.Name, cephHealth.Status.Issues)
			return false, nil
		}
		cephCluster, err := c.GetCephCluster(clusterName)
		if err != nil {
			TF.Log.Error().Err(err).Msg("")
			return false, nil
		}
		if !cephCluster.Spec.External.Enable {
			if *cephHealth.Status.HealthReport.OsdAnalysis.CephClusterSpecGeneration != cephCluster.GetGeneration() {
				TF.Log.Error().Msgf("CephDeploymentHealth %s/%s is not validated last CephCluster version (%d)", cephHealth.Namespace, cephHealth.Name, cephCluster.GetGeneration())
				return false, nil
			}
		}
		return true, nil
	})
	return err
}
