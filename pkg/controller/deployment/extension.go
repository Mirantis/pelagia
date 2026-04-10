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
	"strings"

	"github.com/pkg/errors"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentConfig) castExtensions() error {
	c.log.Debug().Msg("casting cephdeployment spec fields")
	errs := []string{}

	putError := func(msg string, err error) {
		c.log.Error().Err(err).Msg(msg)
		errs = append(errs, msg)
	}

	castedClusterSpec, err := c.cdConfig.cephDpl.Spec.Cluster.GetSpec()
	if err != nil {
		putError("failed to cast cephdeployment fields to Rook API", err)
	} else {
		c.cdConfig.clusterSpec = &castedClusterSpec
	}

	if c.cdConfig.cephDpl.Spec.BlockStorage != nil {
		c.cdConfig.openstackSetup = lcmcommon.IsOpenStackPoolsPresent(c.cdConfig.cephDpl.Spec.BlockStorage.Pools)

		c.cdConfig.pools = make([]string, len(c.cdConfig.cephDpl.Spec.BlockStorage.Pools))
		for idx, pool := range c.cdConfig.cephDpl.Spec.BlockStorage.Pools {
			_, err := pool.GetSpec()
			if err != nil {
				putError(fmt.Sprintf("failed to cast block storage pool '%s' fields to Rook API", pool.Name), err)
			} else {
				c.cdConfig.pools[idx] = buildPoolName(pool)
			}
		}
	}

	for idx, client := range c.cdConfig.cephDpl.Spec.Clients {
		_, err := client.GetSpec()
		if err != nil {
			putError(fmt.Sprintf("failed to cast client #%d to Rook API", idx), err)
		}
	}

	if c.cdConfig.cephDpl.Spec.ObjectStorage != nil {
		for _, rgw := range c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws {
			_, err := rgw.GetSpec()
			if err != nil {
				putError(fmt.Sprintf("failed to cast rgw '%s' to Rook API", rgw.Name), err)
			}
		}
		for _, user := range c.cdConfig.cephDpl.Spec.ObjectStorage.Users {
			_, err := user.GetSpec()
			if err != nil {
				putError(fmt.Sprintf("failed to cast user '%s' to Rook API", user.Name), err)
			}
		}
		for _, realm := range c.cdConfig.cephDpl.Spec.ObjectStorage.Realms {
			_, err := realm.GetSpec()
			if err != nil {
				putError(fmt.Sprintf("failed to cast realm '%s' to Rook API", realm.Name), err)
			}
		}
		for _, zonegroup := range c.cdConfig.cephDpl.Spec.ObjectStorage.Zonegroups {
			_, err := zonegroup.GetSpec()
			if err != nil {
				putError(fmt.Sprintf("failed to cast zonegroup '%s' to Rook API", zonegroup.Name), err)
			}
		}
		for _, zone := range c.cdConfig.cephDpl.Spec.ObjectStorage.Zones {
			_, err := zone.GetSpec()
			if err != nil {
				putError(fmt.Sprintf("failed to cast zone '%s' to Rook API", zone.Name), err)
			}
		}
	}

	if c.cdConfig.cephDpl.Spec.SharedFilesystem != nil {
		for _, cephfs := range c.cdConfig.cephDpl.Spec.SharedFilesystem.Filesystems {
			_, err := cephfs.GetSpec()
			if err != nil {
				putError(fmt.Sprintf("failed to cast ceph filesystem '%s' to Rook API", cephfs.Name), err)
			}
		}
	}

	expandedNodes, err := lcmcommon.GetExpandedCephDeploymentNodeList(c.context, c.api.Client, c.cdConfig.cephDpl.Spec)
	if err != nil {
		putError("failed to expand nodes list", err)
	} else {
		c.cdConfig.nodesListExpanded = expandedNodes
	}

	if len(errs) > 0 {
		return errors.Errorf("failed to cast spec fields: %s", strings.Join(errs, ", "))
	}
	return nil
}
