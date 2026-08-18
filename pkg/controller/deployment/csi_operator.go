/*
Copyright 2026 Mirantis IT.

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

	csiopapi "github.com/ceph/ceph-csi-operator/api/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1storage "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/v3/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/v3/pkg/common"
)

func (c *cephDeploymentConfig) ensureCsiResources() (bool, error) {
	if !c.lcmConfig.DeployParams.CSIParams.Manage {
		c.log.Warn().Msg("ensure cephcsi resources is disabled for Pelagia")
		return false, nil
	}
	updatedOpConfig, err := c.ensureCsiOperatorConfig()
	if err != nil {
		return false, err
	}
	updatedDrivers, err := c.ensureCsiDrivers()
	if err != nil {
		return false, err
	}
	return updatedOpConfig || updatedDrivers, nil
}

func (c *cephDeploymentConfig) ensureCsiOperatorConfig() (bool, error) {
	c.log.Debug().Msg("ensure cephcsi operatorconfig")
	opConfig := csiopapi.OperatorConfig{}
	opConfig.Name = cephCsiOperatorConfigName
	opConfig.Namespace = c.lcmConfig.RookNamespace
	present := true
	err := c.api.ClientNoCache.Get(c.context, crclient.ObjectKeyFromObject(&opConfig), &opConfig)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			c.log.Error().Err(err).Msg("")
			return false, errors.Wrapf(err, "failed to verify cephcsi OperatorConfig '%s/%s'", opConfig.Namespace, opConfig.Name)
		}
		present = false
	} else if c.lcmConfig.DeployParams.CSIParams.KeepExisting {
		// skip default opconfig object, if nothing provided in spec and keep exist set
		if !resourceCreatedByPelagia(opConfig.Labels) && (c.cdConfig.cephDpl.Spec.CSIResources == nil || c.cdConfig.cephDpl.Spec.CSIResources.OperatorConfig == nil) {
			c.log.Warn().Msgf("found cephcsi OperatorConfig '%s/%s', created not by Pelagia, 'DEPLOYMENT_CSI_KEEP_EXISTING_ON_UPGRADE' is set and no override provided in spec, skipping",
				opConfig.Namespace, opConfig.Name)
			return false, nil
		}
	}
	newOpConfig := csiopapi.OperatorConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cephCsiOperatorConfigName,
			Namespace: c.lcmConfig.RookNamespace,
			Labels:    baseResourceLabels,
		},
		Spec: csiopapi.OperatorConfigSpec{
			DriverSpecDefaults: c.generateDefaultDriverSpec(),
		},
	}
	if c.cdConfig.cephDpl.Spec.CSIResources != nil && c.cdConfig.cephDpl.Spec.CSIResources.OperatorConfig != nil {
		opConfigFromSpec, _ := c.cdConfig.cephDpl.Spec.CSIResources.OperatorConfig.GetSpec()
		if c.cdConfig.cephDpl.Spec.CSIResources.OperatorConfig.FullOverride {
			newOpConfig.Spec = opConfigFromSpec
			// always set cluster name and image set explicitly
			if newOpConfig.Spec.DriverSpecDefaults.ClusterName == nil {
				newOpConfig.Spec.DriverSpecDefaults.ClusterName = &c.cdConfig.cephDpl.Name
			}
		} else {
			if c.lcmConfig.DeployParams.CSIParams.KeepExisting && present {
				mergeDriverSpecs(opConfigFromSpec.DriverSpecDefaults, opConfig.Spec.DriverSpecDefaults)
			} else {
				mergeDriverSpecs(opConfigFromSpec.DriverSpecDefaults, newOpConfig.Spec.DriverSpecDefaults)
			}
			newOpConfig.Spec = opConfigFromSpec
		}
	}
	if present {
		labelsUpdated := lcmcommon.AlignBaseLabels(*c.log, "OperatorConfig", &opConfig.ObjectMeta, newOpConfig.Labels)
		specUpdated := !reflect.DeepEqual(newOpConfig.Spec, opConfig.Spec)
		if specUpdated {
			lcmcommon.ShowObjectDiff(*c.log, opConfig.Spec, newOpConfig.Spec)
			opConfig.Spec = newOpConfig.Spec
		}
		if specUpdated || labelsUpdated {
			c.log.Info().Msgf("updating cephcsi operator config '%s/%s'", opConfig.Namespace, opConfig.Name)
			err = c.api.ClientNoCache.Update(c.context, &opConfig)
			if err != nil {
				c.log.Error().Err(err).Msg("")
				return false, errors.Wrapf(err, "failed to update cephcsi OperatorConfig '%s/%s'", opConfig.Namespace, opConfig.Name)
			}
		}
		return specUpdated || labelsUpdated, nil
	}
	c.log.Info().Msgf("creating cephcsi operator config '%s/%s'", newOpConfig.Namespace, newOpConfig.Name)
	err = c.api.ClientNoCache.Create(c.context, &newOpConfig)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return false, errors.Wrapf(err, "failed to create cephcsi OperatorConfig '%s/%s'", newOpConfig.Namespace, newOpConfig.Name)
	}
	return true, nil
}

func (c *cephDeploymentConfig) generateDefaultDriverSpec() *csiopapi.DriverSpec {
	return &csiopapi.DriverSpec{
		ImageSet:         &corev1.LocalObjectReference{Name: cephCsiOperatorImageConfigMapName},
		ClusterName:      &c.cdConfig.cephDpl.Name,
		GenerateOMapInfo: lcmcommon.PtrTo(false),
		FsGroupPolicy:    v1storage.FileFSGroupPolicy,

		NodePlugin: &csiopapi.NodePluginSpec{
			PodCommonSpec: csiopapi.PodCommonSpec{
				PrioritylClassName: lcmcommon.PtrTo(""),
				Affinity: &corev1.Affinity{
					NodeAffinity: c.lcmConfig.DeployParams.CSIParams.NodePluginNodeAffinity,
				},
				Tolerations: c.lcmConfig.DeployParams.CSIParams.NodePluginToleration,
			},
			Resources:              csiopapi.NodePluginResourcesSpec{},
			KubeletDirPath:         c.lcmConfig.DeployParams.CSIParams.KubeletPath,
			EnableSeLinuxHostMount: lcmcommon.PtrTo(true),
		},
		ControllerPlugin: &csiopapi.ControllerPluginSpec{
			PodCommonSpec: csiopapi.PodCommonSpec{
				PrioritylClassName: lcmcommon.PtrTo(""),
				Affinity: &corev1.Affinity{
					NodeAffinity: c.lcmConfig.DeployParams.CSIParams.ControllerPluginNodeAffinity,
				},
				Tolerations: c.lcmConfig.DeployParams.CSIParams.ControllerPluginToleration,
			},

			Replicas:  lcmcommon.PtrTo(int32(2)),
			Resources: csiopapi.ControllerPluginResourcesSpec{},
		},
		DeployCsiAddons:  &c.lcmConfig.DeployParams.CSIParams.DeployCSIAddons,
		CephFsClientType: csiopapi.KernelCephFsClient,
	}
}

func (c *cephDeploymentConfig) ensureCsiDrivers() (bool, error) {
	c.log.Debug().Msg("ensure cephcsi drivers")
	driversToHave := map[cephlcmv1alpha1.CSIDriverType]int{}
	driversNamesToHave := map[string]bool{}
	// add default drivers for default creation
	if c.lcmConfig.DeployParams.CSIParams.CreateDefaultRBDDriver {
		driversToHave[cephlcmv1alpha1.RBDCSIDriver] = -1
	}
	if c.lcmConfig.DeployParams.CSIParams.CreateDefaultCephFSDriver {
		driversToHave[cephlcmv1alpha1.CephFSCSIDriver] = -1
	}
	if c.lcmConfig.DeployParams.CSIParams.CreateDefaultNFSDriver {
		driversToHave[cephlcmv1alpha1.NFSCSIDriver] = -1
	}
	if c.cdConfig.cephDpl.Spec.CSIResources != nil {
		for idx, driver := range c.cdConfig.cephDpl.Spec.CSIResources.Drivers {
			driversToHave[driver.Type] = idx
		}
	}

	errs := 0
	updated := false
	for driverType, driverIdx := range driversToHave {
		cephDriver := csiopapi.Driver{}
		cephDriver.Name = fmt.Sprintf(cephCsiDriverNameTemplate, c.lcmConfig.RookNamespace, driverType)
		driversNamesToHave[cephDriver.Name] = true
		cephDriver.Namespace = c.lcmConfig.RookNamespace
		present := true
		err := c.api.ClientNoCache.Get(c.context, crclient.ObjectKeyFromObject(&cephDriver), &cephDriver)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				errs++
				c.log.Error().Err(err).Msgf("failed to verify cephcsi Driver '%s/%s'", cephDriver.Namespace, cephDriver.Name)
				continue
			}
			present = false
		} else if c.lcmConfig.DeployParams.CSIParams.KeepExisting {
			// if no spec override - skip
			if driversToHave[driverType] == -1 && !resourceCreatedByPelagia(cephDriver.Labels) {
				c.log.Warn().Msgf("found cephcsi Driver '%s/%s' created not by Pelagia, 'DEPLOYMENT_CSI_KEEP_EXISTING_ON_UPGRADE' is set and no override provided in spec, skipping",
					cephDriver.Namespace, cephDriver.Name)
				continue
			}
		}
		var driverSpec csiopapi.DriverSpec
		if driverIdx >= 0 {
			driverSpec, _ = c.cdConfig.cephDpl.Spec.CSIResources.Drivers[driverIdx].GetSpec()
		} else {
			driverSpec = csiopapi.DriverSpec{}
		}
		newDriver := csiopapi.Driver{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cephDriver.Name,
				Namespace: c.lcmConfig.RookNamespace,
				Labels:    baseResourceLabels,
			},
			Spec: driverSpec,
		}
		if driverType == cephlcmv1alpha1.CephFSCSIDriver {
			u := &unstructured.Unstructured{}
			u.SetName("volumegroupsnapshotclasses.groupsnapshot.storage.k8s.io")
			u.SetAPIVersion("apiextensions.k8s.io/v1")
			u.SetKind("CustomResourceDefinition")
			err := c.api.ClientNoCache.Get(c.context, crclient.ObjectKeyFromObject(u), u)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					errs++
					c.log.Error().Err(err).Msg("failed to check CRD")
				}
			} else {
				newDriver.Spec.SnapshotPolicy = csiopapi.VolumeGroupSnapshotPolicy
			}
		}
		if present {
			labelsUpdated := lcmcommon.AlignBaseLabels(*c.log, "Driver", &cephDriver.ObjectMeta, newDriver.Labels)
			// if we want just extend present driver - and do not full override
			if driverIdx >= 0 && !c.cdConfig.cephDpl.Spec.CSIResources.Drivers[driverIdx].FullOverride {
				mergeDriverSpecs(&newDriver.Spec, &cephDriver.Spec)
			}
			specUpdated := !reflect.DeepEqual(newDriver.Spec, cephDriver.Spec)
			if specUpdated {
				lcmcommon.ShowObjectDiff(*c.log, cephDriver.Spec, newDriver.Spec)
				cephDriver.Spec = newDriver.Spec
			}
			updated = specUpdated || labelsUpdated
			if updated {
				c.log.Info().Msgf("updating cephcsi driver '%s/%s'", cephDriver.Namespace, cephDriver.Name)
				err = c.api.ClientNoCache.Update(c.context, &cephDriver)
				if err != nil {
					errs++
					c.log.Error().Msgf("failed to update cephcsi Driver '%s/%s'", cephDriver.Namespace, cephDriver.Name)
				}
			}
			continue
		}
		c.log.Info().Msgf("creating cephcsi driver '%s/%s'", newDriver.Namespace, newDriver.Name)
		err = c.api.ClientNoCache.Create(c.context, &newDriver)
		if err != nil {
			errs++
			c.log.Error().Msgf("failed to create cephcsi Driver '%s/%s'", newDriver.Namespace, newDriver.Name)
		} else {
			updated = true
		}
	}

	csiDrivers := &csiopapi.DriverList{}
	err := c.api.ClientNoCache.List(c.context, csiDrivers, &crclient.ListOptions{Namespace: c.lcmConfig.RookNamespace})
	if err != nil {
		errs++
		c.log.Error().Err(err).Msgf("failed to check cephcsi Drivers to remove in '%s' namespace", c.lcmConfig.RookNamespace)
	} else {
		for _, driver := range csiDrivers.Items {
			if driversNamesToHave[driver.Name] {
				continue
			}
			if !resourceCreatedByPelagia(driver.Labels) && c.lcmConfig.DeployParams.CSIParams.KeepExisting {
				continue
			}
			c.log.Info().Msgf("removing cephcsi driver '%s/%s'", driver.Namespace, driver.Name)
			err = c.api.ClientNoCache.Delete(c.context, &driver)
			if err != nil {
				errs++
				c.log.Error().Msgf("failed to remove cephcsi Driver '%s/%s'", driver.Namespace, driver.Name)
			} else {
				updated = true
			}
		}
	}

	if errs > 0 {
		return false, errors.New("failed to ensure CephCSI Drivers")
	}
	return updated, nil
}

func (c *cephDeploymentConfig) deleteCsiOperatorResources() (bool, error) {
	errs := 0
	removed, err := c.deleteCsiClientProfile()
	if err != nil {
		c.log.Error().Err(err).Msg("failed to remove clientprofile object")
		errs++
	}
	if removed {
		cephConRemoved, err := c.deleteCsiCephConnection()
		if err != nil {
			c.log.Error().Err(err).Msg("failed to remove cephconnection object")
			errs++
		}
		driversRemoved, err := c.deleteCsiDrivers()
		if err != nil {
			c.log.Error().Err(err).Msg("failed to remove driver(s) object")
			errs++
		}
		opConfigRemoved, err := c.deleteCsiOperatorConfig()
		if err != nil {
			c.log.Error().Err(err).Msg("failed to remove operatorconfig object")
			errs++
		}
		removed = cephConRemoved && driversRemoved && opConfigRemoved
	}
	if errs > 0 {
		return false, errors.New("failed to cleanup CSI Operator resources")
	}
	return removed, nil
}

func (c *cephDeploymentConfig) deleteCsiClientProfile() (bool, error) {
	csiClientProfiles := &csiopapi.ClientProfileList{}
	err := c.api.ClientNoCache.List(c.context, csiClientProfiles, &crclient.ListOptions{Namespace: c.lcmConfig.RookNamespace})
	if err != nil {
		return false, errors.Wrapf(err, "failed to list csi ClientProfiles in '%s' namespace", c.lcmConfig.RookNamespace)
	}
	removed := true
	for _, csiClientProfile := range csiClientProfiles.Items {
		err := c.api.ClientNoCache.Delete(c.context, &csiClientProfile)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, errors.Wrapf(err, "failed to delete csi ClientProfile '%s/%s'", csiClientProfile.Namespace, csiClientProfile.Name)
		}
		c.log.Info().Msgf("removing csi ClientProfile '%s/%s'", csiClientProfile.Namespace, csiClientProfile.Name)
		removed = false
	}
	return removed, nil
}

func (c *cephDeploymentConfig) deleteCsiCephConnection() (bool, error) {
	csiCephConnections := &csiopapi.CephConnectionList{}
	err := c.api.ClientNoCache.List(c.context, csiCephConnections, &crclient.ListOptions{Namespace: c.lcmConfig.RookNamespace})
	if err != nil {
		return false, errors.Wrapf(err, "failed to list csi CephConnections in '%s' namespace", c.lcmConfig.RookNamespace)
	}
	removed := true
	for _, csiCephConnection := range csiCephConnections.Items {
		err := c.api.ClientNoCache.Delete(c.context, &csiCephConnection)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, errors.Wrapf(err, "failed to delete csi CephConnection '%s/%s'", csiCephConnection.Namespace, csiCephConnection.Name)
		}
		c.log.Info().Msgf("removing csi CephConnection '%s/%s'", csiCephConnection.Namespace, csiCephConnection.Name)
		removed = false
	}
	return removed, nil
}

func (c *cephDeploymentConfig) deleteCsiOperatorConfig() (bool, error) {
	csiOperatorConfigs := &csiopapi.OperatorConfigList{}
	err := c.api.ClientNoCache.List(c.context, csiOperatorConfigs, &crclient.ListOptions{Namespace: c.lcmConfig.RookNamespace})
	if err != nil {
		return false, errors.Wrapf(err, "failed to list csi OperatorConfigs in '%s' namespace", c.lcmConfig.RookNamespace)
	}
	removed := true
	for _, config := range csiOperatorConfigs.Items {
		if config.Spec.DriverSpecDefaults != nil && config.Spec.DriverSpecDefaults.ClusterName != nil {
			// remove all drivers in our namespace
			removed = false
			err := c.api.ClientNoCache.Delete(c.context, &config)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return true, nil
				}
				return false, errors.Wrapf(err, "failed to delete csi OperatorConfig '%s/%s'", config.Namespace, config.Name)
			}
			c.log.Info().Msgf("removing csi OperatorConfig '%s/%s'", config.Namespace, config.Name)
		}
	}
	return removed, nil
}

func (c *cephDeploymentConfig) deleteCsiDrivers() (bool, error) {
	csiDrivers := &csiopapi.DriverList{}
	err := c.api.ClientNoCache.List(c.context, csiDrivers, &crclient.ListOptions{Namespace: c.lcmConfig.RookNamespace})
	if err != nil {
		return false, errors.Wrapf(err, "failed to list csi Drivers in '%s' namespace", c.lcmConfig.RookNamespace)
	}
	removed := true
	for _, driver := range csiDrivers.Items {
		// remove all drivers in our namespace
		removed = false
		err := c.api.ClientNoCache.Delete(c.context, &driver)
		if err != nil {
			return false, errors.Wrapf(err, "failed to delete csi Driver '%s/%s'", driver.Namespace, driver.Name)
		}
		c.log.Info().Msgf("removing csi Driver '%s/%s'", driver.Namespace, driver.Name)
	}
	return removed, nil
}

// copy of ceph-csi-operator function from https://github.com/ceph/ceph-csi-operator/blob/v1.0.4/internal/controller/driver_controller.go#L1760
// function is not public, but we need to use merging, to have consistency with upstream
// except for ImageSet field - they missed in upstream
// mergeDriverSpecs will fill in any unset fields in dest with a copy of the same field in src
func mergeDriverSpecs(dest, src *csiopapi.DriverSpec) {
	// Create a copy of the src, making sure that any value copied into dest is a not shared
	// with the original src
	src = src.DeepCopy()

	if dest.Log == nil {
		dest.Log = src.Log
	}
	if dest.ImageSet == nil {
		dest.ImageSet = src.ImageSet
	}
	if dest.ClusterName == nil {
		dest.ClusterName = src.ClusterName
	}
	if dest.GRpcTimeout == 0 {
		dest.GRpcTimeout = src.GRpcTimeout
	}
	if dest.SnapshotPolicy == "" {
		dest.SnapshotPolicy = src.SnapshotPolicy
	}
	if dest.GenerateOMapInfo == nil {
		dest.GenerateOMapInfo = src.GenerateOMapInfo
	}
	if dest.EnableFencing == nil {
		dest.EnableFencing = src.EnableFencing
	}
	if dest.FsGroupPolicy == "" {
		dest.FsGroupPolicy = src.FsGroupPolicy
	}
	if dest.Encryption == nil {
		dest.Encryption = src.Encryption
	}
	if src.NodePlugin != nil {
		if dest.NodePlugin == nil {
			dest.NodePlugin = src.NodePlugin
		} else {
			dest, src := dest.NodePlugin, src.NodePlugin
			if dest.ServiceAccountName == nil {
				dest.ServiceAccountName = src.ServiceAccountName
			}
			if dest.PrioritylClassName == nil {
				dest.PrioritylClassName = src.PrioritylClassName
			}
			if dest.Labels == nil {
				dest.Labels = src.Labels
			}
			if dest.Annotations == nil {
				dest.Annotations = src.Annotations
			}
			if dest.Affinity == nil {
				dest.Affinity = src.Affinity
			}
			if dest.Tolerations == nil {
				dest.Tolerations = src.Tolerations
			}
			if dest.Volumes == nil {
				dest.Volumes = src.Volumes
			}
			if dest.ImagePullPolicy == "" {
				dest.ImagePullPolicy = src.ImagePullPolicy
			}
			if dest.UpdateStrategy == nil {
				dest.UpdateStrategy = src.UpdateStrategy
			}
			if dest.KubeletDirPath == "" {
				dest.KubeletDirPath = src.KubeletDirPath
			}
			if dest.EnableSeLinuxHostMount == nil {
				dest.EnableSeLinuxHostMount = src.EnableSeLinuxHostMount
			}
			if dest.Resources.Registrar == nil {
				dest.Resources.Registrar = src.Resources.Registrar
			}
			if dest.Resources.Liveness == nil {
				dest.Resources.Liveness = src.Resources.Liveness
			}
			if dest.Resources.Plugin == nil {
				dest.Resources.Plugin = src.Resources.Plugin
			}
			if dest.Resources.LogRotator == nil {
				dest.Resources.LogRotator = src.Resources.LogRotator
			}
			if dest.Resources.Addons == nil {
				dest.Resources.Addons = src.Resources.Addons
			}
			if dest.ContainerExtraArgs == nil {
				dest.ContainerExtraArgs = src.ContainerExtraArgs
			}
		}
	}
	if src.ControllerPlugin != nil {
		if dest.ControllerPlugin == nil {
			dest.ControllerPlugin = src.ControllerPlugin
		} else {
			dest, src := dest.ControllerPlugin, src.ControllerPlugin
			if dest.ServiceAccountName == nil {
				dest.ServiceAccountName = src.ServiceAccountName
			}
			if dest.PrioritylClassName == nil {
				dest.PrioritylClassName = src.PrioritylClassName
			}
			if dest.Labels == nil {
				dest.Labels = src.Labels
			}
			if dest.Annotations == nil {
				dest.Annotations = src.Annotations
			}
			if dest.Affinity == nil {
				dest.Affinity = src.Affinity
			}
			if dest.Tolerations == nil {
				dest.Tolerations = src.Tolerations
			}
			if dest.Volumes == nil {
				dest.Volumes = src.Volumes
			}
			if dest.ImagePullPolicy == "" {
				dest.ImagePullPolicy = src.ImagePullPolicy
			}
			if dest.Replicas == nil {
				dest.Replicas = src.Replicas
			}
			if dest.Privileged == nil {
				dest.Privileged = src.Privileged
			}
			if dest.Resources.Attacher == nil {
				dest.Resources.Attacher = src.Resources.Attacher
			}
			if dest.Resources.Snapshotter == nil {
				dest.Resources.Snapshotter = src.Resources.Snapshotter
			}
			if dest.Resources.Resizer == nil {
				dest.Resources.Resizer = src.Resources.Resizer
			}
			if dest.Resources.Provisioner == nil {
				dest.Resources.Provisioner = src.Resources.Provisioner
			}
			if dest.Resources.OMapGenerator == nil {
				dest.Resources.OMapGenerator = src.Resources.OMapGenerator
			}
			if dest.Resources.Liveness == nil {
				dest.Resources.Liveness = src.Resources.Liveness
			}
			if dest.Resources.Plugin == nil {
				dest.Resources.Plugin = src.Resources.Plugin
			}
			if dest.Resources.LogRotator == nil {
				dest.Resources.LogRotator = src.Resources.LogRotator
			}
			if dest.Resources.Addons == nil {
				dest.Resources.Addons = src.Resources.Addons
			}
			if dest.ContainerExtraArgs == nil {
				dest.ContainerExtraArgs = src.ContainerExtraArgs
			}
		}
	}
	if dest.AttachRequired == nil {
		dest.AttachRequired = src.AttachRequired
	}
	if dest.Liveness == nil {
		dest.Liveness = src.Liveness
	}
	if dest.LeaderElection == nil {
		dest.LeaderElection = src.LeaderElection
	}
	if dest.DeployCsiAddons == nil {
		dest.DeployCsiAddons = src.DeployCsiAddons
	}
	if dest.KernelMountOptions == nil {
		dest.KernelMountOptions = src.KernelMountOptions
	}
	if dest.FuseMountOptions == nil {
		dest.FuseMountOptions = src.FuseMountOptions
	}
	if dest.CephFsClientType == "" {
		dest.CephFsClientType = src.CephFsClientType
	}
}
