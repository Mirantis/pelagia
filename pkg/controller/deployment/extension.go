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
	// TODO: enable later
	//castedClusterSpec, err := c.cdConfig.cephDpl.Spec.Cluster.GetSpec()
	//if err != nil {
	//	return err
	//}
	//c.cdConfig.clusterSpec = &castedClusterSpec

	expandedNodes, err := lcmcommon.GetExpandedCephDeploymentNodeList(c.context, c.api.Client, c.cdConfig.cephDpl.Spec)
	if err != nil {
		return errors.Wrap(err, "spec: failed to expand nodes list")
	}
	c.cdConfig.nodesListExpanded = expandedNodes

	return nil
}
