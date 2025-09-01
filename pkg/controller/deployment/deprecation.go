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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *cephDeploymentConfig) ensureDeprecatedFields() error {
	fields := []string{}
	if len(fields) > 0 {
		log.Info().Msgf("dropping deprecated spec fields (%s) for CephDeployment %s/%s", strings.Join(fields, ", "), c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
		_, err := c.api.CephLcmclientset.LcmV1alpha1().CephDeployments(c.cdConfig.cephDpl.Namespace).Update(c.context, c.cdConfig.cephDpl, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to update CephDeployment spec")
		}
	}
	return nil
}
