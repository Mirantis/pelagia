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
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func generatePoolSpec(poolSpec *cephlcmv1alpha1.CephPoolSpec, role string) (newpool *cephv1.PoolSpec) {
	poolSpecResource := cephv1.PoolSpec{
		FailureDomain: poolSpec.FailureDomain,
		CrushRoot:     poolSpec.CrushRoot,
		DeviceClass:   poolSpec.DeviceClass,
	}
	if poolSpec.Replicated != nil {
		poolSpecResource.Replicated = cephv1.ReplicatedSpec{
			Size: poolSpec.Replicated.Size,
		}
		if poolSpec.Replicated.TargetSizeRatio == 0 {
			poolSpecResource.Replicated.TargetSizeRatio = poolsDefaultTargetSizeRatioByRole(role)
		} else {
			poolSpecResource.Replicated.TargetSizeRatio = poolSpec.Replicated.TargetSizeRatio
		}
	}
	if poolSpec.ErasureCoded != nil {
		poolSpecResource.ErasureCoded = cephv1.ErasureCodedSpec{
			CodingChunks: poolSpec.ErasureCoded.CodingChunks,
			DataChunks:   poolSpec.ErasureCoded.DataChunks,
			Algorithm:    poolSpec.ErasureCoded.Algorithm,
		}
	}
	if poolSpec.Mirroring != nil {
		switch poolSpec.Mirroring.Mode {
		case "pool", "image":
			poolSpecResource.Mirroring = cephv1.MirroringSpec{
				Enabled: true,
				Mode:    poolSpec.Mirroring.Mode,
			}
		}
	}
	if len(poolSpec.Parameters) > 0 {
		poolSpecResource.Parameters = poolSpec.Parameters
	}
	poolSpecResource.EnableCrushUpdates = poolSpec.EnableCrushUpdates
	return &poolSpecResource
}

func generatePool(pool cephlcmv1alpha1.CephPool, namespace string) (newpool *cephv1.CephBlockPool) {
	cephpool := cephv1.CephBlockPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildPoolName(pool),
			Namespace: namespace,
		},
	}
	cephpool.Spec = cephv1.NamedBlockPoolSpec{PoolSpec: *generatePoolSpec(&pool.CephPoolSpec, pool.Role)}
	if lcmcommon.Contains(builtinCephPools, pool.Name) {
		cephpool.Spec.Name = pool.Name
	}
	if pool.PreserveOnDelete {
		cephpool.Annotations = map[string]string{poolPreserveOnDeleteAnnotation: "true"}
	}
	return &cephpool
}

func (c *cephDeploymentConfig) ensurePools() (bool, error) {
	c.log.Info().Msg("ensure ceph block pools")
	// List CephBlockPools
	pools, err := c.api.Rookclientset.CephV1().CephBlockPools(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to get list pools")
	}

	presentPools := map[string]*cephv1.CephBlockPool{}
	for _, pool := range pools.Items {
		if !lcmcommon.Contains(builtinCephPools, pool.Spec.Name) {
			presentPools[pool.Name] = pool.DeepCopy()
		}
	}
	errMsg := make([]error, 0)
	poolsChanged := false
	for _, cephDplPool := range c.cdConfig.cephDpl.Spec.Pools {
		newPool := generatePool(cephDplPool, c.lcmConfig.RookNamespace)
		if presentPool, ok := presentPools[newPool.Name]; ok {
			if presentPool.Status == nil || !isTypeReadyToUpdate(presentPool.Status.Phase) {
				err := fmt.Sprintf("found not ready CephBlockPool %s/%s, waiting for readiness", c.lcmConfig.RookNamespace, presentPool.Name)
				if presentPool.Status != nil {
					err = fmt.Sprintf("%s (current phase is %v)", err, presentPool.Status.Phase)
				}
				c.log.Error().Msg(err)
				errMsg = append(errMsg, errors.New(err))
			} else {
				if !reflect.DeepEqual(newPool.Spec, presentPool.Spec) {
					lcmcommon.ShowObjectDiff(*c.log, presentPool.Spec, newPool.Spec)
					presentPool.Spec = newPool.Spec
					if err := c.processBlockPools(objectUpdate, presentPool); err != nil {
						errMsg = append(errMsg, err)
					}
					poolsChanged = true
				}
			}
			delete(presentPools, newPool.Name)
		} else {
			if err := c.processBlockPools(objectCreate, newPool); err != nil {
				errMsg = append(errMsg, err)
			}
			poolsChanged = true
		}
	}

	for _, pool := range presentPools {
		if pool.Annotations != nil && pool.Annotations[poolPreserveOnDeleteAnnotation] == "true" {
			c.log.Info().Msgf("CephBlockPool %s/%s contains annotation '%s', skip deletion", pool.Namespace, pool.Name, poolPreserveOnDeleteAnnotation)
			continue
		}
		if err := c.processBlockPools(objectDelete, pool); err != nil {
			errMsg = append(errMsg, err)
		}
		poolsChanged = true
	}

	if len(errMsg) == 1 {
		return false, errors.Wrap(errMsg[0], "failed to ensure CephBlockPools")
	} else if len(errMsg) > 1 {
		return false, errors.New("failed to ensure CephBlockPools, multiple errors during pools ensure")
	}
	return poolsChanged, nil
}

func (c *cephDeploymentConfig) deletePools() (bool, error) {
	poolList, err := c.api.Rookclientset.CephV1().CephBlockPools(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list ceph block pools")
	}
	if len(poolList.Items) == 0 {
		return true, nil
	}
	errMsgs := 0
	for _, pool := range poolList.Items {
		// skip builtin pool remove, because it will lead to cephobjectstore cleanup fail
		// .rgw.root will be cleaned during cephobjectstore cleanup
		if pool.Spec.Name == ".rgw.root" {
			c.log.Warn().Msgf("skipping pool '%s/%s' remove, should be removed during CephObjectStore cleanup", pool.Namespace, pool.Name)
			continue
		}
		if err := c.processBlockPools(objectDelete, &pool); err != nil {
			errMsgs++
		}
	}
	if errMsgs > 0 {
		return false, errors.New("some ceph block pools failed to delete")
	}
	return false, nil
}

func (c *cephDeploymentConfig) processBlockPools(process objectProcess, pool *cephv1.CephBlockPool) error {
	var err error
	switch process {
	case objectCreate:
		c.log.Info().Msgf("creating CephBlockPool %s/%s", pool.Namespace, pool.Name)
		_, err = c.api.Rookclientset.CephV1().CephBlockPools(pool.Namespace).Create(c.context, pool, metav1.CreateOptions{})
	case objectUpdate:
		c.log.Info().Msgf("updating CephBlockPool %s/%s", pool.Namespace, pool.Name)
		_, err = c.api.Rookclientset.CephV1().CephBlockPools(pool.Namespace).Update(c.context, pool, metav1.UpdateOptions{})
	case objectDelete:
		c.log.Info().Msgf("removing CephBlockPool %s/%s", pool.Namespace, pool.Name)
		err = c.api.Rookclientset.CephV1().CephBlockPools(pool.Namespace).Delete(c.context, pool.Name, metav1.DeleteOptions{})
	}
	if err != nil {
		if process == objectDelete && apierrors.IsNotFound(err) {
			return nil
		}
		err = errors.Wrapf(err, "failed to %v CephBlockPool %s/%s", process, pool.Namespace, pool.Name)
		c.log.Error().Msg(err.Error())
		return err
	}
	return nil
}
