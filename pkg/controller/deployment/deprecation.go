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
	"k8s.io/apimachinery/pkg/runtime"
)

//var (
//	msgTmpl    = "found deprecated field spec.%s, moving to spec.%s"
//	errMsgTmpl = "found deprecated field spec.%s, but conflicts with spec.%s. Keep correct and remove not needed fields manually"
//)

func (c *cephDeploymentConfig) isMigrationRequired() (bool, error) {
	migrated := false
	specErrors := 0
	if len(c.cdConfig.cephDpl.Spec.Clients) > 0 {
		for idx, client := range c.cdConfig.cephDpl.Spec.Clients {
			if client.OldSpec.Raw != nil || client.OldSpec.Object != nil {
				if client.ClientSpec.Raw != nil || client.ClientSpec.Object != nil {
					c.log.Error().Msgf("spec item spec.clients[%[1]d] contains conflicting fields with spec.clients[%[1]d].spec. Put CephClient config under spec.clients[%[1]d].spec field manually", idx)
					specErrors++
					continue
				}
				newClient := client.DeepCopy()
				newClient.ClientSpec.Raw = client.OldSpec.Raw
				newClient.ClientSpec.Object = client.OldSpec.Object
				newClient.OldSpec = runtime.RawExtension{}
				clientSpec, err := newClient.GetSpec()
				if err != nil {
					c.log.Error().Err(err).Msgf("failed to decode client item #%d", idx)
					specErrors++
					continue
				}
				// check for backward compatibility with previously created clients for MOSK
				switch clientSpec.Name {
				case "nova", "glance", "cinder", "manila":
					newClient.Role = clientSpec.Name
				}
				c.log.Warn().Msgf("spec item spec.clients[%[1]d] contains deprecated config, moving under spec.clients[%[1]d].spec field", idx)
				migrated = true
				c.cdConfig.cephDpl.Spec.Clients[idx] = *newClient
			}
		}
	}
	if specErrors > 0 {
		return false, errors.New("failed to verify spec for deprecated fields")
	}
	return migrated, nil
}

func (c *cephDeploymentConfig) ensureDeprecatedFields() (bool, error) {
	migrated, err := c.isMigrationRequired()
	if err != nil {
		return false, err
	}

	if migrated {
		c.log.Info().Msgf("removing deprecated params from CephDeployment %s/%s spec", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
		_, err := c.api.CephLcmclientset.LcmV1alpha1().CephDeployments(c.cdConfig.cephDpl.Namespace).Update(c.context, c.cdConfig.cephDpl, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to update CephDeployment spec")
		}
	}
	return migrated, nil
}
