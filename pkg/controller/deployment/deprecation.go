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
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/v3/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

var (
	msgTmpl    = "found deprecated field spec.%s, moving to spec.%s"
	errMsgTmpl = "found deprecated field spec.%s, but conflicts with spec.%s. Keep correct and remove not needed fields manually"
)

func isParamForRuntime(section, key string) bool {
	// use previously defined function to determine params is for runtime or for config file
	// operator can do additional alignment manually if needed once migration completed
	return strings.HasPrefix(key, "osd") ||
		strings.HasPrefix(key, "clog") ||
		strings.HasPrefix(section, "osd")
}

func (c *cephDeploymentConfig) isMigrationRequired() (bool, error) {
	migrated := false
	specErrors := 0
	if len(c.cdConfig.cephDpl.Spec.OldClients) > 0 {
		if len(c.cdConfig.cephDpl.Spec.Clients) > 0 {
			c.log.Error().Msgf(errMsgTmpl, "clients", "cephxClients")
			specErrors++
		} else {
			c.log.Warn().Msgf(msgTmpl, "clients", "cephxClients")
			newClients := make([]cephlcmv1alpha1.CephClient, len(c.cdConfig.cephDpl.Spec.OldClients))
			for idx, client := range c.cdConfig.cephDpl.Spec.OldClients {
				newClient := cephlcmv1alpha1.CephClient{ClientSpec: client.RawExtension}
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
				newClients[idx] = newClient
			}
			if specErrors == 0 {
				c.cdConfig.cephDpl.Spec.Clients = newClients
				c.cdConfig.cephDpl.Spec.OldClients = nil
				migrated = true
			}
		}
	}

	if len(c.cdConfig.cephDpl.Spec.RookConfig) > 0 {
		// allow both fields (deprecated and new) be present at the same time
		// but do not allow same options be present at the same time
		newRuntimeConfig := map[string]map[string]string{}
		newStaticConfig := map[string]map[string]string{}
		if c.cdConfig.cephDpl.Spec.ExtraOpts != nil && len(c.cdConfig.cephDpl.Spec.ExtraOpts.RuntimeCephConfig) > 0 {
			newRuntimeConfig = c.cdConfig.cephDpl.Spec.ExtraOpts.RuntimeCephConfig
		}
		if c.cdConfig.cephDpl.Spec.ExtraOpts != nil && len(c.cdConfig.cephDpl.Spec.ExtraOpts.StaticCephConfig) > 0 {
			newStaticConfig = c.cdConfig.cephDpl.Spec.ExtraOpts.StaticCephConfig
		}
		conflict := false
		for key, value := range c.cdConfig.cephDpl.Spec.RookConfig {
			items := strings.Split(key, sectionDelimiter)
			section := "global"
			option := items[0]
			if len(items) > 1 {
				section = items[0]
				option = items[1]
			}
			if isParamForRuntime(section, option) {
				if _, ok := newRuntimeConfig[section]; ok {
					if _, ok := newRuntimeConfig[section][option]; ok {
						c.log.Error().Msgf(errMsgTmpl, fmt.Sprintf("rookConfig.%s", key), fmt.Sprintf("extraOpts.runtimeCephConfig.%s.%s", section, option))
						specErrors++
						conflict = true
					} else {
						newRuntimeConfig[section][option] = value
					}
				} else {
					newRuntimeConfig[section] = map[string]string{option: value}
				}
			} else {
				if _, ok := newStaticConfig[section]; ok {
					if _, ok := newStaticConfig[section][option]; ok {
						c.log.Error().Msgf(errMsgTmpl, fmt.Sprintf("rookConfig.%s", key), fmt.Sprintf("extraOpts.staticCephConfig.%s.%s", section, option))
						specErrors++
						conflict = true
					} else {
						newStaticConfig[section][option] = value
					}
				} else {
					newStaticConfig[section] = map[string]string{option: value}
				}
			}
		}
		if !conflict {
			c.cdConfig.cephDpl.Spec.RookConfig = nil
			if len(newStaticConfig) > 0 {
				c.log.Warn().Msgf(msgTmpl, "rookConfig", "extraOpts.staticCephConfig")
				if c.cdConfig.cephDpl.Spec.ExtraOpts == nil {
					c.cdConfig.cephDpl.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{}
				}
				c.cdConfig.cephDpl.Spec.ExtraOpts.StaticCephConfig = newStaticConfig
				migrated = true
			}
			if len(newRuntimeConfig) > 0 {
				c.log.Warn().Msgf(msgTmpl, "rookConfig", "extraOpts.runtimeCephConfig")
				if c.cdConfig.cephDpl.Spec.ExtraOpts == nil {
					c.cdConfig.cephDpl.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{}
				}
				c.cdConfig.cephDpl.Spec.ExtraOpts.RuntimeCephConfig = newRuntimeConfig
				migrated = true
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
