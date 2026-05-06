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
	v1core "k8s.io/api/core/v1"
	v1storage "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func getStorageClassNameCephFS(cephFsName string, poolName string) string {
	if poolName == "data0" {
		return cephFsName + "-cephfs"
	}
	return fmt.Sprintf("%s-%s", cephFsName, poolName)
}

func generateStorageClassPoolBased(clusterid, poolName string, storageOpts cephlcmv1alpha1.CephStorageClassSpec, namespace string, isExternal bool) *v1storage.StorageClass {
	storageclass := v1storage.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:        poolName,
			Labels:      lcmcommon.ExtendLabels(map[string]string{rookStorageClassLabelKey: "true"}, baseResourceLabels),
			Annotations: map[string]string{rookDefaultSCAnnotationKey: fmt.Sprintf("%v", storageOpts.Default)},
		},
		Provisioner: rookRBDProvisionerName,
		Parameters: map[string]string{
			"clusterID":   clusterid,
			"pool":        poolName,
			"imageFormat": "2",
			"csi.storage.k8s.io/provisioner-secret-name":      "rook-csi-rbd-provisioner",
			"csi.storage.k8s.io/provisioner-secret-namespace": namespace,
			"csi.storage.k8s.io/node-stage-secret-name":       "rook-csi-rbd-node",
			"csi.storage.k8s.io/node-stage-secret-namespace":  namespace,
		},
	}

	volumeExpansion := storageOpts.AllowVolumeExpansion || isExternal
	if volumeExpansion {
		storageclass.AllowVolumeExpansion = &volumeExpansion
		storageclass.Parameters["csi.storage.k8s.io/controller-expand-secret-name"] = "rook-csi-rbd-provisioner"
		storageclass.Parameters["csi.storage.k8s.io/controller-expand-secret-namespace"] = namespace
		storageclass.Parameters["csi.storage.k8s.io/fstype"] = "ext4"
	}

	if storageOpts.MapOptions != "" {
		storageclass.Parameters["mapOptions"] = storageOpts.MapOptions
	}
	if storageOpts.UnmapOptions != "" {
		storageclass.Parameters["unmapOptions"] = storageOpts.UnmapOptions
	}
	if storageOpts.ImageFeatures != "" {
		storageclass.Parameters["imageFeatures"] = storageOpts.ImageFeatures
	} else {
		storageclass.Parameters["imageFeatures"] = "layering"
	}

	if storageOpts.ReclaimPolicy != "" {
		reclaimPolicy := v1core.PersistentVolumeReclaimPolicy(storageOpts.ReclaimPolicy)
		storageclass.ReclaimPolicy = &reclaimPolicy
	}

	return &storageclass
}

func generateStorageClassCephFSBased(clusterID, cephFsName, dataPool, namespace string, keepAfterRemove bool) *v1storage.StorageClass {
	// let set some parameters static
	volumeExpansion := true
	reclaimPolicy := v1core.PersistentVolumeReclaimDelete
	storageclass := v1storage.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: getStorageClassNameCephFS(cephFsName, dataPool),
			Labels: lcmcommon.ExtendLabels(map[string]string{
				rookStorageClassLabelKey:         "true",
				rookStorageClassKeepOnSpecRemove: fmt.Sprintf("%v", keepAfterRemove),
			}, baseResourceLabels),
		},
		Provisioner: rookCephFSProvisionerName,
		Parameters: map[string]string{
			"clusterID": clusterID,
			"pool":      fmt.Sprintf("%s-%s", cephFsName, dataPool),
			"fsName":    cephFsName,
			"csi.storage.k8s.io/provisioner-secret-name":            "rook-csi-cephfs-provisioner",
			"csi.storage.k8s.io/provisioner-secret-namespace":       namespace,
			"csi.storage.k8s.io/node-stage-secret-name":             "rook-csi-cephfs-node",
			"csi.storage.k8s.io/node-stage-secret-namespace":        namespace,
			"csi.storage.k8s.io/controller-expand-secret-name":      "rook-csi-cephfs-provisioner",
			"csi.storage.k8s.io/controller-expand-secret-namespace": namespace,
		},
		ReclaimPolicy:        &reclaimPolicy,
		AllowVolumeExpansion: &volumeExpansion,
	}

	return &storageclass
}

func (c *cephDeploymentConfig) ensureStorageClasses() (bool, error) {
	c.log.Debug().Msg("ensure storage classes")
	storageClassesList, err := c.api.Kubeclientset.StorageV1().StorageClasses().List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to get storage classes list")
	}

	storageClassesToCreate := make([]*v1storage.StorageClass, 0)
	storageClassesToUpdate := make([]*v1storage.StorageClass, 0)
	storageClassesToDelete := map[string]bool{}
	for _, storageClass := range storageClassesList.Items {
		if storageClass.Labels[rookStorageClassLabelKey] == "true" && storageClass.Labels[rookStorageClassKeepOnSpecRemove] != "true" {
			storageClassesToDelete[storageClass.Name] = true
		}
	}

	if c.cdConfig.cephDpl.Spec.BlockStorage != nil {
		for idx, cephDplPool := range c.cdConfig.cephDpl.Spec.BlockStorage.Pools {
			poolName := c.cdConfig.pools[idx]
			storageResource := generateStorageClassPoolBased(c.lcmConfig.RookNamespace, poolName, cephDplPool.StorageClassOpts, c.lcmConfig.RookNamespace, c.cdConfig.clusterSpec.External.Enable)
			found := false
			for _, storageClass := range storageClassesList.Items {
				if poolName == storageClass.Name {
					found = true
					delete(storageClassesToDelete, storageClass.Name)
					updated := lcmcommon.AlignBaseLabels(*c.log, "StorageClass", &storageClass.ObjectMeta, storageResource.Labels)
					if storageClass.Annotations == nil {
						storageClass.SetAnnotations(map[string]string{})
					}
					if storageClass.Annotations[rookDefaultSCAnnotationKey] != fmt.Sprintf("%v", cephDplPool.StorageClassOpts.Default) {
						c.log.Info().Msgf("setting annotation '%s=%s' for storage class %s", rookDefaultSCAnnotationKey, fmt.Sprintf("%v", cephDplPool.StorageClassOpts.Default), storageClass.Name)
						storageClass.Annotations[rookDefaultSCAnnotationKey] = fmt.Sprintf("%v", cephDplPool.StorageClassOpts.Default)
						updated = true
					}
					if !reflect.DeepEqual(storageClass.Parameters, storageResource.Parameters) {
						lcmcommon.ShowObjectDiff(*c.log, storageClass.Parameters, storageResource.Parameters)
						c.log.Warn().Msgf("storageclass parameters update won't be applied for storage class '%[1]s', since parameters section is immutable,"+
							" need to recreate storage class '%[1]s' to apply new parameters", storageClass.Name)
					}
					if updated {
						storageClassesToUpdate = append(storageClassesToUpdate, &storageClass)
					}
				}
			}
			if !found {
				storageClassesToCreate = append(storageClassesToCreate, storageResource)
			}
		}
	}

	if c.cdConfig.cephDpl.Spec.SharedFilesystem != nil {
		for _, cephFS := range c.cdConfig.cephDpl.Spec.SharedFilesystem.Filesystems {
			castedSpec, _ := cephFS.GetSpec()
			cephFsDataPoolNames := make([]string, len(castedSpec.DataPools))
			for idx, dataPool := range castedSpec.DataPools {
				cephFsDataPoolNames[idx] = dataPool.Name
			}
			for _, dataPoolName := range cephFsDataPoolNames {
				newStorageClass := true
				storageClassName := getStorageClassNameCephFS(cephFS.Name, dataPoolName)
				storageResource := generateStorageClassCephFSBased(c.lcmConfig.RookNamespace, cephFS.Name, dataPoolName, c.lcmConfig.RookNamespace, castedSpec.PreserveFilesystemOnDelete)
				delete(storageClassesToDelete, storageClassName)
				for _, storageClass := range storageClassesList.Items {
					if storageClass.Name == storageClassName {
						newStorageClass = false
						changedBaseLabels := lcmcommon.AlignBaseLabels(*c.log, "StorageClass", &storageClass.ObjectMeta, storageResource.Labels)
						if changedBaseLabels {
							storageClassesToUpdate = append(storageClassesToUpdate, &storageClass)
						}
						break
					}
				}
				if newStorageClass {
					storageClassesToCreate = append(storageClassesToCreate, storageResource)
				}
			}
		}
	}

	errMsg := make([]error, 0)
	updated := len(storageClassesToCreate) > 0 || len(storageClassesToUpdate) > 0 || len(storageClassesToDelete) > 0

	err = c.createStorageClasses(storageClassesToCreate, c.cdConfig.clusterSpec.External.Enable)
	if err != nil {
		c.log.Error().Err(err).Msg("failed to create storageclasses")
		errMsg = append(errMsg, errors.Wrap(err, "failed to create storageclasses"))
	}
	err = c.updateStorageClasses(storageClassesToUpdate)
	if err != nil {
		c.log.Error().Err(err).Msg("failed to update storageclasses")
		errMsg = append(errMsg, errors.Wrap(err, "failed to update storageclasses"))
	}
	err = c.removeStorageClasses(storageClassesToDelete)
	if err != nil {
		c.log.Error().Err(err).Msg("failed to delete storageclasses")
		errMsg = append(errMsg, errors.Wrap(err, "failed to delete storageclasses"))
	}
	if len(errMsg) == 1 {
		return false, errMsg[0]
	} else if len(errMsg) > 1 {
		return false, errors.New("multiple errors during storageclasses ensure")
	}
	return updated, nil
}

func (c *cephDeploymentConfig) deleteStorageClasses() (bool, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=true", rookStorageClassLabelKey),
	}
	storageClassesList, err := c.api.Kubeclientset.StorageV1().StorageClasses().List(c.context, listOpts)
	if err != nil {
		return false, errors.Wrap(err, "failed to get storage classes list")
	}
	storageclassesToRemove := map[string]bool{}
	for _, storageClass := range storageClassesList.Items {
		storageclassesToRemove[storageClass.Name] = true
	}
	if len(storageclassesToRemove) == 0 {
		return true, nil
	}
	err = c.removeStorageClasses(storageclassesToRemove)
	if err != nil {
		return false, err
	}
	return false, nil
}

func (c *cephDeploymentConfig) getStorageClassesUsage(storageclasses map[string]bool) (map[string]bool, error) {
	pvcList, err := c.api.Kubeclientset.CoreV1().PersistentVolumeClaims(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return storageclasses, errors.Wrap(err, "failed to get list of pvc")
	}
	for _, pvc := range pvcList.Items {
		if pvc.Status.Phase != "Bound" {
			continue
		}
		if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName == "" {
			c.log.Warn().Msgf("PVC %s/%s has no specified storageclass, verify manually and remove if needed", c.lcmConfig.RookNamespace, pvc.Name)
			continue
		}
		if _, ok := storageclasses[*pvc.Spec.StorageClassName]; ok {
			c.log.Warn().Msgf("storage class %s is used for bounded PVC %s/%s", *pvc.Spec.StorageClassName, c.lcmConfig.RookNamespace, pvc.Name)
			storageclasses[*pvc.Spec.StorageClassName] = false
		}
	}
	pvList, err := c.api.Kubeclientset.CoreV1().PersistentVolumes().List(c.context, metav1.ListOptions{})
	if err != nil {
		return storageclasses, errors.Wrap(err, "failed to get list of pv")
	}
	for _, pv := range pvList.Items {
		if pv.Status.Phase != "Bound" {
			continue
		}
		if _, ok := storageclasses[pv.Spec.StorageClassName]; ok {
			c.log.Warn().Msgf("storage class %s is used for bounded PV %s", pv.Spec.StorageClassName, pv.Name)
			storageclasses[pv.Spec.StorageClassName] = false
		}
	}
	return storageclasses, nil
}

func (c *cephDeploymentConfig) createStorageClasses(storageClasses []*v1storage.StorageClass, isExternal bool) error {
	errMsg := make([]error, 0)
	for _, storageclass := range storageClasses {
		if !isExternal {
			if storageclass.Provisioner == rookRBDProvisionerName {
				if !isCephPoolReady(c.context, *c.log, c.api.Rookclientset, c.lcmConfig.RookNamespace, storageclass.Parameters["pool"]) {
					msg := fmt.Sprintf("failed to create StorageClass %s since corresponding %s pool is not ready yet", storageclass.Name, storageclass.Parameters["pool"])
					c.log.Error().Msg(msg)
					errMsg = append(errMsg, errors.New(msg))
					continue
				}
			} else if storageclass.Provisioner == rookCephFSProvisionerName {
				if !isCephFsReady(c.context, *c.log, c.api.Rookclientset, c.lcmConfig.RookNamespace, storageclass.Parameters["fsName"]) {
					msg := fmt.Sprintf("failed to create StorageClass %s since corresponding %s CephFs is not ready yet", storageclass.Name, storageclass.Parameters["fsName"])
					c.log.Error().Msg(msg)
					errMsg = append(errMsg, errors.New(msg))
					continue
				}
			} else {
				msg := fmt.Sprintf("failed to create StorageClass %s, unknown provisioner '%s' name", storageclass.Name, storageclass.Provisioner)
				c.log.Error().Msg(msg)
				errMsg = append(errMsg, errors.New(msg))
				continue
			}
		}
		c.log.Info().Msgf("creating storageclass %s", storageclass.Name)
		_, err := c.api.Kubeclientset.StorageV1().StorageClasses().Create(c.context, storageclass, metav1.CreateOptions{})
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to create StorageClass %q", storageclass.Name)
			errMsg = append(errMsg, errors.Wrapf(err, "failed to create StorageClass %q", storageclass.Name))
		}
	}
	if len(errMsg) == 1 {
		return errMsg[0]
	} else if len(errMsg) > 1 {
		return errors.New("multiple errors during storageclasses create")
	}
	return nil
}

func (c *cephDeploymentConfig) updateStorageClasses(storageClasses []*v1storage.StorageClass) error {
	errMsg := make([]error, 0)
	for _, storageclass := range storageClasses {
		c.log.Info().Msgf("updating storageclass %q", storageclass.Name)
		_, err := c.api.Kubeclientset.StorageV1().StorageClasses().Update(c.context, storageclass, metav1.UpdateOptions{})
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to update storageclass %q", storageclass.Name)
			errMsg = append(errMsg, errors.Wrapf(err, "failed to update StorageClass %q", storageclass.Name))
		}
	}
	if len(errMsg) == 1 {
		return errMsg[0]
	} else if len(errMsg) > 1 {
		return errors.New("multiple errors during storageclasses update")
	}
	return nil
}

func (c *cephDeploymentConfig) removeStorageClasses(storageClasses map[string]bool) error {
	if len(storageClasses) > 0 {
		storageClasses, err := c.getStorageClassesUsage(storageClasses)
		if err != nil {
			return errors.Wrap(err, "failed to check storage classes usage")
		}
		errMsg := 0
		for storageclassName, canRemove := range storageClasses {
			if !canRemove {
				c.log.Error().Msgf("can't delete used storageclass %s", storageclassName)
				errMsg++
				continue
			}
			c.log.Info().Msgf("removing storageclass %q", storageclassName)
			err = c.api.Kubeclientset.StorageV1().StorageClasses().Delete(c.context, storageclassName, metav1.DeleteOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				c.log.Error().Err(err).Msgf("failed to delete storageclass %s", storageclassName)
				errMsg++
			}
		}
		if errMsg > 0 {
			return errors.New("delete storageclass(es) failed")
		}
	}
	return nil
}
