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
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmclient "github.com/Mirantis/pelagia/pkg/client/clientset/versioned"
)

func IsClusterMaintenanceActing(ctx context.Context, cephLcmclientset lcmclient.Interface, namespace, name string) (bool, error) {
	cdm, err := cephLcmclientset.LcmV1alpha1().CephDeploymentMaintenances(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get CephDeploymentMaintenance %s/%s", namespace, name)
	}
	return cdm.Status != nil && (cdm.Status.State == cephlcmv1alpha1.MaintenanceActing || cdm.Status.State == cephlcmv1alpha1.MaintenanceFailing), nil
}

func IsOpenStackPoolsPresent(pools []cephlcmv1alpha1.CephPool) bool {
	// since on validation stage we are checking that pools section is correct
	// we can just simply check any openstack pool existence now
	expectedRoles := []string{"images", "vms", "backup", "volumes"}
	for _, pool := range pools {
		if Contains(expectedRoles, pool.Role) {
			return true
		}
	}
	return false
}
