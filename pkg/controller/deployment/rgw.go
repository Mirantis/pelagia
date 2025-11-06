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
	"sort"
	"strings"

	bktv1alpha1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	rookUtils "github.com/rook/rook/pkg/operator/k8sutil"
	v1 "k8s.io/api/core/v1"
	v1storage "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentConfig) ensureRgw() (bool, error) {
	c.log.Debug().Msg("ensure rgw")
	// Ensure that we have rgw designed by spec
	err := c.ensureRgwConsistence()
	if err != nil {
		return false, err
	}

	// Skip rgw reconcile if rgw is not ready
	c.log.Debug().Msg("check rgw object store status")
	rgwStoreExists, err := c.statusRgw()
	if err != nil {
		return false, errors.Wrap(err, "failed to ensure rgw")
	}

	rgwConfigurationChanged := false
	c.log.Debug().Msg("ensure rgw ssl cert")
	changed, err := c.ensureRgwInternalSslCert()
	if err != nil {
		return false, errors.Wrap(err, "failed to ensure rgw ssl cert")
	}
	rgwConfigurationChanged = rgwConfigurationChanged || changed

	// Ensure rgw object store
	c.log.Debug().Msg("ensure rgw object store")
	changed, err = c.ensureRgwObject()
	if err != nil {
		return false, errors.Wrap(err, "failed to ensure rgw object store")
	}
	rgwConfigurationChanged = rgwConfigurationChanged || changed

	errMsg := make([]error, 0)
	if !c.cdConfig.cephDpl.Spec.External && rgwStoreExists {
		// we are not support auto hostname update if:
		// - version below octopus, that means migration is in progress
		// - multisite case and hostnames should be updated manually with script
		if c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Zone == nil {
			c.log.Debug().Msg("ensure rgw zonegroup hostnames")
			changed, err = c.ensureDefaultZoneGroupHostnames()
			if err != nil {
				err = errors.Wrap(err, "failed to ensure rgw zonegroup hostnames")
				c.log.Error().Err(err).Msg("failed to ensure rgw zonegroup hostnames")
				errMsg = append(errMsg, err)
			}
			rgwConfigurationChanged = rgwConfigurationChanged || changed
		}
	}

	// Create rgw storageClass if necessary
	c.log.Debug().Msg("ensure rgw storage class")
	changed, err = c.ensureRgwStorageClass()
	if err != nil {
		c.log.Error().Err(err).Msg("failed to ensure rgw storage class")
		errMsg = append(errMsg, errors.Wrap(err, "failed to ensure rgw storage class"))
	}
	rgwConfigurationChanged = rgwConfigurationChanged || changed

	// Create/delete rgw buckets
	c.log.Debug().Msg("ensure rgw buckets")
	changed, err = c.ensureRgwBuckets()
	if err != nil {
		c.log.Error().Err(err).Msg("failed to ensure rgw buckets")
		errMsg = append(errMsg, errors.Wrap(err, "failed to ensure rgw buckets"))
	}
	rgwConfigurationChanged = rgwConfigurationChanged || changed

	// if openstack pools are present - create ceilomenter metrics user as well
	if lcmcommon.IsOpenStackPoolsPresent(c.cdConfig.cephDpl.Spec.Pools) && !c.cdConfig.cephDpl.Spec.External {
		serviceUsers := []cephlcmv1alpha1.CephRGWUser{
			{
				Name:        rgwMetricsUser,
				DisplayName: rgwMetricsUser,
				Capabilities: &cephv1.ObjectUserCapSpec{
					User:     "read",
					Bucket:   "read",
					MetaData: "read",
					Usage:    "read",
				},
			},
		}
		c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.ObjectUsers = append(serviceUsers, c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.ObjectUsers...)
	}
	// Create/delete rgw users
	c.log.Debug().Msg("ensure rgw users")
	changed, err = c.ensureRgwUsers()
	if err != nil {
		c.log.Error().Err(err).Msg("failed to ensure rgw users")
		errMsg = append(errMsg, errors.Wrap(err, "failed to ensure rgw users"))
	}
	rgwConfigurationChanged = rgwConfigurationChanged || changed

	if !c.cdConfig.cephDpl.Spec.External {
		// Ensure rgw external service (for openstack keystone integration)
		c.log.Debug().Msg("ensure rgw external service")
		changed, err = c.ensureExternalService()
		if err != nil {
			c.log.Error().Err(err).Msg("failed to ensure rgw external service")
			errMsg = append(errMsg, errors.Wrap(err, "failed to ensure rgw external service"))
		}
		rgwConfigurationChanged = rgwConfigurationChanged || changed
	}

	// Return error if exists
	if len(errMsg) == 1 {
		return false, errMsg[0]
	} else if len(errMsg) > 1 {
		return false, errors.New("multiple errors during rgw ensure")
	}
	return rgwConfigurationChanged, nil
}

// If someone removed rgw section from spec, need to clean all stuff at once
func (c *cephDeploymentConfig) deleteRgw(actualRgw string, keepSyncStore bool) (bool, error) {
	// List all rgw users and buckets
	users, err := c.api.Rookclientset.CephV1().CephObjectStoreUsers(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to get list of rgw users")
	}
	buckets, err := c.api.Claimclientset.ObjectbucketV1alpha1().ObjectBucketClaims(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to get list of rgw buckets")
	}

	errMsg := 0
	deleteCompleted := true

	// Delete all (or except expected) rgw users if exists
	for _, user := range users.Items {
		if actualRgw != "" && user.Spec.Store == actualRgw {
			continue
		}
		deleteCompleted = false
		if err := c.processRgwUsers(objectDelete, &user); err != nil {
			errMsg++
		}
	}
	// Delete all (or except expected) rgw buckets if exists
	for _, bucket := range buckets.Items {
		if actualRgw != "" && bucket.Spec.StorageClassName == rgwStorageClassName {
			continue
		}
		deleteCompleted = false
		if err := c.processRgwBuckets(objectDelete, &bucket); err != nil {
			errMsg++
		}
	}

	if errMsg > 0 {
		return false, errors.New("failed to remove some rgw user/buckets")
	}
	if !deleteCompleted {
		return false, nil
	}

	// Delete rgw object store if exists
	rgwList, err := c.api.Rookclientset.CephV1().CephObjectStores(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.New("failed to list rgw object stores")
	}
	for _, rgw := range rgwList.Items {
		if actualRgw != "" && (rgw.Name == actualRgw || (keepSyncStore && rgw.Name == rgwSyncDaemonName(actualRgw))) {
			continue
		}

		deleteCompleted = false
		// Delete rgw external svc if exists first
		externalSvcName := buildRGWName(rgw.Name, "external")
		c.log.Info().Msgf("rgw external service '%s/%s' cleanup if present", c.lcmConfig.RookNamespace, externalSvcName)
		err = c.api.Kubeclientset.CoreV1().Services(c.lcmConfig.RookNamespace).Delete(c.context, externalSvcName, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			c.log.Error().Err(err).Msgf("failed to remove rgw external service %s", externalSvcName)
			errMsg++
			continue
		}
		c.log.Info().Msgf("rgw object store %s/%s cleanup", c.lcmConfig.RookNamespace, rgw.Name)
		err := c.api.Rookclientset.CephV1().CephObjectStores(c.lcmConfig.RookNamespace).Delete(c.context, rgw.Name, metav1.DeleteOptions{})
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to remove ceph object store %s", rgw.Name)
			errMsg++
		}
		delete(resourceUpdateTimestamps.cephConfigMap, rgwConfigSectionName(rgw.Name))
	}

	// Delete rgw storage class if exists only when complete remove
	if actualRgw == "" && deleteCompleted {
		err = c.api.Kubeclientset.StorageV1().StorageClasses().Delete(c.context, rgwStorageClassName, metav1.DeleteOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				c.log.Error().Err(err).Msgf("failed to delete rgw storage class %s", rgwStorageClassName)
				errMsg++
			}
		} else {
			c.log.Info().Msgf("removing rgw storage class %s", rgwStorageClassName)
			deleteCompleted = false
		}
		resourceUpdateTimestamps.rgwRuntimeParams = ""
	}
	if errMsg > 0 {
		return false, errors.New("failed to cleanup rgw object store resources")
	}
	return deleteCompleted, nil
}

func (c *cephDeploymentConfig) ensureRgwObject() (bool, error) {
	cephDplRGW := c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw
	namespace := c.lcmConfig.RookNamespace
	rgwStores := []*cephv1.CephObjectStore{}
	if c.cdConfig.cephDpl.Spec.External {
		// secret is required for RGW external work
		_, err := c.api.Kubeclientset.CoreV1().Secrets(namespace).Get(c.context, rgwAdminUserSecretName, metav1.GetOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to get rgw admin user secret %s/%s", namespace, rgwAdminUserSecretName)
		}
		rgwExtResource, err := generateRgwExternal(cephDplRGW, namespace)
		if err != nil {
			return false, errors.Wrap(err, "failed to generate external rgw")
		}
		rgwStores = append(rgwStores, rgwExtResource)
	} else {
		zoneDefined := false
		if cephDplRGW.Zone != nil && cephDplRGW.Zone.Name != "" {
			if c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite != nil {
				for _, zone := range c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite.Zones {
					if zone.Name == cephDplRGW.Zone.Name {
						zoneDefined = true
						break
					}
				}
			}
			if !zoneDefined {
				return false, errors.Errorf("failed to generate rgw with unknown %s zone", cephDplRGW.Zone.Name)
			}
		}
		isDefaultRealm := c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite == nil
		useDedicatedNodes := false
		for _, node := range c.cdConfig.cephDpl.Spec.Nodes {
			if lcmcommon.Contains(node.Roles, "rgw") {
				useDedicatedNodes = true
				break
			}
		}
		rgwMainResource := generateRgw(cephDplRGW, namespace, useDedicatedNodes, false, isDefaultRealm, c.cdConfig.cephDpl.Spec.HyperConverge)
		rgwStores = append(rgwStores, rgwMainResource)
		// if multisite requires split traffic for sync - create separate daemon
		if c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite != nil && cephDplRGW.Gateway.SplitDaemonForMultisiteTrafficSync {
			rgwSyncResource := generateRgw(cephDplRGW, namespace, useDedicatedNodes, true, isDefaultRealm, c.cdConfig.cephDpl.Spec.HyperConverge)
			rgwStores = append(rgwStores, rgwSyncResource)
		}
	}
	changed := false
	for idx := range rgwStores {
		rgwResource := rgwStores[idx]
		rgw, err := c.api.Rookclientset.CephV1().CephObjectStores(namespace).Get(c.context, rgwResource.Name, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return false, errors.Wrap(err, "failed to get rgw")
			}
			c.log.Info().Msgf("create rgw object store %s/%s", namespace, rgwResource.Name)
			_, err := c.api.Rookclientset.CephV1().CephObjectStores(namespace).Create(c.context, rgwResource, metav1.CreateOptions{})
			if err != nil {
				return false, errors.Wrap(err, "failed to create rgw")
			}
			changed = true
		} else {
			if !reflect.DeepEqual(rgw.Spec, rgwResource.Spec) {
				// when rgw.Spec.Zone.Name is empty and going to be changed, probably that
				// is switching to multisite configuration
				if rgw.Spec.Zone.Name != rgwResource.Spec.Zone.Name && rgw.Spec.Zone.Name != "" {
					return false, errors.New("failed to update rgw, zone change is not supported")
				}
				lcmcommon.ShowObjectDiff(*c.log, rgw.Spec, rgwResource.Spec)
				rgw.Spec = rgwResource.Spec
				c.log.Info().Msgf("update rgw object store %s/%s", namespace, rgwResource.Name)
				_, err := c.api.Rookclientset.CephV1().CephObjectStores(namespace).Update(c.context, rgw, metav1.UpdateOptions{})
				if err != nil {
					return false, errors.Wrap(err, "failed to update rgw")
				}
				changed = true
			}
		}
	}
	return changed, nil
}

func (c *cephDeploymentConfig) ensureDefaultZoneGroupHostnames() (bool, error) {
	if c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.SkipAutoZoneGroupHostnameUpdate {
		c.log.Debug().Msg("skipping zonegroup hostname auto configuration, since marked to skip in spec")
		return false, nil
	}
	cmd := fmt.Sprintf("radosgw-admin zonegroup get --rgw-zonegroup=%s --format json", c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name)
	var zonegroupInfo lcmcommon.ZoneGroupInfo
	err := lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd, &zonegroupInfo)
	if err != nil {
		c.log.Error().Err(err).Msg("failed to get zonegroups info")
		return false, errors.Wrapf(err, "failed to get zonegroups info for cluster '%s/%s'", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
	}
	domain := ""
	publicHostnameToUse := c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name
	ingressTLS := getIngressTLS(c.cdConfig.cephDpl)
	if ingressTLS != nil {
		domain = ingressTLS.Domain
		if ingressTLS.Hostname != "" {
			publicHostnameToUse = ingressTLS.Hostname
		}
	} else {
		if lcmcommon.IsOpenStackPoolsPresent(c.cdConfig.cephDpl.Spec.Pools) && c.lcmConfig.DeployParams.OpenstackCephSharedNamespace != "" {
			openstackSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.DeployParams.OpenstackCephSharedNamespace).Get(c.context, openstackRgwCredsName, metav1.GetOptions{})
			if err != nil {
				if !apierrors.IsNotFound(err) {
					err = errors.Wrapf(err, "failed to get rgw creds secret %s", openstackRgwCredsName)
					c.log.Error().Err(err).Msg("")
					return false, err
				}
			} else {
				domain = string(openstackSecret.Data["public_domain"])
			}
		}
	}

	rgwDNSNameFromConfig, presentInConfig := "", false
	if c.cdConfig.cephDpl.Spec.RookConfig != nil {
		rgwDNSNameFromConfig, presentInConfig = c.cdConfig.cephDpl.Spec.RookConfig["rgw_dns_name"]
	}
	// our workaround to automatically set required hostnames for zonegroup, since Rook
	// doesn't have such ability
	cmd = fmt.Sprintf("/usr/local/bin/zonegroup_hostnames_update.sh --rgw-zonegroup %s", c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name)
	if domain == "" && !presentInConfig {
		if len(zonegroupInfo.Hostnames) == 0 {
			return false, nil
		}
		// unset hostnames if any, since it is default behaviour for rook and default zonegroup - no any hostnames
		c.log.Info().Msgf("unsetting hostnames for zonegroup %s in cluster '%s/%s': no public dns name provided", c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
		cmd = fmt.Sprintf("%s --unset", cmd)
	} else {
		publicName := fmt.Sprintf("%s.%s", publicHostnameToUse, domain)
		if presentInConfig {
			publicName = rgwDNSNameFromConfig
		}
		internalName := fmt.Sprintf("%s.%s.svc", buildRGWName(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, ""), c.lcmConfig.RookNamespace)
		newNames := []string{publicName, internalName}
		sort.Strings(zonegroupInfo.Hostnames)
		sort.Strings(newNames)
		if reflect.DeepEqual(zonegroupInfo.Hostnames, newNames) {
			return false, nil
		}
		c.log.Info().Msgf("updating hostnames for zonegroup %s in cluster '%s/%s' (old: %v, new %v)",
			c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name, zonegroupInfo.Hostnames, newNames)
		cmd = fmt.Sprintf("%s --hostnames %s", cmd, strings.Join(newNames, ","))
	}

	c.log.Info().Msgf("updating hostnames for zonegroup %s for cluster '%s/%s'", c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
	_, err = lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, cmd)
	if err != nil {
		c.log.Error().Err(err).Msgf("failed to update zonegroup %s hostnames for cluster '%s/%s'",
			c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
		return false, errors.Wrapf(err, "failed to update zonegroup '%s' hostnames for cluster '%s/%s'", c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
	}
	return true, nil
}

func (c *cephDeploymentConfig) ensureRgwStorageClass() (bool, error) {
	_, err := c.api.Kubeclientset.StorageV1().StorageClasses().Get(c.context, rgwStorageClassName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
		c.log.Info().Msgf("create rgw storage class %s", rgwStorageClassName)
		scResource := generateRgwStorageClass(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, rgwStorageClassName, c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name)
		_, err := c.api.Kubeclientset.StorageV1().StorageClasses().Create(c.context, &scResource, metav1.CreateOptions{})
		if err != nil {
			return false, errors.Wrap(err, "failed to create rgw storage class")
		}
		return true, nil
	}
	return false, nil
}

func (c *cephDeploymentConfig) ensureRgwUsers() (bool, error) {
	usersList, err := c.api.Rookclientset.CephV1().CephObjectStoreUsers(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list rgw users")
	}
	presentUsers := map[string]*cephv1.CephObjectStoreUser{}
	for _, user := range usersList.Items {
		presentUsers[user.Name] = user.DeepCopy()
	}
	errMsg := make([]error, 0)
	changed := false
	for _, cephDplRGWUser := range c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.ObjectUsers {
		newUser := generateRgwUser(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, cephDplRGWUser, c.lcmConfig.RookNamespace)
		if presentUser, ok := presentUsers[newUser.Name]; ok {
			if presentUser.Status == nil || presentUser.Status.Phase == rookUtils.ProcessingStatus {
				err := fmt.Sprintf("found not ready CephObjectStoreUser %s/%s, waiting for readiness", c.lcmConfig.RookNamespace, presentUser.Name)
				if presentUser.Status != nil {
					err = fmt.Sprintf("%s (current phase is '%v')", err, presentUser.Status.Phase)
				}
				c.log.Error().Msg(err)
				errMsg = append(errMsg, errors.New(err))
			} else {
				if !reflect.DeepEqual(presentUser.Spec, newUser.Spec) {
					lcmcommon.ShowObjectDiff(*c.log, presentUser.Spec, newUser.Spec)
					presentUser.Spec = newUser.Spec
					if err := c.processRgwUsers(objectUpdate, presentUser); err != nil {
						errMsg = append(errMsg, err)
					}
					changed = true
				}
			}
			delete(presentUsers, newUser.Name)
		} else {
			if err := c.processRgwUsers(objectCreate, &newUser); err != nil {
				errMsg = append(errMsg, err)
			}
			changed = true
		}
	}

	for _, user := range presentUsers {
		if err := c.processRgwUsers(objectDelete, user); err != nil {
			errMsg = append(errMsg, err)
		}
		changed = true
	}

	// Return error if exists
	if len(errMsg) == 1 {
		return false, errors.Wrap(errMsg[0], "failed to ensure CephObjectStoreUsers")
	} else if len(errMsg) > 1 {
		return false, errors.New("failed to ensure CephObjectStoreUsers, multiple errors during users ensure")
	}
	return changed, nil
}

func (c *cephDeploymentConfig) processRgwUsers(process objectProcess, rgwUser *cephv1.CephObjectStoreUser) error {
	var err error
	switch process {
	case objectCreate:
		c.log.Info().Msgf("creating CephObjectStoreUser %s/%s", rgwUser.Namespace, rgwUser.Name)
		_, err = c.api.Rookclientset.CephV1().CephObjectStoreUsers(rgwUser.Namespace).Create(c.context, rgwUser, metav1.CreateOptions{})
	case objectUpdate:
		c.log.Info().Msgf("updating CephObjectStoreUser %s/%s", rgwUser.Namespace, rgwUser.Name)
		_, err = c.api.Rookclientset.CephV1().CephObjectStoreUsers(rgwUser.Namespace).Update(c.context, rgwUser, metav1.UpdateOptions{})
	case objectDelete:
		c.log.Info().Msgf("removing CephObjectStoreUser %s/%s", rgwUser.Namespace, rgwUser.Name)
		err = c.api.Rookclientset.CephV1().CephObjectStoreUsers(rgwUser.Namespace).Delete(c.context, rgwUser.Name, metav1.DeleteOptions{})
	}
	if err != nil {
		if process == objectDelete && apierrors.IsNotFound(err) {
			return nil
		}
		err = errors.Wrapf(err, "failed to %v CephObjectStoreUser %s/%s", process, rgwUser.Namespace, rgwUser.Name)
		c.log.Error().Err(err).Msg("")
		return err
	}
	return nil
}

func (c *cephDeploymentConfig) processRgwBuckets(process objectProcess, rgwBucket *bktv1alpha1.ObjectBucketClaim) error {
	var err error
	switch process {
	case objectCreate:
		c.log.Info().Msgf("creating rgw bucket %s/%s", rgwBucket.Namespace, rgwBucket.Name)
		_, err = c.api.Claimclientset.ObjectbucketV1alpha1().ObjectBucketClaims(rgwBucket.Namespace).Create(c.context, rgwBucket, metav1.CreateOptions{})
	case objectDelete:
		c.log.Info().Msgf("removing rgw bucket %s/%s", rgwBucket.Namespace, rgwBucket.Name)
		err = c.api.Claimclientset.ObjectbucketV1alpha1().ObjectBucketClaims(rgwBucket.Namespace).Delete(c.context, rgwBucket.Name, metav1.DeleteOptions{})
	}
	if err != nil {
		if process == objectDelete && apierrors.IsNotFound(err) {
			return nil
		}
		err = errors.Wrapf(err, "failed to %v rgw bucket %s/%s", process, rgwBucket.Namespace, rgwBucket.Name)
		c.log.Error().Err(err).Msg("")
		return err
	}
	return nil
}

func (c *cephDeploymentConfig) ensureRgwBuckets() (bool, error) {
	bucketsList, err := c.api.Claimclientset.ObjectbucketV1alpha1().ObjectBucketClaims(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list rgw buckets")
	}
	presentBuckets := map[string]*bktv1alpha1.ObjectBucketClaim{}
	for _, bucket := range bucketsList.Items {
		presentBuckets[bucket.Name] = bucket.DeepCopy()
	}
	errMsg := make([]error, 0)
	changed := false
	for _, cephDplRGWBucketName := range c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Buckets {
		newBucket := generateRgwBucket(rgwStorageClassName, cephDplRGWBucketName, c.lcmConfig.RookNamespace)
		if presentBucket, ok := presentBuckets[newBucket.Name]; ok {
			if presentBucket.Status.Phase == bktv1alpha1.ObjectBucketClaimStatusPhasePending {
				err := fmt.Sprintf("found not ready bucket %s/%s, waiting for readiness (current phase is %v)",
					c.lcmConfig.RookNamespace, presentBucket.Name, presentBucket.Status.Phase)
				c.log.Error().Err(errors.New(err)).Msg("")
				errMsg = append(errMsg, errors.New(err))
			}
			delete(presentBuckets, newBucket.Name)
		} else {
			if err := c.processRgwBuckets(objectCreate, &newBucket); err != nil {
				errMsg = append(errMsg, err)
			}
			changed = true
		}
	}

	for _, bucket := range presentBuckets {
		if err := c.processRgwBuckets(objectDelete, bucket); err != nil {
			errMsg = append(errMsg, err)
		}
		changed = true
	}

	// Return error if exists
	if len(errMsg) == 1 {
		return false, errors.Wrap(errMsg[0], "failed to ensure rgw buckets")
	} else if len(errMsg) > 1 {
		return false, errors.New("failed to ensure rgw buckets, multiple errors during buckets ensure")
	}
	return changed, nil
}

func (c *cephDeploymentConfig) ensureExternalService() (bool, error) {
	externalSvcName := buildRGWName(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, "external")
	if isSpecIngressProxyRequired(c.cdConfig.cephDpl.Spec) {
		err := c.api.Kubeclientset.CoreV1().Services(c.lcmConfig.RookNamespace).Delete(c.context, externalSvcName, metav1.DeleteOptions{})
		if err == nil {
			c.log.Info().Msgf("cleanup rgw external service %s in favor of using ingress proxy", externalSvcName)
			return true, nil
		} else if !apierrors.IsNotFound(err) {
			return false, errors.Wrapf(err, "failed to cleanup rgw external service %s", externalSvcName)
		}
		return false, nil
	}
	externalAccessLabel, err := metav1.ParseToLabelSelector(c.lcmConfig.DeployParams.RgwPublicAccessLabel)
	if err != nil {
		return false, errors.Wrapf(err, "failed to parse provided rgw public access label '%s'", c.lcmConfig.DeployParams.RgwPublicAccessLabel)
	}
	externalSvc, err := c.api.Kubeclientset.CoreV1().Services(c.lcmConfig.RookNamespace).Get(c.context, externalSvcName, metav1.GetOptions{})
	externalSvcResource := generateRgwExternalService(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, c.lcmConfig.RookNamespace, externalAccessLabel, c.cdConfig.cephDpl)
	if err != nil {
		if apierrors.IsNotFound(err) {
			c.log.Info().Msgf("create rgw external service %s", externalSvcName)
			_, err := c.api.Kubeclientset.CoreV1().Services(c.lcmConfig.RookNamespace).Create(c.context, &externalSvcResource, metav1.CreateOptions{})
			if err != nil {
				return false, errors.Wrap(err, "failed to create rgw external service")
			}
			return true, nil
		}
		return false, errors.Wrap(err, "failed to get rgw external service")
	}
	updateRequired := false
	externalSvcCur := externalSvc.DeepCopy()
NewPortLoop:
	for _, newPort := range externalSvcResource.Spec.Ports {
		for ix, curPort := range externalSvc.Spec.Ports {
			// we have static Port fields - 80/443
			if newPort.Port == curPort.Port {
				if newPort.Name != curPort.Name || newPort.TargetPort != curPort.TargetPort {
					externalSvc.Spec.Ports[ix].Name = newPort.Name
					externalSvc.Spec.Ports[ix].TargetPort = newPort.TargetPort
					updateRequired = true
				}
				continue NewPortLoop
			}
		}
		externalSvc.Spec.Ports = append(externalSvc.Spec.Ports, newPort)
		updateRequired = true
	}
	updateRequired = updateRequired || !reflect.DeepEqual(externalSvc.Labels, externalSvcResource.Labels)
	if updateRequired {
		externalSvc.Labels = externalSvcResource.Labels
		lcmcommon.ShowObjectDiff(*c.log, externalSvcCur, externalSvc)
		c.log.Info().Msgf("update rgw external service %s", externalSvcName)
		_, err := c.api.Kubeclientset.CoreV1().Services(c.lcmConfig.RookNamespace).Update(c.context, externalSvc, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrap(err, "failed to update rgw external service")
		}
	}
	return updateRequired, nil
}

func (c *cephDeploymentConfig) statusRgw() (bool, error) {
	rgw, err := c.api.Rookclientset.CephV1().CephObjectStores(c.lcmConfig.RookNamespace).Get(c.context, c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, metav1.GetOptions{})
	if err != nil {
		c.log.Error().Err(err).Msgf("failed to get rgw object store %s for status ensure: %v", c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, err)
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to get object store")
	}
	if rgw.Status != nil && !isTypeReadyToUpdate(rgw.Status.Phase) {
		return true, errors.Errorf("rgw is not ready to be updated, current phase is %v", rgw.Status.Phase)
	}
	return true, nil
}

func rgwConfigSectionName(rgwName string) string {
	return fmt.Sprintf("client.rgw.%s.a", strings.ReplaceAll(rgwName, "-", "."))
}

func (c *cephDeploymentConfig) ensureRgwConsistence() error {
	rgwList, err := c.api.Rookclientset.CephV1().CephObjectStores(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("failed to list rgw object store")
		return errors.Wrap(err, "failed to list rgw object store")
	}

	syncStorePresent := c.cdConfig.cephDpl.Spec.ObjectStorage.MultiSite != nil && c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Gateway.SplitDaemonForMultisiteTrafficSync

	cleanupRedundant := false
	for _, rgw := range rgwList.Items {
		if rgw.Name == c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name {
			continue
		}
		if syncStorePresent && rgw.Name == rgwSyncDaemonName(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name) {
			continue
		}
		cleanupRedundant = true
		c.log.Warn().Msgf("found odd CephObjectStore '%s/%s', will be removed", rgw.Namespace, rgw.Name)
	}

	if cleanupRedundant {
		_, err := c.deleteRgw(c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.Name, syncStorePresent)
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return errors.Wrap(err, "failed to cleanup inconsistent rgw resources")
		}
	}
	return nil
}

func generateRgwExternalService(name, namespace string, externalAccessLabel *metav1.LabelSelector, cephDpl *cephlcmv1alpha1.CephDeployment) v1.Service {
	svc := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildRGWName(name, "external"),
			Namespace: namespace,
			Labels: map[string]string{
				"app":               "rook-ceph-rgw",
				"rook_object_store": name,
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					Protocol: "TCP",
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: cephDpl.Spec.ObjectStorage.Rgw.Gateway.Port,
					},
				},
				{
					Name:     "https",
					Port:     443,
					Protocol: "TCP",
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: cephDpl.Spec.ObjectStorage.Rgw.Gateway.SecurePort,
					},
				},
			},
			Type:            "LoadBalancer",
			SessionAffinity: "None",
			Selector: map[string]string{
				"app":               "rook-ceph-rgw",
				"rook_cluster":      namespace,
				"rook_object_store": name,
			},
		},
	}
	for key, val := range externalAccessLabel.MatchLabels {
		svc.Labels[key] = val
	}
	return svc
}

func generateRgwBucket(storageClassName, name, namespace string) bktv1alpha1.ObjectBucketClaim {
	return bktv1alpha1.ObjectBucketClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: bktv1alpha1.ObjectBucketClaimSpec{
			GenerateBucketName: name,
			StorageClassName:   storageClassName,
		},
	}
}

func generateRgwStorageClass(storeName, name, namespace, region string) v1storage.StorageClass {
	return v1storage.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Provisioner: "rook-ceph.ceph.rook.io/bucket",
		Parameters: map[string]string{
			"objectStoreName":      storeName,
			"objectStoreNamespace": namespace,
			"region":               region,
		},
	}
}

func generateRgwUser(storename string, user cephlcmv1alpha1.CephRGWUser, namespace string) cephv1.CephObjectStoreUser {
	displayName := user.DisplayName
	if displayName == "" {
		displayName = user.Name
	}
	return cephv1.CephObjectStoreUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      user.Name,
			Namespace: namespace,
		},
		Spec: cephv1.ObjectStoreUserSpec{
			Store:        storename,
			DisplayName:  displayName,
			Capabilities: user.Capabilities,
			Quotas:       user.Quotas,
		},
	}
}

func generateRgw(cephDplRGW cephlcmv1alpha1.CephRGW, namespace string, useDedicatedNodes, syncRgwDaemon, defaultRealm bool, hyperconverge *cephlcmv1alpha1.CephDeploymentHyperConverge) *cephv1.CephObjectStore {
	label := lcmcommon.CephNodeLabels["mon"]
	if useDedicatedNodes {
		label = lcmcommon.CephNodeLabels["rgw"]
	}
	storeName := cephDplRGW.Name
	rgwSectionName := rgwConfigSectionName(cephDplRGW.Name)
	gatewaySpec := cephv1.GatewaySpec{
		// control rgw ssl cert, ceph config changes - need to restart rgw pods as well
		Annotations: map[string]string{
			sslCertGenerationTimestampLabel:                                 resourceUpdateTimestamps.rgwSSLCert,
			fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, "global"): resourceUpdateTimestamps.cephConfigMap["global"],
		},
		Instances:   cephDplRGW.Gateway.Instances,
		Port:        cephDplRGW.Gateway.Port,
		SecurePort:  cephDplRGW.Gateway.SecurePort,
		HostNetwork: cephDplRGW.RgwUseHostNetwork,
		// ssl cert is used by "cert" key
		SSLCertificateRef: rgwSslCertSecretName,
		// ca bundle is used by "cabundle" key
		CaBundleRef: rgwSslCertSecretName,
		Placement: cephv1.Placement{
			NodeAffinity: &v1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
					NodeSelectorTerms: []v1.NodeSelectorTerm{
						{
							MatchExpressions: []v1.NodeSelectorRequirement{
								{
									Key:      label,
									Operator: "In",
									Values: []string{
										"true",
									},
								},
							},
						},
					},
				},
			},
			Tolerations: []v1.Toleration{
				{
					Key:      label,
					Operator: "Exists",
				},
			},
		},
		DisableMultisiteSyncTraffic: cephDplRGW.Gateway.SplitDaemonForMultisiteTrafficSync,
	}
	if syncRgwDaemon {
		storeName = rgwSyncDaemonName(cephDplRGW.Name)
		rgwSectionName = rgwConfigSectionName(storeName)
		gatewaySpec.DisableMultisiteSyncTraffic = false
		gatewaySpec.Instances = 1
		// to be sure that port for sync daemon is not listening already
		if cephDplRGW.Gateway.RgwSyncPort != 0 {
			gatewaySpec.Port = cephDplRGW.Gateway.RgwSyncPort
		} else {
			gatewaySpec.Port = 8380
		}
		gatewaySpec.SecurePort = 0
		gatewaySpec.SSLCertificateRef = ""
	}
	gatewaySpec.Annotations[fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, rgwSectionName)] = resourceUpdateTimestamps.cephConfigMap[rgwSectionName]
	if hyperconverge != nil {
		if _, ok := hyperconverge.Tolerations["rgw"]; ok {
			gatewaySpec.Placement.Tolerations = append(gatewaySpec.Placement.Tolerations, hyperconverge.Tolerations["rgw"].Rules...)
		}
		if res, ok := hyperconverge.Resources["rgw"]; ok {
			gatewaySpec.Resources = res
		}
	}
	if cephDplRGW.Gateway.Resources != nil {
		gatewaySpec.Resources = *cephDplRGW.Gateway.Resources
	}

	cephObjectStore := &cephv1.CephObjectStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storeName,
			Namespace: namespace,
		},
		Spec: cephv1.ObjectStoreSpec{
			PreservePoolsOnDelete: cephDplRGW.PreservePoolsOnDelete,
			Gateway:               gatewaySpec,
			DefaultRealm:          defaultRealm,
		},
	}
	if cephDplRGW.Zone != nil && cephDplRGW.Zone.Name != "" {
		cephObjectStore.Spec.Zone = *cephDplRGW.Zone
	} else {
		cephObjectStore.Spec.MetadataPool = *generatePoolSpec(cephDplRGW.MetadataPool, "rgw metadata")
		cephObjectStore.Spec.DataPool = *generatePoolSpec(cephDplRGW.DataPool, "rgw data")
	}
	if cephDplRGW.HealthCheck != nil {
		cephObjectStore.Spec.HealthCheck = *cephDplRGW.HealthCheck
	}
	return cephObjectStore
}

func generateRgwExternal(cephDplRGW cephlcmv1alpha1.CephRGW, namespace string) (*cephv1.CephObjectStore, error) {
	if cephDplRGW.Gateway.ExternalRgwEndpoint != nil {
		objectStoreExternal := &cephv1.CephObjectStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cephDplRGW.Name,
				Namespace: namespace,
			},
			Spec: cephv1.ObjectStoreSpec{
				Gateway: cephv1.GatewaySpec{
					Port:                 cephDplRGW.Gateway.Port,
					ExternalRgwEndpoints: []cephv1.EndpointAddress{*cephDplRGW.Gateway.ExternalRgwEndpoint},
				},
			},
		}

		objectStoreExternal.Spec.Gateway.SecurePort = cephDplRGW.Gateway.SecurePort
		objectStoreExternal.Spec.Gateway.SSLCertificateRef = rgwSslCertSecretName
		objectStoreExternal.Spec.Gateway.CaBundleRef = rgwSslCertSecretName
		objectStoreExternal.Spec.Gateway.Annotations = map[string]string{sslCertGenerationTimestampLabel: resourceUpdateTimestamps.rgwSSLCert}
		// Return external rgw object store
		return objectStoreExternal, nil
	}
	return nil, errors.New("external RGW endpoint is not specified for external ceph cluster")
}

func (c *cephDeploymentConfig) ensureRgwInternalSslCert() (bool, error) {
	publicCacert := ""
	if !c.cdConfig.cephDpl.Spec.External && c.cdConfig.cephDpl.Spec.IngressConfig != nil {
		tlsConfig := getIngressTLS(c.cdConfig.cephDpl)
		if tlsConfig != nil {
			if tlsConfig.TLSCerts != nil {
				publicCacert = tlsConfig.TLSCerts.Cacert
			} else if tlsConfig.TLSSecretRefName != "" {
				ingressSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, tlsConfig.TLSSecretRefName, metav1.GetOptions{})
				if err != nil {
					return false, errors.Wrapf(err, "failed to get ingress secret %s/%s", c.lcmConfig.RookNamespace, tlsConfig.TLSSecretRefName)
				}
				publicCacert = string(ingressSecret.Data["ca.crt"])
			}
		}
	}
	if publicCacert == "" && lcmcommon.IsOpenStackPoolsPresent(c.cdConfig.cephDpl.Spec.Pools) && c.lcmConfig.DeployParams.OpenstackCephSharedNamespace != "" {
		openstackSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.DeployParams.OpenstackCephSharedNamespace).Get(c.context, openstackRgwCredsName, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return false, errors.Wrapf(err, "failed to get rgw creds secret %s/%s", c.lcmConfig.DeployParams.OpenstackCephSharedNamespace, openstackRgwCredsName)
			}
		} else {
			publicCacert = string(openstackSecret.Data["ca_cert"])
		}
	}

	var cert string
	var cacert string
	rgwSslCert, rgwCertErr := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, rgwSslCertSecretName, metav1.GetOptions{})
	if rgwCertErr != nil && !apierrors.IsNotFound(rgwCertErr) {
		return false, errors.Wrapf(rgwCertErr, "failed to get secret %s/%s", c.lcmConfig.RookNamespace, rgwSslCertSecretName)
	}

	if c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.SSLCert != nil {
		customSSLCert := c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.SSLCert
		cert = customSSLCert.TLSKey + "\n" + customSSLCert.TLSCert + "\n" + customSSLCert.Cacert
		cacert = customSSLCert.Cacert
		err := lcmcommon.VerifyCertificateExpireDate([]byte(cacert))
		if err != nil {
			return false, errors.Wrap(err, "ssl verification failed for provided rgw ssl certs in spec")
		}
	} else {
		generateNew := true
		if rgwCertErr == nil && rgwSslCert.Data["cacert"] != nil && rgwSslCert.Data["cert"] != nil {
			err := lcmcommon.VerifyCertificateExpireDate(rgwSslCert.Data["cacert"])
			if err != nil {
				if c.cdConfig.cephDpl.Spec.ObjectStorage.Rgw.SSLCertInRef {
					return false, errors.Wrapf(err, "ssl verification failed for rgw ssl certs provided in '%s' secret, update manually", rgwSslCertSecretName)
				}
				// do not fail with cert expired error if rgw-ssl-certificate was
				// generated by us to make it renew
				c.log.Error().Err(err).Msg("rgw ssl certs verification failed")
			} else {
				cacert = string(rgwSslCert.Data["cacert"])
				cert = string(rgwSslCert.Data["cert"])
				generateNew = false
			}
		}
		if generateNew {
			c.log.Info().Msg("generating new Rgw SSL certificate (self-signed)")
			certName := fmt.Sprintf("*.%s.svc.cluster.local", c.lcmConfig.RookNamespace)
			dnsNames := []string{fmt.Sprintf("*.%s.svc", c.lcmConfig.RookNamespace), fmt.Sprintf("*.%s.svc.cluster.local", c.lcmConfig.RookNamespace)}
			tlsKey, tlsCert, caCert, err := lcmcommon.GenerateSelfSignedCert("kubernetes-rgw", certName, dnsNames)
			if err != nil {
				return false, errors.Wrap(err, "failed to generate rgw ssl cert")
			}
			cert = tlsKey + tlsCert + caCert
			cacert = caCert
		}
	}

	caBundle := []string{cacert}
	if publicCacert != "" {
		caBundle = append(caBundle, publicCacert)
	}
	// if provided multisite cabundle - we need it
	if c.lcmConfig.DeployParams.MultisiteCabundleSecretRef != "" {
		multisiteCabundleSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, c.lcmConfig.DeployParams.MultisiteCabundleSecretRef, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return false, errors.Wrapf(err, "failed to get multisite cabundle secret %s/%s", c.lcmConfig.RookNamespace, c.lcmConfig.DeployParams.MultisiteCabundleSecretRef)
			}
		} else if bundle, ok := multisiteCabundleSecret.Data["cabundle"]; ok && len(bundle) > 0 {
			caBundle = append(caBundle, string(bundle))
		} else {
			return false, errors.Errorf("multisite cabundle secret %s/%s has no provided 'cabundle' or empty", c.lcmConfig.RookNamespace, c.lcmConfig.DeployParams.MultisiteCabundleSecretRef)
		}
	}
	bundleCert := strings.Join(caBundle, "\n") + "\n"

	// save to global var last time of changing cert later for
	// create and update cases, to do not lose rgw restart when it is needed
	// for create case - cert may be removed during usual reconcilation and
	// then can be recreated (self-signed or update in spec and failed controller)
	newTime := lcmcommon.GetCurrentTimeString()
	newSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rgwSslCertSecretName,
			Namespace: c.lcmConfig.RookNamespace,
			Annotations: map[string]string{
				sslCertGenerationTimestampLabel: newTime,
			},
		},
		Data: map[string][]byte{
			"cert":     []byte(cert),
			"cacert":   []byte(cacert),
			"cabundle": []byte(bundleCert),
		},
	}

	changed := false
	if apierrors.IsNotFound(rgwCertErr) {
		c.log.Info().Msgf("creating Rgw SSL cert %s/%s", c.lcmConfig.RookNamespace, rgwSslCertSecretName)
		_, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Create(c.context, newSecret, metav1.CreateOptions{})
		if err != nil {
			return false, errors.Wrap(err, "failed to create rgw ssl cert secret")
		}
		changed = true
		resourceUpdateTimestamps.rgwSSLCert = newTime
	} else {
		if reflect.DeepEqual(newSecret.Data, rgwSslCert.Data) {
			// case for controller restart - to keep var is aligned
			if rgwSslCert.Annotations != nil {
				resourceUpdateTimestamps.rgwSSLCert = rgwSslCert.Annotations[sslCertGenerationTimestampLabel]
			}
			return false, nil
		}
		rgwSslCert.Data = newSecret.Data
		if rgwSslCert.Annotations == nil {
			rgwSslCert.Annotations = map[string]string{}
		}
		rgwSslCert.Annotations[sslCertGenerationTimestampLabel] = newTime
		c.log.Info().Msgf("updating Rgw SSL cert %s/%s", c.lcmConfig.RookNamespace, rgwSslCertSecretName)
		_, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Update(c.context, rgwSslCert, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrap(err, "failed to update rgw ssl cert secret")
		}
		changed = true
		resourceUpdateTimestamps.rgwSSLCert = newTime
	}
	return changed, nil
}

func (c *cephDeploymentConfig) deleteRgwInternalSslCert() (bool, error) {
	resourceUpdateTimestamps.rgwSSLCert = ""
	err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Delete(c.context, rgwSslCertSecretName, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, errors.Wrapf(err, "failed to delete rgw ssl cert secret %s/%s", c.lcmConfig.RookNamespace, rgwSslCertSecretName)
	}
	c.log.Info().Msgf("removed rgw ssl cert secret %s/%s", c.lcmConfig.RookNamespace, rgwSslCertSecretName)
	return false, nil
}

func (c *cephDeploymentConfig) deleteRgwAdminOpsSecret() (bool, error) {
	err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Delete(c.context, rgwAdminUserSecretName, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, errors.Wrapf(err, "failed to delete rgw admin ops secret %s/%s", c.lcmConfig.RookNamespace, rgwAdminUserSecretName)
	}
	c.log.Info().Msgf("removed rgw admin ops secret %s/%s", c.lcmConfig.RookNamespace, rgwAdminUserSecretName)
	return false, nil
}

func (c *cephDeploymentConfig) deleteRgwBuiltInPool() (bool, error) {
	poolName := getBuiltinPoolName(".rgw.root")
	err := c.api.Rookclientset.CephV1().CephBlockPools(c.lcmConfig.RookNamespace).Delete(c.context, poolName, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, errors.Wrapf(err, "failed to delete builtin rgw pool %s/%s", c.lcmConfig.RookNamespace, poolName)
	}
	c.log.Info().Msgf("removed builtin CephBlockPool %s/%s", c.lcmConfig.RookNamespace, poolName)
	return false, nil
}
