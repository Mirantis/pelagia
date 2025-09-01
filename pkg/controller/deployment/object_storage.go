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

	"github.com/pkg/errors"
)

func (c *cephDeploymentConfig) ensureObjectStorage() (bool, error) {
	errCollector := make([]error, 0)
	// Delete all object storage stuff if there is no objectstore section
	if c.cdConfig.cephDpl.Spec.ObjectStorage == nil {
		c.log.Info().Msg("no objectStorage section, skip rgw/multisite ensure and cleanup all object storage stuff if present")
		removed, err := c.deleteObjectStorage()
		if err != nil {
			c.log.Error().Err(err).Msg("error deleting object storage object")
			return false, err
		}
		return !removed, nil
	}
	c.log.Info().Msg("ensure object storage")
	objectStorageChanged := false
	if !c.cdConfig.cephDpl.Spec.External {
		if c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite == nil {
			c.log.Info().Msg("no object storage multisite section, skip multisite ensure and cleanup all multisite stuff")
			removed, err := c.deleteMultiSite()
			if err != nil {
				msg := fmt.Sprintf("failed to cleanup object storage multisite: %v", err)
				c.log.Error().Err(err).Msg(msg)
				errCollector = append(errCollector, errors.New(msg))
			}
			objectStorageChanged = !removed
		} else {
			// Ensure ceph rgw multisite processing
			changed, err := c.ensureRgwMultiSite()
			if err != nil {
				c.log.Error().Err(err).Msg("failed to ensure object storage multisite")
				errCollector = append(errCollector, errors.Wrap(err, "failed to ensure ceph object storage multisite"))
			}
			objectStorageChanged = changed
		}
	}

	// Ensure ceph rgw processing
	changed, err := c.ensureRgw()
	if err != nil {
		c.log.Error().Err(err).Msg("failed to ensure ceph rgw")
		errCollector = append(errCollector, errors.Wrap(err, "failed to ensure ceph rgw"))
	}
	objectStorageChanged = objectStorageChanged || changed

	// Return error if exists
	if len(errCollector) == 1 {
		return false, errCollector[0]
	} else if len(errCollector) > 1 {
		return false, errors.New("multiple errors during object storage ensure")
	}
	return objectStorageChanged, nil
}

func (c *cephDeploymentConfig) deleteObjectStorage() (bool, error) {
	errorsNumber := 0
	rgwRemoved, err := c.deleteRgw("", false)
	if err != nil {
		c.log.Error().Err(err).Msg("error deleting rgw")
		errorsNumber++
	}
	if rgwRemoved {
		certsRemoved, err := c.deleteRgwInternalSslCert()
		if err != nil {
			c.log.Error().Err(err).Msg("error deleting rgw ssl cert")
			errorsNumber++
		}
		rgwRemoved = certsRemoved
		if !c.cdConfig.cephDpl.Spec.External {
			multisiteRemoved, err := c.deleteMultiSite()
			if err != nil {
				c.log.Error().Err(err).Msg("error deleting rgw multisite")
				errorsNumber++
			}
			rgwRemoved = rgwRemoved && multisiteRemoved
			// remove built-in pool only after multisite cleanup if present
			if multisiteRemoved {
				builtInPoolRemoved, err := c.deleteRgwBuiltInPool()
				if err != nil {
					c.log.Error().Err(err).Msg("error deleting builtin .rgw.root pool")
					errorsNumber++
				}
				rgwRemoved = rgwRemoved && builtInPoolRemoved
			}
		} else {
			keysRemoved, err := c.deleteRgwAdminOpsSecret()
			if err != nil {
				c.log.Error().Err(err).Msg("error deleting rgw admin ops secret")
				errorsNumber++
			}
			rgwRemoved = rgwRemoved && keysRemoved
		}
	}
	if errorsNumber > 0 {
		return false, errors.New("failed to cleanup object storage")
	}
	return rgwRemoved, nil
}
