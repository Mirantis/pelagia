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
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *cephDeploymentConfig) isMigrationRequired() bool {
	return false
}

//var (
//	msgTmpl    = "found deprecated field spec.%s, moving to spec.%s"
//	errMsgTmpl = "found deprecated field spec.%s, but conflicts with spec.%s. Keep correct and remove not needed fields manually"
//)

func (c *cephDeploymentConfig) ensureDeprecatedFields() (bool, error) {
	if !c.isMigrationRequired() {
		return false, nil
	}

	c.log.Info().Msgf("removing deprecated params from CephDeployment %s/%s spec", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
	_, err := c.api.CephLcmclientset.LcmV1alpha1().CephDeployments(c.cdConfig.cephDpl.Namespace).Update(c.context, c.cdConfig.cephDpl, metav1.UpdateOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to update CephDeployment spec")
	}
	return true, nil
}
