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
	"github.com/pkg/errors"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentConfig) castExtensions() error {
	c.log.Debug().Msg("casting cephdeployment spec fields")
	castedClusterSpec, err := c.cdConfig.cephDpl.Spec.Cluster.GetSpec()
	if err != nil {
		c.log.Error().Err(err).Msg("failed to cast cephdeployment fields to Rook API")
		return err
	}
	c.cdConfig.clusterSpec = &castedClusterSpec

	if c.cdConfig.cephDpl.Spec.BlockStorage != nil {
		c.cdConfig.openstackSetup = lcmcommon.IsOpenStackPoolsPresent(c.cdConfig.cephDpl.Spec.BlockStorage.Pools)

		c.cdConfig.pools = make([]string, len(c.cdConfig.cephDpl.Spec.BlockStorage.Pools))
		for idx, pool := range c.cdConfig.cephDpl.Spec.BlockStorage.Pools {
			c.cdConfig.pools[idx] = buildPoolName(pool)
		}
	}

	expandedNodes, err := lcmcommon.GetExpandedCephDeploymentNodeList(c.context, c.api.Client, c.cdConfig.cephDpl.Spec)
	if err != nil {
		return errors.Wrap(err, "spec: failed to expand nodes list")
	}
	c.cdConfig.nodesListExpanded = expandedNodes

	return nil
}
