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
	"strings"

	"github.com/pkg/errors"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

func (c *cephDeploymentConfig) ensureObjectStorage() (bool, error) {
	errCollector := make([]string, 0)
	// if no obj section specified, set it to empty to allow check no resources exist
	if c.cdConfig.cephDpl.Spec.ObjectStorage == nil {
		c.cdConfig.cephDpl.Spec.ObjectStorage = &cephlcmv1alpha1.CephObjectStorage{}
	}
	objectStorageChanged := false
	if !c.cdConfig.clusterSpec.External.Enable {
		// Ensure ceph rgw multisite processing
		changed, err := c.ensureRgwMultiSite()
		if err != nil {
			msg := "failed to ensure object storage multisite"
			c.log.Error().Err(err).Msg(msg)
			errCollector = append(errCollector, msg)
		}
		objectStorageChanged = changed
	}

	// Ensure ceph rgw processing
	changed, err := c.ensureRgw()
	if err != nil {
		msg := "failed to ensure ceph rgw"
		c.log.Error().Err(err).Msg(msg)
		errCollector = append(errCollector, msg)
	}
	objectStorageChanged = objectStorageChanged || changed

	// if we dont have obj storage - cleanup not needed resources
	// once no any pre-req exists
	if len(errCollector) == 0 && !objectStorageChanged {
		if c.cdConfig.clusterSpec.External.Enable {
			if len(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws) == 0 {
				keysRemoved, err := c.deleteRgwAdminOpsSecret()
				if err != nil {
					msg := "failed to delete external rgw admin ops secret"
					c.log.Error().Err(err).Msg(msg)
					errCollector = append(errCollector, msg)
				}
				objectStorageChanged = objectStorageChanged || !keysRemoved
			}
		} else {
			if len(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws) == 0 && len(c.cdConfig.cephDpl.Spec.ObjectStorage.Zones) == 0 {
				builtInPoolRemoved, err := c.deleteRgwBuiltInPool()
				if err != nil {
					msg := "failed to delete builtin .rgw.root pool"
					c.log.Error().Err(err).Msg(msg)
					errCollector = append(errCollector, msg)
				}
				objectStorageChanged = objectStorageChanged || !builtInPoolRemoved
			}
		}
	}

	// Return error if exists
	if len(errCollector) > 0 {
		return false, errors.Errorf("error(s) during object storage ensure: %s", strings.Join(errCollector, ", "))
	}
	return objectStorageChanged, nil
}

func (c *cephDeploymentConfig) deleteObjectStorage() (bool, error) {
	errorsNumber := 0
	rgwRemoved, err := c.deleteRgw("")
	if err != nil {
		c.log.Error().Err(err).Msg("error deleting rgw")
		errorsNumber++
	}
	if rgwRemoved {
		certsRemoved, err := c.deleteSelfSignedCerts(nil)
		if err != nil {
			c.log.Error().Err(err).Msg("failed to cleanup odd rgw secrets")
			errorsNumber++
		}
		rgwRemoved = rgwRemoved && certsRemoved
		if !c.cdConfig.clusterSpec.External.Enable {
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
