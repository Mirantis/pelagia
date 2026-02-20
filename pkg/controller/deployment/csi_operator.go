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
	csiopapi "github.com/ceph/ceph-csi-operator/api/v1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

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
			if *config.Spec.DriverSpecDefaults.ClusterName == c.cdConfig.cephDpl.Name {
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
		if driver.Spec.ClusterName != nil && *driver.Spec.ClusterName == c.cdConfig.cephDpl.Name {
			removed = false
			err := c.api.ClientNoCache.Delete(c.context, &driver)
			if err != nil {
				return false, errors.Wrapf(err, "failed to delete csi Driver '%s/%s'", driver.Namespace, driver.Name)
			}
			c.log.Info().Msgf("removing csi Driver '%s/%s'", driver.Namespace, driver.Name)
		}
	}
	return removed, nil
}
