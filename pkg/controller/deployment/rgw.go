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

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	rookUtils "github.com/rook/rook/pkg/operator/k8sutil"
	v1 "k8s.io/api/core/v1"
	v1storage "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentConfig) ensureRgw() (bool, error) {
	c.log.Debug().Msg("ensure object stores")
	// Ensure that we have rgw designed by spec
	consistent, err := c.ensureRgwConsistence()
	if err != nil {
		return false, err
	}
	rgwConfigurationChanged := !consistent

	errMsg := make([]string, 0)
	rockoonRgw := ""
	for idx, rgw := range c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws {
		// Skip rgw reconcile if rgw is not ready
		c.log.Debug().Msgf("check rgw object store '%s' status", rgw.Name)
		err := c.statusRgw(rgw.Name)
		if err != nil {
			msg := fmt.Sprintf("failed to check rgw '%s' state", rgw.Name)
			c.log.Error().Err(err).Msg(msg)
			errMsg = append(errMsg, msg)
			continue
		}

		if !c.cdConfig.clusterSpec.External.Enable {
			c.log.Debug().Msgf("ensure ssl cert for rgw object store '%s'", rgw.Name)
			changed, err := c.ensureRgwSslCert(idx)
			if err != nil {
				msg := fmt.Sprintf("failed to ensure rgw ssl cert for rgw '%s'", rgw.Name)
				c.log.Error().Err(err).Msg(msg)
				errMsg = append(errMsg, msg)
				continue
			}
			rgwConfigurationChanged = rgwConfigurationChanged || changed
		}

		// Ensure rgw object store
		c.log.Debug().Msgf("ensure rgw object store '%s", rgw.Name)
		changed, err := c.ensureRgwObject(idx)
		if err != nil {
			msg := fmt.Sprintf("failed to ensure rgw object store '%s'", rgw.Name)
			c.log.Error().Err(err).Msg(msg)
			errMsg = append(errMsg, msg)
			continue
		}
		rgwConfigurationChanged = rgwConfigurationChanged || changed

		// do not create storage class or external service if not needed
		// if it is some service like multisite sync daemon or multi-instance rgw api
		if !rgw.AuxiliaryService {
			// Create rgw storageClass if necessary
			c.log.Debug().Msgf("ensure rgw '%s' storage class", rgw.Name)
			changed, err = c.ensureRgwStorageClass(rgw.Name)
			if err != nil {
				msg := fmt.Sprintf("failed to ensure rgw '%s' storage class", rgw.Name)
				c.log.Error().Err(err).Msg(msg)
				errMsg = append(errMsg, msg)
			}
			rgwConfigurationChanged = rgwConfigurationChanged || changed

			if !c.cdConfig.clusterSpec.External.Enable {
				// Ensure rgw external service (for default public access)
				c.log.Debug().Msgf("ensure rgw '%s' external service", rgw.Name)
				changed, err = c.ensureExternalService(idx)
				if err != nil {
					msg := fmt.Sprintf("failed to ensure rgw '%s' external service", rgw.Name)
					c.log.Error().Err(err).Msg(msg)
					errMsg = append(errMsg, msg)
				}
				rgwConfigurationChanged = rgwConfigurationChanged || changed
			}
		}

		// if openstack pools are present - create ceilomenter metrics user as well
		if rgw.UsedByRockoon && rockoonRgw == "" && !c.cdConfig.clusterSpec.External.Enable {
			// take first rockoon rgw as default for now
			rockoonRgw = rgw.Name
			userRaw, _ := cephlcmv1alpha1.DecodeStructToRaw(
				cephv1.ObjectStoreUserSpec{
					Store:       rockoonRgw,
					DisplayName: rgwMetricsUser,
					Capabilities: &cephv1.ObjectUserCapSpec{
						User:     "read",
						Bucket:   "read",
						MetaData: "read",
						Usage:    "read",
					},
				},
			)
			serviceUser := cephlcmv1alpha1.CephObjectStoreUser{
				Name: rgwMetricsUser,
				Spec: runtime.RawExtension{Raw: userRaw},
			}
			c.cdConfig.cephDpl.Spec.ObjectStorage.Users = append([]cephlcmv1alpha1.CephObjectStoreUser{serviceUser}, c.cdConfig.cephDpl.Spec.ObjectStorage.Users...)
		}

		// Create/delete rgw users
		c.log.Debug().Msgf("ensure rgw '%s' users", rgw.Name)
		changed, err = c.ensureRgwUsers(rgw.Name)
		if err != nil {
			msg := fmt.Sprintf("failed to ensure rgw '%s' users", rgw.Name)
			c.log.Error().Err(err).Msg(msg)
			errMsg = append(errMsg, msg)
		}
		rgwConfigurationChanged = rgwConfigurationChanged || changed
	}

	// Return error if exists
	if len(errMsg) > 0 {
		return false, errors.Errorf("error(s) during rgw ensure: %s", strings.Join(errMsg, ", "))
	}
	return rgwConfigurationChanged, nil
}

// If someone removed rgw section from spec, need to clean all stuff at once
func (c *cephDeploymentConfig) deleteRgw(rgwToRemove string) (bool, error) {
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
		if rgwToRemove != "" && user.Spec.Store != rgwToRemove {
			continue
		}
		deleteCompleted = false
		if err := c.processRgwUsers(objectDelete, &user); err != nil {
			errMsg++
		}
	}
	// Delete all (or except expected) rgw buckets if exists
	for _, bucket := range buckets.Items {
		if rgwToRemove != "" && bucket.Spec.StorageClassName != getRgwStorageClass(rgwToRemove) {
			continue
		}
		deleteCompleted = false
		c.log.Info().Msgf("removing rgw bucket %s/%s", c.lcmConfig.RookNamespace, bucket.Name)
		err = c.api.Claimclientset.ObjectbucketV1alpha1().ObjectBucketClaims(c.lcmConfig.RookNamespace).Delete(c.context, bucket.Name, metav1.DeleteOptions{})
		if err != nil {
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
		if rgwToRemove != "" && rgw.Name != rgwToRemove {
			continue
		}

		deleteCompleted = false
		canRemoveCurrentRgw := true
		if !c.cdConfig.clusterSpec.External.Enable {
			// Delete rgw external svc if exists first
			externalSvcName := buildRGWName(rgw.Name, "external")
			err = c.api.Kubeclientset.CoreV1().Services(c.lcmConfig.RookNamespace).Delete(c.context, externalSvcName, metav1.DeleteOptions{})
			if err != nil {
				if !apierrors.IsNotFound(err) {
					c.log.Error().Err(err).Msgf("failed to remove rgw external service %s", externalSvcName)
					errMsg++
					continue
				}
			} else {
				c.log.Info().Msgf("rgw external service '%s/%s' removing", c.lcmConfig.RookNamespace, externalSvcName)
				canRemoveCurrentRgw = false
			}
		}
		rgwStorageClass := getRgwStorageClass(rgw.Name)
		err = c.api.Kubeclientset.StorageV1().StorageClasses().Delete(c.context, rgwStorageClass, metav1.DeleteOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				c.log.Error().Err(err).Msgf("failed to delete rgw storage class %s", rgwStorageClass)
				errMsg++
				continue
			}
		} else {
			c.log.Info().Msgf("removing rgw storage class %s", rgwStorageClass)
			canRemoveCurrentRgw = false
		}

		if canRemoveCurrentRgw {
			c.log.Info().Msgf("rgw object store %s/%s cleanup", c.lcmConfig.RookNamespace, rgw.Name)
			err := c.api.Rookclientset.CephV1().CephObjectStores(c.lcmConfig.RookNamespace).Delete(c.context, rgw.Name, metav1.DeleteOptions{})
			if err != nil {
				c.log.Error().Err(err).Msgf("failed to remove ceph object store %s", rgw.Name)
				errMsg++
			}
			delete(resourceUpdateTimestamps.cephConfigMap, rgwConfigSectionName(rgw.Name))
			delete(resourceUpdateTimestamps.rgwSSLCert, rgw.Name)
		}
	}
	if rgwToRemove == "" {
		resourceUpdateTimestamps.rgwRuntimeParams = ""
	}

	if errMsg > 0 {
		return false, errors.New("failed to cleanup rgw object store resources")
	}
	return deleteCompleted, nil
}

func (c *cephDeploymentConfig) ensureRgwObject(rgwIndexInSpec int) (bool, error) {
	namespace := c.lcmConfig.RookNamespace
	var rgwStore *cephv1.CephObjectStore
	rgwName := c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws[rgwIndexInSpec].Name
	castedSpec, _ := c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws[rgwIndexInSpec].GetSpec()
	if c.cdConfig.clusterSpec.External.Enable {
		// secret is required for RGW external work
		_, err := c.api.Kubeclientset.CoreV1().Secrets(namespace).Get(c.context, rgwAdminUserSecretName, metav1.GetOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to get rgw admin user secret %s/%s", namespace, rgwAdminUserSecretName)
		}
		rgwStore, err = generateRgwExternal(castedSpec, rgwName, namespace)
		if err != nil {
			return false, errors.Wrap(err, "failed to generate external rgw")
		}
	} else {
		zoneDefined := false
		if castedSpec.Zone.Name != "" {
			for _, zone := range c.cdConfig.cephDpl.Spec.ObjectStorage.Zones {
				if zone.Name == castedSpec.Zone.Name {
					zoneDefined = true
					break
				}
			}
			if !zoneDefined {
				return false, errors.Errorf("failed to generate rgw with unknown %s zone", castedSpec.Zone.Name)
			}
		}
		useDedicatedNodes := false
		for _, node := range c.cdConfig.cephDpl.Spec.Nodes {
			if lcmcommon.Contains(node.Roles, "rgw") {
				useDedicatedNodes = true
				break
			}
		}
		rgwStore = generateRgw(castedSpec, rgwName, namespace, useDedicatedNodes)
	}
	changed := false
	rgw, err := c.api.Rookclientset.CephV1().CephObjectStores(namespace).Get(c.context, rgwStore.Name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return false, errors.Wrap(err, "failed to get rgw")
		}
		c.log.Info().Msgf("create rgw object store %s/%s", namespace, rgwStore.Name)
		_, err := c.api.Rookclientset.CephV1().CephObjectStores(namespace).Create(c.context, rgwStore, metav1.CreateOptions{})
		if err != nil {
			return false, errors.Wrap(err, "failed to create rgw")
		}
		changed = true
	} else {
		if !reflect.DeepEqual(rgw.Spec, rgwStore.Spec) {
			// when rgw.Spec.Zone.Name is empty and going to be changed, probably that
			// is switching to multisite configuration
			if rgw.Spec.Zone.Name != rgwStore.Spec.Zone.Name && rgw.Spec.Zone.Name != "" {
				return false, errors.New("failed to update rgw, zone change is not supported")
			}
			lcmcommon.ShowObjectDiff(*c.log, rgw.Spec, rgwStore.Spec)
			rgw.Spec = rgwStore.Spec
			c.log.Info().Msgf("update rgw object store %s/%s", namespace, rgwStore.Name)
			_, err := c.api.Rookclientset.CephV1().CephObjectStores(namespace).Update(c.context, rgw, metav1.UpdateOptions{})
			if err != nil {
				return false, errors.Wrap(err, "failed to update rgw")
			}
			changed = true
		}
	}
	return changed, nil
}

func (c *cephDeploymentConfig) ensureRgwStorageClass(rgwName string) (bool, error) {
	scName := getRgwStorageClass(rgwName)
	_, err := c.api.Kubeclientset.StorageV1().StorageClasses().Get(c.context, scName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
		scResource := generateRgwStorageClass(rgwName, scName, c.lcmConfig.RookNamespace)
		c.log.Info().Msgf("create rgw storage class %s", scResource.Name)
		_, err := c.api.Kubeclientset.StorageV1().StorageClasses().Create(c.context, &scResource, metav1.CreateOptions{})
		if err != nil {
			return false, errors.Wrap(err, "failed to create rgw storage class")
		}
		return true, nil
	}
	return false, nil
}

func (c *cephDeploymentConfig) ensureRgwUsers(rgwName string) (bool, error) {
	usersList, err := c.api.Rookclientset.CephV1().CephObjectStoreUsers(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list rgw users")
	}
	presentUsers := map[string]*cephv1.CephObjectStoreUser{}
	for _, user := range usersList.Items {
		if user.Spec.Store == rgwName {
			presentUsers[user.Name] = user.DeepCopy()
		}
	}
	errMsg := make([]error, 0)
	changed := false
	for _, rgwUser := range c.cdConfig.cephDpl.Spec.ObjectStorage.Users {
		userCasted, _ := rgwUser.GetSpec()
		if userCasted.Store != rgwName {
			continue
		}
		if userCasted.DisplayName == "" {
			userCasted.DisplayName = rgwUser.Name
		}
		newUser := cephv1.CephObjectStoreUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      rgwUser.Name,
				Namespace: c.lcmConfig.RookNamespace,
			},
			Spec: userCasted,
		}
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

func (c *cephDeploymentConfig) ensureExternalService(rgwIndexInSpec int) (bool, error) {
	rgwName := c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws[rgwIndexInSpec].Name
	externalSvcName := buildRGWName(rgwName, "external")
	proxyToBeDeployed, skipReason, err := c.canDeployIngressProxy()
	if err != nil {
		return false, errors.Wrap(err, "failed to check ingress proxy presence")
	}
	if proxyToBeDeployed || c.lcmConfig.DeployParams.RgwPublicAccessLabel == "" {
		err := c.api.Kubeclientset.CoreV1().Services(c.lcmConfig.RookNamespace).Delete(c.context, externalSvcName, metav1.DeleteOptions{})
		if err == nil {
			if proxyToBeDeployed {
				c.log.Info().Msgf("cleanup rgw external service %s in favor of using ingress proxy", externalSvcName)
			} else {
				c.log.Info().Msgf("cleanup rgw external service %s since %s", externalSvcName, skipReason)
			}
			return true, nil
		} else if !apierrors.IsNotFound(err) {
			return false, errors.Wrapf(err, "failed to cleanup rgw external service %s", externalSvcName)
		}
		return false, nil
	}
	externalAccessLabel, err := metav1.ParseToLabelSelector(c.lcmConfig.DeployParams.RgwPublicAccessLabel)
	if err != nil {
		// extra fallback case should not happen at all - because label parsed in config controller
		// and used default in case of any problems
		return false, errors.Wrapf(err, "failed to parse provided rgw public access label '%s'", c.lcmConfig.DeployParams.RgwPublicAccessLabel)
	}
	castedSpec, _ := c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws[rgwIndexInSpec].GetSpec()
	externalSvc, err := c.api.Kubeclientset.CoreV1().Services(c.lcmConfig.RookNamespace).Get(c.context, externalSvcName, metav1.GetOptions{})
	externalSvcResource := generateRgwExternalService(rgwName, c.lcmConfig.RookNamespace, externalAccessLabel, castedSpec.Gateway.Port, castedSpec.Gateway.SecurePort)
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

func (c *cephDeploymentConfig) statusRgw(rgwName string) error {
	rgw, err := c.api.Rookclientset.CephV1().CephObjectStores(c.lcmConfig.RookNamespace).Get(c.context, rgwName, metav1.GetOptions{})
	if err != nil {
		c.log.Error().Err(err).Msgf("failed to get rgw object store %s for status ensure: %v", rgwName, err)
		if apierrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "failed to get object store")
	}
	if rgw.Status != nil && !isTypeReadyToUpdate(rgw.Status.Phase) {
		return errors.Errorf("rgw is not ready to be updated, current phase is %v", rgw.Status.Phase)
	}
	return nil
}

func rgwConfigSectionName(rgwName string) string {
	return fmt.Sprintf("client.rgw.%s.a", strings.ReplaceAll(rgwName, "-", "."))
}

func (c *cephDeploymentConfig) ensureRgwConsistence() (bool, error) {
	rgwList, err := c.api.Rookclientset.CephV1().CephObjectStores(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("failed to list rgw object store")
		return false, errors.Wrap(err, "failed to list rgw object store")
	}

	consistent := true
	zonesInUse := map[string]bool{}
	rgwInSpec := map[string]bool{}
	rgwExists := map[string]bool{}
	for _, rgw := range c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws {
		rgwCasted, _ := rgw.GetSpec()
		if rgwCasted.Zone.Name != "" {
			zonesInUse[rgwCasted.Zone.Name] = true
		}
		rgwInSpec[rgw.Name] = true
	}
	for _, rgw := range rgwList.Items {
		rgwExists[rgw.Name] = true
		if rgwInSpec[rgw.Name] {
			continue
		}
		consistent = false
		// check that we have no dependend store, like sync daemon
		if rgw.Spec.Gateway.DisableMultisiteSyncTraffic && zonesInUse[rgw.Spec.Zone.Name] {
			c.log.Info().Msgf("found CephObjectStore '%s/%s' which is not described in spec and has disable sync traffic, but related zone is still in use. Remove other store first or remove 'disableMultisiteSyncTraffic' option.",
				rgw.Namespace, rgw.Name)
			continue
		}
		c.log.Warn().Msgf("found odd CephObjectStore '%s/%s', will be removed", rgw.Namespace, rgw.Name)
		_, err := c.deleteRgw(rgw.Name)
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return false, errors.Wrap(err, "failed to cleanup inconsistent rgw resources")
		}
	}
	removedSsl, err := c.deleteSelfSignedCerts(rgwExists)
	if err != nil {
		return false, errors.Wrap(err, "failed to cleanup odd rgw secrets")
	}
	consistent = consistent && removedSsl
	return consistent, nil
}

func generateRgwExternalService(name, namespace string, externalAccessLabel *metav1.LabelSelector, port, securePort int32) v1.Service {
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
						IntVal: port,
					},
				},
				{
					Name:     "https",
					Port:     443,
					Protocol: "TCP",
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: securePort,
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

func getRgwStorageClass(rgwName string) string {
	return fmt.Sprintf("%s-bucket", rgwName)
}

func generateRgwStorageClass(storeName, storageClassName, namespace string) v1storage.StorageClass {
	return v1storage.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: storageClassName,
		},
		Provisioner: "rook-ceph.ceph.rook.io/bucket",
		Parameters: map[string]string{
			"objectStoreName":      storeName,
			"objectStoreNamespace": namespace,
			"region":               storeName,
		},
	}
}

func generateRgw(cephDplRGW cephv1.ObjectStoreSpec, name, namespace string, useDedicatedNodes bool) *cephv1.CephObjectStore {
	label := lcmcommon.CephNodeLabels["mon"]
	if useDedicatedNodes {
		label = lcmcommon.CephNodeLabels["rgw"]
	}
	storeName := name
	rgwSectionName := rgwConfigSectionName(name)

	if cephDplRGW.Gateway.Annotations == nil {
		cephDplRGW.Gateway.Annotations = map[string]string{}
	}
	// control default rgw ssl cert, ceph config changes - need to restart rgw pods as well
	if t, ok := resourceUpdateTimestamps.rgwSSLCert[name]; ok {
		cephDplRGW.Gateway.Annotations[sslCertGenerationTimestampLabel] = t
	}
	cephDplRGW.Gateway.Annotations[fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, "global")] = resourceUpdateTimestamps.cephConfigMap["global"]
	cephDplRGW.Gateway.Annotations[fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, rgwSectionName)] = resourceUpdateTimestamps.cephConfigMap[rgwSectionName]

	if cephDplRGW.Gateway.SecurePort != 0 {
		if cephDplRGW.Gateway.SSLCertificateRef == "" {
			cephDplRGW.Gateway.SSLCertificateRef = getRgwDefaultSSLCertificateName(name)
		}
		if cephDplRGW.Gateway.CaBundleRef == "" {
			cephDplRGW.Gateway.CaBundleRef = cephDplRGW.Gateway.SSLCertificateRef
		}
	}
	if cephDplRGW.DataPool.Replicated.Size > 0 {
		if cephDplRGW.DataPool.Replicated.TargetSizeRatio == 0 {
			cephDplRGW.DataPool.Replicated.TargetSizeRatio = poolsDefaultTargetSizeRatioByRole("rgw data")
		}
	}

	// overwrite due nodes section
	cephDplRGW.Gateway.Placement.NodeAffinity = &v1.NodeAffinity{
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
	}
	cephDplRGW.Gateway.Placement.Tolerations = append([]v1.Toleration{{Key: label, Operator: "Exists"}}, cephDplRGW.Gateway.Placement.Tolerations...)

	// Rook issue https://github.com/rook/rook/issues/15984
	// Set obviously defaultRealm to avoid creating default zone/zonegroup
	if cephDplRGW.Zone.Name == "" {
		cephDplRGW.DefaultRealm = true
	}

	return &cephv1.CephObjectStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storeName,
			Namespace: namespace,
		},
		Spec: cephDplRGW,
	}
}

func generateRgwExternal(cephDplRGW cephv1.ObjectStoreSpec, name, namespace string) (*cephv1.CephObjectStore, error) {
	if len(cephDplRGW.Gateway.ExternalRgwEndpoints) > 0 {
		objectStoreExternal := &cephv1.CephObjectStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: cephDplRGW,
		}
		return objectStoreExternal, nil
	}
	return nil, errors.New("external RGW endpoints is not specified for external ceph cluster")
}

func getRgwDefaultSSLCertificateName(rgwName string) string {
	return fmt.Sprintf("%s-ssl-cert", rgwName)
}

func (c *cephDeploymentConfig) ensureRgwSslCert(rgwIndexInSpec int) (bool, error) {
	rgwName := c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws[rgwIndexInSpec].Name
	rgwCasted, _ := c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws[rgwIndexInSpec].GetSpec()
	servedByIngress := c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws[rgwIndexInSpec].ServedByIngress
	usedByRockoon := c.cdConfig.cephDpl.Spec.ObjectStorage.Rgws[rgwIndexInSpec].UsedByRockoon

	changed := false
	noCerts := true
	// do not process rgw ssl cert if no secure port specified
	if rgwCasted.Gateway.SecurePort != 0 {
		noCerts = false
		changedCert, err := c.ensureRgwBackendSSLCert(rgwName, rgwCasted.Gateway.SSLCertificateRef)
		if err != nil {
			return false, errors.Wrapf(err, "failed to ensure rgw '%s' ssl certificate", rgwName)
		}
		changed = changedCert
	} else {
		if rgwCasted.Gateway.SSLCertificateRef != "" {
			return false, errors.Errorf("rgw '%s' has provided sslCertificateRef, but no secure port specified", rgwName)
		}
	}
	// if no cabundle ref, no secure port, no ingress, no rockoon setup - cabundle is not needed
	if rgwCasted.Gateway.SecurePort != 0 || servedByIngress || usedByRockoon || rgwCasted.Gateway.CaBundleRef != "" {
		noCerts = false
		changedCa, err := c.ensureRgwCaBundleCert(rgwName, rgwCasted.Gateway.SSLCertificateRef, rgwCasted.Gateway.CaBundleRef, servedByIngress, usedByRockoon)
		if err != nil {
			return false, errors.Wrapf(err, "failed to ensure rgw '%s' cabundle certificate", rgwName)
		}
		changed = changed || changedCa
	}

	if noCerts {
		delete(resourceUpdateTimestamps.rgwSSLCert, rgwName)
	}
	return changed, nil
}

func (c *cephDeploymentConfig) ensureRgwBackendSSLCert(rgwName, certRef string) (bool, error) {
	rgwBackendSSLCertificateSecret := certRef
	if rgwBackendSSLCertificateSecret == "" {
		rgwBackendSSLCertificateSecret = getRgwDefaultSSLCertificateName(rgwName)
		c.log.Debug().Msgf("rgw '%s' has no provided ssl certificate ref, using default '%s/%s'", rgwName, c.lcmConfig.RookNamespace, rgwBackendSSLCertificateSecret)
	} else {
		c.log.Info().Msgf("rgw '%s' has provided sslCertificateRef, operator should take care for certs update and RGW restart", rgwName)
	}

	rgwSslCert, rgwCertErr := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, rgwBackendSSLCertificateSecret, metav1.GetOptions{})
	if rgwCertErr != nil {
		if !apierrors.IsNotFound(rgwCertErr) || certRef != "" {
			return false, errors.Wrapf(rgwCertErr, "failed to get secret %s/%s", c.lcmConfig.RookNamespace, rgwBackendSSLCertificateSecret)
		}
	}

	if rgwCertErr == nil {
		if rgwSslCert.Data["cacert"] != nil && rgwSslCert.Data["cert"] != nil {
			err := lcmcommon.VerifyCertificateExpireDate(rgwSslCert.Data["cacert"])
			if err != nil {
				if certRef != "" {
					return false, errors.Wrapf(err, "ssl verification failed for rgw '%s' ssl certs provided in '%s' secret, update manually",
						rgwName, certRef)
				}
				// do not fail with cert expired error if rgw-ssl-certificate was
				// generated by us to make it renew
				c.log.Error().Err(err).Msg("rgw ssl certs verification failed for self-signed certificate, will be regenerated")
			} else {
				// if secret has our annotations - get them
				// otherwise consider it as created by operator and operator should
				// take care for rgw restart on update
				dropTiming := true
				if len(rgwSslCert.Annotations) > 0 {
					if t, ok := rgwSslCert.Annotations[sslCertGenerationTimestampLabel]; ok {
						resourceUpdateTimestamps.rgwSSLCert[rgwName] = t
						dropTiming = false
					}
				}
				if dropTiming {
					delete(resourceUpdateTimestamps.rgwSSLCert, rgwName)
				}
				return false, nil
			}
		}
		if certRef != "" {
			return false, errors.Errorf("rgw '%s' ssl certs provided in '%s' secret has no required 'cert' and 'cacert' fields",
				rgwName, certRef)
		}
		c.log.Error().Msg("required secret fields 'cert' and 'cacert' are not found, self-signed certificate will be generated")
	}
	// we have expired self-signed or we have not found default self-signed
	c.log.Info().Msg("generating new Rgw SSL certificate (self-signed)")
	certName := fmt.Sprintf("*.%s.svc.cluster.local", c.lcmConfig.RookNamespace)
	dnsNames := []string{fmt.Sprintf("*.%s.svc", c.lcmConfig.RookNamespace), fmt.Sprintf("*.%s.svc.cluster.local", c.lcmConfig.RookNamespace)}
	tlsKey, tlsCert, caCert, err := lcmcommon.GenerateSelfSignedCert("kubernetes-rgw", certName, dnsNames)
	if err != nil {
		return false, errors.Wrap(err, "failed to generate rgw ssl cert")
	}
	// save to global var last time of changing cert later for
	// create and update cases, to do not lose rgw restart when it is needed
	// for create case - cert may be removed during usual reconcilation and
	// then can be recreated (self-signed or update in spec and failed controller)
	newTime := lcmcommon.GetCurrentTimeString()
	changed := false
	if apierrors.IsNotFound(rgwCertErr) {
		newSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      rgwBackendSSLCertificateSecret,
				Namespace: c.lcmConfig.RookNamespace,
				Labels: map[string]string{
					selfSignedCertLabel: rgwName,
				},
				Annotations: map[string]string{
					sslCertGenerationTimestampLabel: newTime,
				},
			},
			Data: map[string][]byte{
				"cert":   []byte(tlsKey + tlsCert + caCert),
				"cacert": []byte(caCert),
			},
		}
		c.log.Info().Msgf("creating self-signed Rgw SSL cert %s/%s", c.lcmConfig.RookNamespace, newSecret.Name)
		_, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Create(c.context, newSecret, metav1.CreateOptions{})
		if err != nil {
			return false, errors.Wrap(err, "failed to create rgw ssl cert secret")
		}
		changed = true
		resourceUpdateTimestamps.rgwSSLCert[rgwName] = newTime
	} else {
		rgwSslCert.Data["cert"] = []byte(tlsKey + tlsCert + caCert)
		rgwSslCert.Data["cacert"] = []byte(caCert)
		if rgwSslCert.Annotations == nil {
			rgwSslCert.Annotations = map[string]string{}
		}
		rgwSslCert.Annotations[sslCertGenerationTimestampLabel] = newTime
		if rgwSslCert.Labels == nil {
			rgwSslCert.Labels = map[string]string{}
		}
		rgwSslCert.Labels[selfSignedCertLabel] = rgwName
		c.log.Info().Msgf("updating self-signed Rgw SSL cert %s/%s", c.lcmConfig.RookNamespace, rgwSslCert.Name)
		_, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Update(c.context, rgwSslCert, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrap(err, "failed to update rgw ssl cert secret")
		}
		changed = true
		resourceUpdateTimestamps.rgwSSLCert[rgwName] = newTime
	}
	return changed, nil
}

func (c *cephDeploymentConfig) ensureRgwCaBundleCert(rgwName, certRef, caBundleRef string, servedByIngress, usedByRockoon bool) (bool, error) {
	if caBundleRef != "" {
		c.log.Info().Msgf("rgw '%s' has provided caBundleRef, operator should take care for cabundle update and RGW restart", rgwName)
		caBundleCert, caCertErr := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, caBundleRef, metav1.GetOptions{})
		if caCertErr != nil {
			return false, errors.Wrapf(caCertErr, "failed to get secret '%s/%s' with cabundle", c.lcmConfig.RookNamespace, caBundleRef)
		}
		if caBundleCert.Data["cabundle"] == nil {
			return false, errors.Errorf("rgw '%s' secret '%s/%s' used for as cabundle has no required field 'cabundle'",
				rgwName, c.lcmConfig.RookNamespace, caBundleRef)
		}
		return false, nil
	}

	publicCacert := ""
	if servedByIngress || usedByRockoon {
		if c.cdConfig.cephDpl.Spec.IngressConfig != nil {
			tlsConfig := getIngressTLS(c.cdConfig.cephDpl)
			if tlsConfig != nil {
				if tlsConfig.TLSCerts != nil {
					publicCacert = tlsConfig.TLSCerts.Cacert
				} else if tlsConfig.TLSSecretRefName != "" {
					ingressSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, tlsConfig.TLSSecretRefName, metav1.GetOptions{})
					if err != nil {
						return false, errors.Wrapf(err, "failed to get ingress secret '%s/%s'", c.lcmConfig.RookNamespace, tlsConfig.TLSSecretRefName)
					}
					publicCacert = string(ingressSecret.Data["ca.crt"])
				}
			}
		} else if usedByRockoon {
			if c.lcmConfig.DeployParams.OpenstackCephSharedNamespace == "" {
				return false, errors.Errorf("rgw '%s' has specified for Rockoon usage, but Pelagia lcmconfig has no set Rockoon-Ceph namespace", rgwName)
			}
			openstackSecret, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.DeployParams.OpenstackCephSharedNamespace).Get(c.context, openstackRgwCredsName, metav1.GetOptions{})
			if err != nil {
				if !apierrors.IsNotFound(err) {
					return false, errors.Wrapf(err, "failed to get rgw creds secret '%s/%s'", c.lcmConfig.DeployParams.OpenstackCephSharedNamespace, openstackRgwCredsName)
				}
				c.log.Warn().Msgf("openstack rgw secret '%s/%s' is not found, probably not created yet, skipping", c.lcmConfig.DeployParams.OpenstackCephSharedNamespace, openstackRgwCredsName)
			} else {
				publicCacert = string(openstackSecret.Data["ca_cert"])
			}
		}
	}

	changed := false
	defaultCaBundleCertName := certRef
	if defaultCaBundleCertName == "" {
		defaultCaBundleCertName = getRgwDefaultSSLCertificateName(rgwName)
	}
	newCertTime := lcmcommon.GetCurrentTimeString()

	rgwSslCert, rgwCertErr := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Get(c.context, defaultCaBundleCertName, metav1.GetOptions{})
	if rgwCertErr != nil {
		if !apierrors.IsNotFound(rgwCertErr) || certRef != "" {
			return false, errors.Wrapf(rgwCertErr, "failed to get secret '%s/%s'", c.lcmConfig.RookNamespace, defaultCaBundleCertName)
		}
		// no cabundle's found - nothing to do
		if publicCacert == "" {
			return false, nil
		}
		newSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      defaultCaBundleCertName,
				Namespace: c.lcmConfig.RookNamespace,
				Annotations: map[string]string{
					sslCertGenerationTimestampLabel: newCertTime,
				},
			},
			Data: map[string][]byte{
				"cabundle": []byte(publicCacert + "\n"),
			},
		}
		c.log.Info().Msgf("creating Rgw cert %s/%s with cabundle", c.lcmConfig.RookNamespace, newSecret.Name)
		_, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Create(c.context, newSecret, metav1.CreateOptions{})
		if err != nil {
			return false, errors.Wrap(err, "failed to create rgw cabundle cert secret")
		}
		changed = true
		resourceUpdateTimestamps.rgwSSLCert[rgwName] = newCertTime
	} else {
		caCerts := []string{}
		if v, ok := rgwSslCert.Data["cacert"]; ok {
			caCerts = append(caCerts, string(v))
		}
		if publicCacert != "" {
			caCerts = append(caCerts, publicCacert)
		}
		// case when base self-signed cert, used in Rgw was corrupted
		// should not happen, just double check
		if len(caCerts) == 0 {
			return false, errors.Errorf("rgw '%s' seems to be must have cabundle, but it is not found", rgwName)
		}
		if rgwSslCert.Annotations == nil {
			rgwSslCert.Annotations = map[string]string{}
		}
		bundleChain := strings.Join(caCerts, "\n") + "\n"
		if v, ok := rgwSslCert.Data["cabundle"]; !ok || string(v) != bundleChain {
			rgwSslCert.Data["cabundle"] = []byte(bundleChain)
			rgwSslCert.Annotations[sslCertGenerationTimestampLabel] = newCertTime
			c.log.Info().Msgf("updating Rgw SSL cert %s/%s with cabundle", c.lcmConfig.RookNamespace, rgwSslCert.Name)
			_, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Update(c.context, rgwSslCert, metav1.UpdateOptions{})
			if err != nil {
				return false, errors.Wrap(err, "failed to update rgw cabundle cert secret")
			}
			changed = true
			resourceUpdateTimestamps.rgwSSLCert[rgwName] = newCertTime
		} else {
			resourceUpdateTimestamps.rgwSSLCert[rgwName] = rgwSslCert.Annotations[sslCertGenerationTimestampLabel]
		}
	}
	c.log.Debug().Msgf("rgw '%s' has no provided cabundle secret ref, using default '%s/%s'", rgwName, c.lcmConfig.RookNamespace, defaultCaBundleCertName)
	return changed, nil
}

func (c *cephDeploymentConfig) deleteSelfSignedCerts(presentRgws map[string]bool) (bool, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: selfSignedCertLabel,
	}
	secrets, err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).List(c.context, listOptions)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get check rgw self-signed certs to remove")
	}
	if len(secrets.Items) == 0 {
		return true, nil
	}
	removed := true
	for _, secret := range secrets.Items {
		if presentRgws != nil && presentRgws[secret.Labels[selfSignedCertLabel]] {
			continue
		}
		removed = false
		err := c.api.Kubeclientset.CoreV1().Secrets(c.lcmConfig.RookNamespace).Delete(c.context, secret.Name, metav1.DeleteOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to delete rgw default self-signed ssl secret %s/%s", c.lcmConfig.RookNamespace, secret.Name)
		}
		c.log.Info().Msgf("removed rgw default self-signed ssl secret %s/%s", c.lcmConfig.RookNamespace, secret.Name)
	}
	return removed, nil
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
