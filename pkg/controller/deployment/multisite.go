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
	"reflect"
	"strings"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

func (c *cephDeploymentConfig) ensureRgwMultiSite() (bool, error) {
	c.log.Debug().Msg("ensure rgw multisite")
	multisiteChanged := false
	changed, err := c.ensureRealms()
	if err != nil {
		c.log.Error().Err(err).Msg("error ensuring rgw realms")
		return false, errors.Wrap(err, "failed to ensure realms")
	}
	multisiteChanged = multisiteChanged || changed
	changed, err = c.ensureZoneGroups()
	if err != nil {
		c.log.Error().Err(err).Msg("error ensuring rgw zonegroups")
		return false, errors.Wrap(err, "failed to ensure zone groups")
	}
	multisiteChanged = multisiteChanged || changed
	changed, err = c.ensureZones()
	if err != nil {
		c.log.Error().Err(err).Msg("error ensuring rgw zones")
		return false, errors.Wrap(err, "failed to ensure zones")
	}
	multisiteChanged = multisiteChanged || changed

	return multisiteChanged, nil
}

func (c *cephDeploymentConfig) ensureRealms() (bool, error) {
	c.log.Debug().Msgf("ensure rgw realms for %s/%s", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
	realms := c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite.Realms
	realmsReal, err := c.api.Rookclientset.CephV1().CephObjectRealms(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get list CephObjectRealms in '%s' namespace", c.lcmConfig.RookNamespace)
	}
	zoneGroupsReal, err := c.api.Rookclientset.CephV1().CephObjectZoneGroups(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get list CephObjectZoneGroups in '%s' namespace", c.lcmConfig.RookNamespace)
	}
	realmsInUse := map[string]string{}
	for _, zoneGroup := range zoneGroupsReal.Items {
		realmsInUse[zoneGroup.Spec.Realm] = zoneGroup.Name
	}

	realmsToCreate := make([]cephlcmv1alpha1.CephRGWRealm, 0)
	realmsToUpdate := make([]cephv1.CephObjectRealm, 0)
	realmsToDelete := map[string]bool{}
	secretsToUpdate := []*v1.Secret{}
	for _, realm := range realmsReal.Items {
		realmsToDelete[realm.Name] = true
	}

	getRealmSecretData := func(accessKey, secretKey string) map[string][]byte {
		return map[string][]byte{
			"access-key": []byte(accessKey),
			"secret-key": []byte(secretKey),
		}
	}

	createSecret := func(secretName, accessKey, secretKey string) error {
		c.log.Info().Msgf("creating Secret '%s/%s'", c.lcmConfig.RookNamespace, secretName)
		secretRealm := v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: c.lcmConfig.RookNamespace,
			},
			Data: getRealmSecretData(accessKey, secretKey),
		}
		_, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Create(c.context, &secretRealm, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to create Secret '%s/%s'", c.lcmConfig.RookNamespace, secretName)
		}
		return nil
	}

	// find the default realm - it is a realm with no pull spec and used in rgw
	defaultRealmName := ""
	if c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Zone != nil {
		for _, zone := range c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite.Zones {
			if c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Zone.Name == zone.Name {
				zoneGroupName := zone.ZoneGroup
				for _, zoneGroup := range c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite.ZoneGroups {
					if zoneGroup.Name == zoneGroupName {
						realmName := zoneGroup.Realm
						for _, realm := range c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite.Realms {
							if realm.Name == realmName {
								defaultRealmName = realm.Name
								break
							}
						}
						break
					}
				}
				break
			}
		}
	}

	errCollector := make([]string, 0)
	for _, realm := range realms {
		found := false
		for _, existingRealm := range realmsReal.Items {
			if realm.Name == existingRealm.Name {
				found = true
				delete(realmsToDelete, realm.Name)
				pullEndpoint := ""
				if realm.Pull != nil {
					pullEndpoint = realm.Pull.Endpoint
					secretName := fmt.Sprintf("%s-keys", realm.Name)
					secret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, secretName, metav1.GetOptions{})
					if err != nil {
						if apierrors.IsNotFound(err) {
							c.log.Warn().Msgf("secret '%s/%s' is not found for present CephObjectRealm '%s', recreating", c.lcmConfig.RookNamespace, secretName, realm.Name)
							secretErr := createSecret(fmt.Sprintf("%s-keys", realm.Name), realm.Pull.AccessKey, realm.Pull.SecretKey)
							if secretErr != nil {
								c.log.Error().Err(secretErr).Msg("failed to create cephobjectrealm secret")
								errCollector = append(errCollector, secretErr.Error())
								continue
							}
						} else {
							msg := fmt.Sprintf("failed to get secret '%s/%s' for CephObjectRealm '%s': %s", c.lcmConfig.RookNamespace, secretName, realm.Name, err.Error())
							c.log.Error().Err(err).Msg(msg)
							errCollector = append(errCollector, msg)
							continue
						}
					} else {
						newSecretData := getRealmSecretData(realm.Pull.AccessKey, realm.Pull.SecretKey)
						if !reflect.DeepEqual(newSecretData, secret.Data) {
							secret.Data = newSecretData
							secretsToUpdate = append(secretsToUpdate, secret)
						}
					}
				}
				changed := false
				if pullEndpoint != existingRealm.Spec.Pull.Endpoint {
					existingRealm.Spec.Pull.Endpoint = pullEndpoint
					changed = true
				}
				if defaultRealmName == existingRealm.Name && !existingRealm.Spec.DefaultRealm {
					existingRealm.Spec.DefaultRealm = true
					changed = true
				}
				if changed {
					realmsToUpdate = append(realmsToUpdate, existingRealm)
				}
			}
		}
		if !found {
			realm.DefaultRealm = defaultRealmName == realm.Name
			realmsToCreate = append(realmsToCreate, realm)
		}
	}

	changed := len(realmsToCreate) > 0 || len(realmsToUpdate) > 0 || len(secretsToUpdate) > 0
	for _, realm := range realmsToCreate {
		realmResource := cephv1.CephObjectRealm{
			ObjectMeta: metav1.ObjectMeta{
				Name:      realm.Name,
				Namespace: c.lcmConfig.RookNamespace,
			},
		}
		if realm.Pull != nil {
			realmResource.Spec = cephv1.ObjectRealmSpec{
				Pull: cephv1.PullSpec{
					Endpoint: realm.Pull.Endpoint,
				},
			}
			secretErr := createSecret(fmt.Sprintf("%s-keys", realm.Name), realm.Pull.AccessKey, realm.Pull.SecretKey)
			if secretErr != nil {
				c.log.Error().Err(secretErr).Msg("failed to create cephobjectrealm secret")
				errCollector = append(errCollector, secretErr.Error())
				continue
			}
		}
		if realm.DefaultRealm {
			realmResource.Spec.DefaultRealm = true
		}
		c.log.Info().Msgf("creating CephObjectRealm '%s/%s'", c.lcmConfig.RookNamespace, realm.Name)
		_, err := c.api.Rookclientset.CephV1().CephObjectRealms(c.lcmConfig.RookNamespace).Create(c.context, &realmResource, metav1.CreateOptions{})
		if err != nil {
			msg := fmt.Sprintf("failed to create CephObjectRealm '%s/%s': %s", c.lcmConfig.RookNamespace, realm.Name, err.Error())
			c.log.Error().Err(err).Msg(msg)
			errCollector = append(errCollector, msg)
		}
	}

	for _, realm := range realmsToUpdate {
		c.log.Info().Msgf("updating CephObjectRealm '%s/%s'", c.lcmConfig.RookNamespace, realm.Name)
		_, err := c.api.Rookclientset.CephV1().CephObjectRealms(c.lcmConfig.RookNamespace).Update(c.context, &realm, metav1.UpdateOptions{})
		if err != nil {
			msg := fmt.Sprintf("failed to update CephObjectRealm '%s/%s': %s", c.lcmConfig.RookNamespace, realm.Name, err.Error())
			c.log.Error().Err(err).Msg(msg)
			errCollector = append(errCollector, msg)
		}
	}

	for idx := range secretsToUpdate {
		c.log.Info().Msgf("updating Secret '%s/%s'", c.lcmConfig.RookNamespace, secretsToUpdate[idx].Name)
		_, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Update(c.context, secretsToUpdate[idx], metav1.UpdateOptions{})
		if err != nil {
			msg := fmt.Sprintf("failed to update Secret '%s/%s': %s", c.lcmConfig.RookNamespace, secretsToUpdate[idx].Name, err.Error())
			c.log.Error().Err(err).Msg(msg)
			errCollector = append(errCollector, msg)
		}
	}

	for realm := range realmsToDelete {
		if realmsInUse[realm] != "" {
			c.log.Error().Msgf("can't remove CephObjectRealm '%s' since it is used by CephObjectZoneGroup '%s/%s'", realm, c.lcmConfig.RookNamespace, realmsInUse[realm])
		} else {
			changed = true
			if err = c.deleteRealm(realm); err != nil {
				c.log.Error().Err(err).Msgf("failed to remove realm %q", realm)
				errCollector = append(errCollector, err.Error())
			}
		}
	}

	// Return error if exists
	if len(errCollector) > 0 {
		return false, errors.New(strings.Join(errCollector, ", "))
	}
	return changed, nil
}

func (c *cephDeploymentConfig) ensureZoneGroups() (bool, error) {
	c.log.Debug().Msgf("ensuring zonegroups for %s/%s", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
	zoneGroups := c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite.ZoneGroups
	zoneGroupsReal, err := c.api.Rookclientset.CephV1().CephObjectZoneGroups(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get list CephObjectZoneGroups in '%s' namespace", c.lcmConfig.RookNamespace)
	}
	zonesReal, err := c.api.Rookclientset.CephV1().CephObjectZones(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get list CephObjectZones in '%s' namespace", c.lcmConfig.RookNamespace)
	}
	zoneGroupsInUse := map[string]string{}
	for _, zone := range zonesReal.Items {
		zoneGroupsInUse[zone.Spec.ZoneGroup] = zone.Name
	}

	zoneGroupsToCreate := make([]cephlcmv1alpha1.CephRGWZoneGroup, 0)
	zoneGroupsToDelete := map[string]bool{}
	for _, zoneGroup := range zoneGroupsReal.Items {
		zoneGroupsToDelete[zoneGroup.Name] = true
	}

	for _, zoneGroup := range zoneGroups {
		found := false
		for _, existingZoneGroup := range zoneGroupsReal.Items {
			if zoneGroup.Name == existingZoneGroup.Name {
				found = true
				delete(zoneGroupsToDelete, zoneGroup.Name)
			}
		}
		if !found {
			zoneGroupsToCreate = append(zoneGroupsToCreate, zoneGroup)
		}
	}

	errCollector := make([]string, 0)
	changed := len(zoneGroupsToCreate) > 0
	for _, zoneGroup := range zoneGroupsToCreate {
		zoneGroupResource := cephv1.CephObjectZoneGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      zoneGroup.Name,
				Namespace: c.lcmConfig.RookNamespace,
			},
			Spec: cephv1.ObjectZoneGroupSpec{
				Realm: zoneGroup.Realm,
			},
		}
		c.log.Info().Msgf("creating CephObjectZoneGroup %q", zoneGroup.Name)
		_, err := c.api.Rookclientset.CephV1().CephObjectZoneGroups(c.lcmConfig.RookNamespace).Create(c.context, &zoneGroupResource, metav1.CreateOptions{})
		if err != nil {
			msg := fmt.Sprintf("failed to create CephObjectZoneGroup '%s/%s': %s", c.lcmConfig.RookNamespace, zoneGroup.Name, err.Error())
			c.log.Error().Err(err).Msg(msg)
			errCollector = append(errCollector, msg)
		}
	}

	for zoneGroup := range zoneGroupsToDelete {
		if zoneGroupsInUse[zoneGroup] != "" {
			c.log.Error().Msgf("can't remove CephObjectZoneGroup '%s' since it is used by CephObjectZone '%s/%s'", zoneGroup, c.lcmConfig.RookNamespace, zoneGroupsInUse[zoneGroup])
		} else {
			changed = true
			if err = c.deleteZoneGroup(zoneGroup); err != nil {
				c.log.Error().Err(err).Msgf("failed to remove CephObjectZoneGroup %q", zoneGroup)
				errCollector = append(errCollector, err.Error())
			}
		}
	}

	// Return error if exists
	if len(errCollector) > 0 {
		return false, errors.New(strings.Join(errCollector, ", "))
	}
	return changed, nil
}

func (c *cephDeploymentConfig) ensureZones() (bool, error) {
	c.log.Debug().Msgf("ensuring zones for %s/%s", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
	zones := c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite.Zones
	zonesReal, err := c.api.Rookclientset.CephV1().CephObjectZones(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get list CephObjectZones in '%s' namespace", c.lcmConfig.RookNamespace)
	}
	zonesInUse, err := c.getZonesInUse()
	if err != nil {
		c.log.Error().Err(err).Msg("failed to check zones in use")
		return false, errors.Wrap(err, "failed to check zones in use")
	}
	proxyDeployed, _, err := c.canDeployIngressProxy()
	if err != nil {
		return false, errors.Wrap(err, "failed to check ingress proxy setup")
	}
	zonesToCreate := make([]cephv1.CephObjectZone, 0)
	zonesToUpdate := make([]cephv1.CephObjectZone, 0)
	zonesToDelete := map[string]bool{}
	for _, zone := range zonesReal.Items {
		zonesToDelete[zone.Name] = true
	}

	preservePoolsOnDelete := c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.PreservePoolsOnDelete
	for _, zone := range zones {
		found := false
		zoneResource := cephv1.CephObjectZone{
			ObjectMeta: metav1.ObjectMeta{
				Name:      zone.Name,
				Namespace: c.lcmConfig.RookNamespace,
			},
			Spec: cephv1.ObjectZoneSpec{
				ZoneGroup:             zone.ZoneGroup,
				MetadataPool:          *generatePoolSpec(&zone.MetadataPool, "rgw metadata"),
				DataPool:              *generatePoolSpec(&zone.DataPool, "rgw data"),
				CustomEndpoints:       zone.EndpointsForZone,
				PreservePoolsOnDelete: preservePoolsOnDelete,
			},
		}
		if len(zoneResource.Spec.CustomEndpoints) == 0 && zonesInUse[zone.Name] != "" {
			// if no endpoints specified - put default external lb ip and port as endpoint
			// in case of using ingress - no default, user has to add endpoints manually
			// in cose of no public access - nothing to do
			if c.lcmConfig.DeployParams.RgwPublicAccessLabel != "" {
				if proxyDeployed {
					c.log.Warn().Msgf("detected ingress proxy usage, but zone '%s' has no endpoints specified", zone.Name)
				} else {
					externalSvcName := buildRGWName(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, "external")
					externalSvc, err := c.api.Kubeclientset.CoreV1().Services(c.lcmConfig.RookNamespace).Get(c.context, externalSvcName, metav1.GetOptions{})
					if err != nil {
						if !apierrors.IsNotFound(err) {
							c.log.Error().Err(err).Msgf("failed to get ip of external service %q", externalSvcName)
							return false, errors.Wrap(err, "failed to get ip of external service")
						}
						c.log.Warn().Msgf("zone '%s' has no endpoints specified, service '%s' is not created yet, leaving empty", zone.Name, externalSvcName)
					} else {
						c.log.Warn().Msgf("zone '%s' has no endpoints specified, using service '%s' external ip address and http port as endpoint if available", zone.Name, externalSvcName)
						if len(externalSvc.Status.LoadBalancer.Ingress) > 0 {
							zoneResource.Spec.CustomEndpoints = []string{fmt.Sprintf("http://%s:80", externalSvc.Status.LoadBalancer.Ingress[0].IP)}
						}
					}
				}
			} else {
				c.log.Warn().Msgf("zone '%s' has no endpoints specified, no public access for rgw configured", zone.Name)
			}
		}
		for _, existingZone := range zonesReal.Items {
			if zone.Name == existingZone.Name {
				found = true
				delete(zonesToDelete, zone.Name)
				if !reflect.DeepEqual(zoneResource.Spec, existingZone.Spec) {
					c.log.Info().Msgf("update detected for CephObjectZone '%s'", zone.Name)
					lcmcommon.ShowObjectDiff(*c.log, existingZone.Spec, zoneResource.Spec)
					existingZone.Spec = zoneResource.Spec
					zonesToUpdate = append(zonesToUpdate, existingZone)
				}
			}
		}
		if !found {
			zonesToCreate = append(zonesToCreate, zoneResource)
		}
	}

	errCollector := make([]string, 0)
	changed := len(zonesToCreate) > 0 || len(zonesToUpdate) > 0
	for _, zone := range zonesToCreate {
		c.log.Info().Msgf("creating CephObjectZone %q", zone.Name)
		_, err = c.api.Rookclientset.CephV1().CephObjectZones(c.lcmConfig.RookNamespace).Create(c.context, &zone, metav1.CreateOptions{})
		if err != nil {
			msg := fmt.Sprintf("failed to create CephObjectZone '%s/%s': %s", c.lcmConfig.RookNamespace, zone.Name, err.Error())
			c.log.Error().Err(err).Msg(msg)
			errCollector = append(errCollector, msg)
		}
	}

	for _, zone := range zonesToUpdate {
		c.log.Info().Msgf("updating CephObjectZone %q", zone.Name)
		c.log.Warn().Msgf("update '%s' CephObjectZone's data and metadata pools are not reflected on the Ceph cluster, do update manually if needed", zone.Name)
		_, err = c.api.Rookclientset.CephV1().CephObjectZones(c.lcmConfig.RookNamespace).Update(c.context, &zone, metav1.UpdateOptions{})
		if err != nil {
			msg := fmt.Sprintf("failed to update CephObjectZone '%s/%s': %s", c.lcmConfig.RookNamespace, zone.Name, err.Error())
			c.log.Error().Err(err).Msg(msg)
			errCollector = append(errCollector, msg)
		}
	}

	for zone := range zonesToDelete {
		if zonesInUse[zone] != "" {
			c.log.Error().Msgf("can't remove CephObjectZone '%s' since it is used by CephObjectStore '%s/%s'", zone, c.lcmConfig.RookNamespace, zonesInUse[zone])
		} else {
			changed = true
			if err := c.deleteZone(zone); err != nil {
				c.log.Error().Err(err).Msgf("failed to delete zone %q", zone)
				errCollector = append(errCollector, err.Error())
			}
		}
	}

	// Return error if exists
	if len(errCollector) > 0 {
		return false, errors.New(strings.Join(errCollector, ", "))
	}
	return changed, nil
}

func (c *cephDeploymentConfig) deleteMultiSite() (bool, error) {
	zones, err := c.api.Rookclientset.CephV1().CephObjectZones(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("failed to list cephobjectzones")
		return false, errors.Wrapf(err, "failed to get list CephObjectZones in '%s' namespace", c.lcmConfig.RookNamespace)
	}
	zoneGroups, err := c.api.Rookclientset.CephV1().CephObjectZoneGroups(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("failed to list cephobjectzonegroups")
		return false, errors.Wrapf(err, "failed to get list CephObjectZoneGroups in '%s' namespace", c.lcmConfig.RookNamespace)
	}
	realms, err := c.api.Rookclientset.CephV1().CephObjectRealms(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("failed to list cephobjectrealms")
		return false, errors.Wrapf(err, "failed to get list CephObjectRealms in '%s' namespace", c.lcmConfig.RookNamespace)
	}
	zonesInUse, err := c.getZonesInUse()
	if err != nil {
		c.log.Error().Err(err).Msg("failed to check zones in use")
		return false, errors.Wrap(err, "failed to check zones in use")
	}
	errCollector := []string{}
	zoneGroupsInUse := map[string]string{}
	multisiteResourcesRemoved := true
	for _, zone := range zones.Items {
		multisiteResourcesRemoved = false
		zoneGroupsInUse[zone.Spec.ZoneGroup] = zone.Name
		if zonesInUse[zone.Name] != "" {
			c.log.Info().Msgf("detected CephObjectStore '%s/%s' is using CephObjectZone '%s', waiting until its removed to proceed zone cleanup", c.lcmConfig.RookNamespace, zonesInUse[zone.Name], zone.Name)
			continue
		}
		if err = c.deleteZone(zone.Name); err != nil {
			c.log.Error().Err(err).Msgf("failed to delete zone %q", zone.Name)
			errCollector = append(errCollector, err.Error())
		}
	}
	realmsInUse := map[string]string{}
	for _, zoneGroup := range zoneGroups.Items {
		multisiteResourcesRemoved = false
		realmsInUse[zoneGroup.Spec.Realm] = zoneGroup.Name
		if zoneGroupsInUse[zoneGroup.Name] != "" {
			c.log.Info().Msgf("detected CephObjectZoneGroup '%s' is used by CephObjectZone '%s', waiting until its removed to proceed zone group cleanup", zoneGroup.Name, zoneGroupsInUse[zoneGroup.Name])
			continue
		}
		if err = c.deleteZoneGroup(zoneGroup.Name); err != nil {
			c.log.Error().Err(err).Msgf("failed to delete zonegroup %q", zoneGroup.Name)
			errCollector = append(errCollector, err.Error())
		}
	}
	for _, realm := range realms.Items {
		multisiteResourcesRemoved = false
		if realmsInUse[realm.Name] != "" {
			c.log.Info().Msgf("detected CephObjectRealm '%s' is used by CephObjectZoneGroup '%s', waiting until its removed to proceed realm cleanup", realm.Name, realmsInUse[realm.Name])
			continue
		}
		if err = c.deleteRealm(realm.Name); err != nil {
			c.log.Error().Err(err).Msgf("failed to delete realm %q", realm.Name)
			errCollector = append(errCollector, err.Error())
		}
	}
	if len(errCollector) > 0 {
		return false, errors.Errorf("failed to cleanup multisite: %s", strings.Join(errCollector, ", "))
	}
	return multisiteResourcesRemoved, nil
}

func (c *cephDeploymentConfig) deleteZone(zone string) error {
	c.log.Info().Msgf("removing CephObjectZone '%s/%s'", c.lcmConfig.RookNamespace, zone)
	err := c.api.Rookclientset.CephV1().CephObjectZones(c.lcmConfig.RookNamespace).Delete(c.context, zone, metav1.DeleteOptions{})
	if err == nil || apierrors.IsNotFound(err) {
		return nil
	}
	return errors.Wrapf(err, "failed to delete CephObjectZone '%s/%s'", c.lcmConfig.RookNamespace, zone)
}

func (c *cephDeploymentConfig) deleteZoneGroup(zoneGroup string) error {
	c.log.Info().Msgf("removing CephObjectZoneGroup '%s/%s'", c.lcmConfig.RookNamespace, zoneGroup)
	// Rook doesn't remove zonegroup - so we need to clean up it manually
	// may be someday it will be fixed in Rook, but v1.17 - still not
	var zoneGroupCmd struct {
		ZoneGroups []string `json:"zonegroups"`
	}
	err := lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, "radosgw-admin zonegroup list", &zoneGroupCmd)
	if err != nil {
		return errors.Wrap(err, "failed to check zonegroup list")
	}
	for _, zoneGroupPresent := range zoneGroupCmd.ZoneGroups {
		if zoneGroupPresent == zoneGroup {
			cmd := fmt.Sprintf("radosgw-admin zonegroup delete --rgw-zonegroup=%s", zoneGroup)
			_, err = lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd)
			if err != nil {
				return errors.Wrapf(err, "failed to remove zonegroup '%s'", zoneGroup)
			}
		}
	}
	err = c.api.Rookclientset.CephV1().CephObjectZoneGroups(c.lcmConfig.RookNamespace).Delete(c.context, zoneGroup, metav1.DeleteOptions{})
	if err == nil || apierrors.IsNotFound(err) {
		return nil
	}
	return errors.Wrapf(err, "failed to delete CephObjectZoneGroup '%s/%s'", c.lcmConfig.RookNamespace, zoneGroup)
}

func (c *cephDeploymentConfig) deleteRealm(realm string) error {
	secretKey := fmt.Sprintf("%s-keys", realm)
	c.log.Info().Msgf("removing Secret '%s/%s' for CephObjectRealm '%s/%s'", c.lcmConfig.RookNamespace, secretKey, c.lcmConfig.RookNamespace, realm)
	err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Delete(c.context, secretKey, metav1.DeleteOptions{})
	if err == nil || apierrors.IsNotFound(err) {
		c.log.Info().Msgf("removing CephObjectRealm '%s/%s'", c.lcmConfig.RookNamespace, realm)
		// Rook doesn't remove realm - so we need to clean up it manually
		// may be someday it will be fixed in Rook, but v1.17 - still not
		var realmCmd struct {
			Realms []string `json:"realms"`
		}
		err = lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, "radosgw-admin realm list", &realmCmd)
		if err != nil {
			return errors.Wrap(err, "failed to check realm list")
		}
		for _, realmPresent := range realmCmd.Realms {
			if realmPresent == realm {
				cmd := fmt.Sprintf("radosgw-admin realm rm --rgw-realm=%s", realm)
				_, err = lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd)
				if err != nil {
					return errors.Wrapf(err, "failed to remove realm '%s'", realm)
				}
			}
		}
		err = c.api.Rookclientset.CephV1().CephObjectRealms(c.lcmConfig.RookNamespace).Delete(c.context, realm, metav1.DeleteOptions{})
		if err == nil || apierrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "failed to delete CephObjectRealm '%s/%s'", c.lcmConfig.RookNamespace, realm)
	}
	return errors.Wrapf(err, "failed to delete Secret '%s/%s' for CephObjectRealm '%s/%s'", c.lcmConfig.RookNamespace, secretKey, c.lcmConfig.RookNamespace, realm)
}

func (c *cephDeploymentConfig) getZonesInUse() (map[string]string, error) {
	zonesInUse := map[string]string{}
	rgwList, err := c.api.Rookclientset.CephV1().CephObjectStores(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, rgw := range rgwList.Items {
		if rgw.Spec.Zone.Name != "" {
			zonesInUse[rgw.Spec.Zone.Name] = rgw.Name
		} else {
			zonesInUse[rgw.Name] = rgw.Name
		}
	}
	return zonesInUse, nil
}
