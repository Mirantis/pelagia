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

package health

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	bktv1alpha1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentHealthConfig) rookObjectsVerification() (*lcmv1alpha1.RookCephObjectsStatus, []string) {
	issuesForRookObjects := []string{}
	cephClusterStatus, cephStatusIssues := c.checkCephCluster()
	if len(cephStatusIssues) > 0 {
		issuesForRookObjects = append(issuesForRookObjects, cephStatusIssues...)
	}
	if cephClusterStatus == nil {
		return nil, issuesForRookObjects
	}
	newRookObjectsReport := &lcmv1alpha1.RookCephObjectsStatus{
		CephCluster: cephClusterStatus,
	}
	cephBlockPoolsStatus, cephBlockPoolsIssues := c.checkCephBlockPools()
	if len(cephBlockPoolsIssues) > 0 {
		issuesForRookObjects = append(issuesForRookObjects, cephBlockPoolsIssues...)
	}
	if cephBlockPoolsStatus != nil {
		newRookObjectsReport.BlockStorage = &lcmv1alpha1.BlockStorageStatus{
			CephBlockPools: cephBlockPoolsStatus,
		}
	}

	cephClientsStatus, cephClientsIssues := c.checkCephClients()
	if len(cephClientsIssues) > 0 {
		issuesForRookObjects = append(issuesForRookObjects, cephClientsIssues...)
	}
	newRookObjectsReport.CephClients = cephClientsStatus

	cephObjectStoreStatus, cephObjectStoreIssues := c.checkCephObjectStores()
	if len(cephObjectStoreIssues) > 0 {
		issuesForRookObjects = append(issuesForRookObjects, cephObjectStoreIssues...)
	}
	cephObjectUsersStatus, cephObjectUsersIssues := c.checkCephObjectUsers()
	if len(cephObjectUsersIssues) > 0 {
		issuesForRookObjects = append(issuesForRookObjects, cephObjectUsersIssues...)
	}
	objectBucketClaimsStatus, objectBucketClaimsIssues := c.checkObjectBucketClaims()
	if len(objectBucketClaimsIssues) > 0 {
		issuesForRookObjects = append(issuesForRookObjects, objectBucketClaimsIssues...)
	}
	cephObjectRealmsStatus, cephObjectRealmsIssues := c.checkCephObjectRealms()
	if len(cephObjectRealmsIssues) > 0 {
		issuesForRookObjects = append(issuesForRookObjects, cephObjectRealmsIssues...)
	}
	cephObjectZoneGroupsStatus, cephObjectZoneGroupsIssues := c.checkCephObjectZoneGroups()
	if len(cephObjectZoneGroupsIssues) > 0 {
		issuesForRookObjects = append(issuesForRookObjects, cephObjectZoneGroupsIssues...)
	}
	cephObjectZonesStatus, cephObjectZonesIssues := c.checkCephObjectZones()
	if len(cephObjectZonesIssues) > 0 {
		issuesForRookObjects = append(issuesForRookObjects, cephObjectZonesIssues...)
	}
	if cephObjectStoreStatus != nil || cephObjectUsersStatus != nil || objectBucketClaimsStatus != nil ||
		cephObjectRealmsStatus != nil || cephObjectZoneGroupsStatus != nil || cephObjectZonesStatus != nil {
		newRookObjectsReport.ObjectStorage = &lcmv1alpha1.ObjectStorageStatus{
			CephObjectStores:     cephObjectStoreStatus,
			CephObjectStoreUsers: cephObjectUsersStatus,
			ObjectBucketClaims:   objectBucketClaimsStatus,
			CephObjectRealms:     cephObjectRealmsStatus,
			CephObjectZoneGroups: cephObjectZoneGroupsStatus,
			CephObjectZones:      cephObjectZonesStatus,
		}
	}

	cephFilesystemsStatus, cephFilesystemsIssues := c.checkCephFilesystems()
	if len(cephFilesystemsIssues) > 0 {
		issuesForRookObjects = append(issuesForRookObjects, cephFilesystemsIssues...)
	}
	if cephFilesystemsStatus != nil {
		newRookObjectsReport.SharedFilesystem = &lcmv1alpha1.SharedFilesystemStatus{
			CephFilesystems: cephFilesystemsStatus,
		}
	}

	return newRookObjectsReport, issuesForRookObjects
}

func skipClusterVerification(cephCluster *cephv1.CephCluster) bool {
	majorVersion := strings.Split(cephCluster.Status.CephVersion.Version, ".")[0]
	majorInt, err := strconv.Atoi(majorVersion)
	if err != nil {
		return false
	}
	return majorInt < 16
}

func (c *cephDeploymentHealthConfig) checkCephCluster() (*cephv1.ClusterStatus, []string) {
	cephCluster, err := c.api.Rookclientset.CephV1().CephClusters(c.lcmConfig.RookNamespace).Get(c.context, c.healthConfig.name, metav1.GetOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		if apierrors.IsNotFound(err) {
			return nil, []string{fmt.Sprintf("cephcluster '%s/%s' object is not found", c.lcmConfig.RookNamespace, c.healthConfig.name)}
		}
		return nil, []string{fmt.Sprintf("failed to get cephcluster '%s/%s' object", c.lcmConfig.RookNamespace, c.healthConfig.name)}
	}
	if cephCluster.Status.CephVersion == nil {
		c.log.Warn().Msgf("cephcluster '%s/%s' object has no valid ceph version, skipping any further verification", c.lcmConfig.RookNamespace, c.healthConfig.name)
		return nil, []string{"cephcluster is creating, no valid cephcluster version in status"}
	}
	if skipClusterVerification(cephCluster) {
		msg := fmt.Sprintf("verification is supported since Ceph Pacific versions (v16.2), current is '%s'", cephCluster.Status.CephVersion.Version)
		c.log.Warn().Msgf("%s", msg)
		return nil, []string{msg}
	}
	c.healthConfig.cephCluster = cephCluster
	return &cephCluster.Status, c.checkClusterStatus()
}

func (c *cephDeploymentHealthConfig) checkClusterStatus() []string {
	issues := make([]string, 0)
	if c.healthConfig.cephCluster.Status.Phase != cephv1.ConditionReady && c.healthConfig.cephCluster.Status.Phase != cephv1.ConditionConnected {
		msg := fmt.Sprintf("cephcluster '%s/%s' object state is '%v'", c.healthConfig.cephCluster.Namespace, c.healthConfig.cephCluster.Name, c.healthConfig.cephCluster.Status.Phase)
		issues = append(issues, msg)
	}
	if c.healthConfig.cephCluster.Status.CephStatus == nil {
		issues = append(issues, fmt.Sprintf("cephcluster '%s/%s' object health info is not available", c.healthConfig.cephCluster.Namespace, c.healthConfig.cephCluster.Name))
		return issues
	}
	if c.healthConfig.cephCluster.Status.CephStatus.Health != "HEALTH_OK" {
		for warning, details := range c.healthConfig.cephCluster.Status.CephStatus.Details {
			if lcmcommon.Contains(c.lcmConfig.HealthParams.CephIssuesToIgnore, warning) {
				c.log.Debug().Msgf("detected ceph cluster health issue '%s', which is ignored by cephdeploymenthealth config", warning)
				continue
			}
			issues = append(issues, fmt.Sprintf("%s: %s", warning, details.Message))
		}
	}
	timeProblem := checkStatusIsNotUpdated(c.healthConfig.cephCluster.Status.CephStatus)
	if timeProblem {
		issues = append(issues, fmt.Sprintf("cephcluster '%s/%s' object status is not updated for last 5 minutes", c.healthConfig.cephCluster.Namespace, c.healthConfig.cephCluster.Name))
	}
	return issues
}

func checkStatusIsNotUpdated(cephStatus *cephv1.CephStatus) bool {
	if cephStatus == nil {
		return true
	}
	parsed, _ := time.Parse(time.RFC3339, cephStatus.LastChecked)
	return int(time.Since(parsed).Seconds()) > 300
}

func (c *cephDeploymentHealthConfig) checkCephBlockPools() (map[string]*cephv1.CephBlockPoolStatus, []string) {
	presentPools, err := c.api.Rookclientset.CephV1().CephBlockPools(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, []string{fmt.Sprintf("failed to list cephblockpools in '%s' namespace", c.lcmConfig.RookNamespace)}
	}
	if len(presentPools.Items) == 0 {
		return nil, nil
	}
	poolsStatus := map[string]*cephv1.CephBlockPoolStatus{}
	issues := make([]string, 0)
	for _, pool := range presentPools.Items {
		poolsStatus[pool.Name] = pool.Status
		if pool.Status == nil {
			issues = append(issues, fmt.Sprintf("cephblockpool '%s/%s' status is not available yet", c.lcmConfig.RookNamespace, pool.Name))
		} else if pool.Status.Phase != cephv1.ConditionReady {
			issues = append(issues, fmt.Sprintf("cephblockpool '%s/%s' is not ready", c.lcmConfig.RookNamespace, pool.Name))
		}
	}
	return poolsStatus, issues
}

func (c *cephDeploymentHealthConfig) checkCephClients() (map[string]*cephv1.CephClientStatus, []string) {
	presentClients, err := c.api.Rookclientset.CephV1().CephClients(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, []string{fmt.Sprintf("failed to list cephclients in '%s' namespace", c.lcmConfig.RookNamespace)}
	}
	if len(presentClients.Items) == 0 {
		return nil, nil
	}
	clientsStatus := map[string]*cephv1.CephClientStatus{}
	issues := make([]string, 0)
	for _, client := range presentClients.Items {
		clientsStatus[client.Name] = client.Status
		if client.Status == nil {
			issues = append(issues, fmt.Sprintf("cephclient '%s/%s' status is not available yet", c.lcmConfig.RookNamespace, client.Name))
		} else if client.Status.Phase != cephv1.ConditionReady {
			issues = append(issues, fmt.Sprintf("cephclient '%s/%s' is not ready", c.lcmConfig.RookNamespace, client.Name))
		}
	}
	return clientsStatus, issues
}

func (c *cephDeploymentHealthConfig) checkCephFilesystems() (map[string]*cephv1.CephFilesystemStatus, []string) {
	cephFsList, err := c.api.Rookclientset.CephV1().CephFilesystems(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, []string{fmt.Sprintf("failed to list cephfilesystems in '%s' namespace", c.lcmConfig.RookNamespace)}
	}
	if len(cephFsList.Items) == 0 {
		return nil, nil
	}
	cephFsStatus := map[string]*cephv1.CephFilesystemStatus{}
	issues := make([]string, 0)
	for _, cephFs := range cephFsList.Items {
		cephFsStatus[cephFs.Name] = cephFs.Status
		if cephFs.Status == nil {
			issues = append(issues, fmt.Sprintf("cephfilesystem '%s/%s' status is not available yet", c.lcmConfig.RookNamespace, cephFs.Name))
		} else if cephFs.Status.Phase != cephv1.ConditionReady {
			issues = append(issues, fmt.Sprintf("cephfilesystem '%s/%s' is not ready", c.lcmConfig.RookNamespace, cephFs.Name))
		}
		// count daemons for future checks
		c.healthConfig.sharedFilesystemOpts.mdsDaemonsDesired[cephFs.Name] = map[string]int{"up:active": int(cephFs.Spec.MetadataServer.ActiveCount)}
		if cephFs.Spec.MetadataServer.ActiveStandby {
			c.healthConfig.sharedFilesystemOpts.mdsDaemonsDesired[cephFs.Name]["up:standby-replay"] = int(cephFs.Spec.MetadataServer.ActiveCount)
		} else {
			c.healthConfig.sharedFilesystemOpts.mdsStandbyDesired += int(cephFs.Spec.MetadataServer.ActiveCount)
		}
	}
	return cephFsStatus, issues
}

func (c *cephDeploymentHealthConfig) checkCephObjectStores() (map[string]*cephv1.ObjectStoreStatus, []string) {
	rgwStoreList, err := c.api.Rookclientset.CephV1().CephObjectStores(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, []string{fmt.Sprintf("failed to list cephobjectstores in '%s' namespace", c.lcmConfig.RookNamespace)}
	}
	if len(rgwStoreList.Items) == 0 {
		return nil, nil
	}
	rgwStoresStatus := map[string]*cephv1.ObjectStoreStatus{}
	issues := make([]string, 0)
	for _, rgw := range rgwStoreList.Items {
		// collect some info for future checks
		if len(rgw.Spec.Gateway.ExternalRgwEndpoints) > 0 {
			// Rook supports rgw external for cephcluster external case only
			c.healthConfig.rgwOpts.external = c.healthConfig.cephCluster.Spec.External.Enable
			c.healthConfig.rgwOpts.storeName = rgw.Name
			if rgw.Status != nil {
				// rook uses info map with endpoint and secureEndpoint keys
				if secure, present := rgw.Status.Info["secureEndpoint"]; present {
					c.healthConfig.rgwOpts.externalEndpoint = secure
				} else if notSecure, present := rgw.Status.Info["endpoint"]; present {
					c.healthConfig.rgwOpts.externalEndpoint = notSecure
				}
			}
		} else {
			c.healthConfig.rgwOpts.desiredRgwDaemons += rgw.Spec.Gateway.Instances
			if rgw.Spec.Gateway.DisableMultisiteSyncTraffic || len(rgwStoreList.Items) == 1 {
				c.healthConfig.rgwOpts.storeName = rgw.Name
			}
		}
		rgwStoresStatus[rgw.Name] = rgw.Status
		if rgw.Status == nil {
			issues = append(issues, fmt.Sprintf("cephobjectstore '%s/%s' status is not available yet", c.lcmConfig.RookNamespace, rgw.Name))
		} else if rgw.Status.Phase != cephv1.ConditionReady && rgw.Status.Phase != cephv1.ConditionConnected {
			issues = append(issues, fmt.Sprintf("cephobjectstore '%s/%s' is not ready", c.lcmConfig.RookNamespace, rgw.Name))
		}
	}
	return rgwStoresStatus, issues
}

func (c *cephDeploymentHealthConfig) checkCephObjectUsers() (map[string]*cephv1.ObjectStoreUserStatus, []string) {
	rgwUsersList, err := c.api.Rookclientset.CephV1().CephObjectStoreUsers(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, []string{fmt.Sprintf("failed to list cephobjectusers in '%s' namespace", c.lcmConfig.RookNamespace)}
	}
	if len(rgwUsersList.Items) == 0 {
		return nil, nil
	}
	rgwUsersStatus := map[string]*cephv1.ObjectStoreUserStatus{}
	issues := make([]string, 0)
	for _, user := range rgwUsersList.Items {
		rgwUsersStatus[user.Name] = user.Status
		if user.Status == nil {
			issues = append(issues, fmt.Sprintf("cephobjectuser '%s/%s' status is not available yet", c.lcmConfig.RookNamespace, user.Name))
		} else if user.Status.Phase != "Ready" {
			issues = append(issues, fmt.Sprintf("cephobjectuser '%s/%s' is not ready", c.lcmConfig.RookNamespace, user.Name))
		}
	}
	return rgwUsersStatus, issues
}

func (c *cephDeploymentHealthConfig) checkObjectBucketClaims() (map[string]bktv1alpha1.ObjectBucketClaimStatus, []string) {
	bucketClaimsList, err := c.api.Claimclientset.ObjectbucketV1alpha1().ObjectBucketClaims(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, []string{fmt.Sprintf("failed to list objectbucketclaims in '%s' namespace", c.lcmConfig.RookNamespace)}
	}
	if len(bucketClaimsList.Items) == 0 {
		return nil, nil
	}
	bucketClaimsStatus := map[string]bktv1alpha1.ObjectBucketClaimStatus{}
	issues := make([]string, 0)
	for _, bucket := range bucketClaimsList.Items {
		bucketClaimsStatus[bucket.Name] = bucket.Status
		if bucket.Status.Phase != bktv1alpha1.ObjectBucketClaimStatusPhaseBound {
			issues = append(issues, fmt.Sprintf("objectbucketclaim '%s/%s' is not ready", c.lcmConfig.RookNamespace, bucket.Name))
		}
	}
	return bucketClaimsStatus, issues
}

func (c *cephDeploymentHealthConfig) checkCephObjectRealms() (map[string]*cephv1.Status, []string) {
	realmsList, err := c.api.Rookclientset.CephV1().CephObjectRealms(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, []string{fmt.Sprintf("failed to list cephobjectrealms in '%s' namespace", c.lcmConfig.RookNamespace)}
	}
	if len(realmsList.Items) == 0 {
		return nil, nil
	}
	realmsStatus := map[string]*cephv1.Status{}
	issues := make([]string, 0)
	for _, realm := range realmsList.Items {
		realmsStatus[realm.Name] = realm.Status
		if realm.Status == nil {
			issues = append(issues, fmt.Sprintf("cephobjectrealm '%s/%s' status is not available yet", c.lcmConfig.RookNamespace, realm.Name))
		} else if realm.Status.Phase != "Ready" {
			issues = append(issues, fmt.Sprintf("cephobjectrealm '%s/%s' is not ready", c.lcmConfig.RookNamespace, realm.Name))
		}
	}
	return realmsStatus, issues
}

func (c *cephDeploymentHealthConfig) checkCephObjectZoneGroups() (map[string]*cephv1.Status, []string) {
	zoneGroupsList, err := c.api.Rookclientset.CephV1().CephObjectZoneGroups(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, []string{fmt.Sprintf("failed to list cephobjectzonegroups in '%s' namespace", c.lcmConfig.RookNamespace)}
	}
	if len(zoneGroupsList.Items) == 0 {
		return nil, nil
	}
	zoneGroupsStatus := map[string]*cephv1.Status{}
	issues := make([]string, 0)
	for _, zonegroup := range zoneGroupsList.Items {
		zoneGroupsStatus[zonegroup.Name] = zonegroup.Status
		if zonegroup.Status == nil {
			issues = append(issues, fmt.Sprintf("cephobjectzonegroup '%s/%s' status is not available yet", c.lcmConfig.RookNamespace, zonegroup.Name))
		} else if zonegroup.Status.Phase != "Ready" {
			issues = append(issues, fmt.Sprintf("cephobjectzonegroup '%s/%s' is not ready", c.lcmConfig.RookNamespace, zonegroup.Name))
		}
	}
	return zoneGroupsStatus, issues
}

func (c *cephDeploymentHealthConfig) checkCephObjectZones() (map[string]*cephv1.Status, []string) {
	zonesList, err := c.api.Rookclientset.CephV1().CephObjectZones(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, []string{fmt.Sprintf("failed to list cephobjectzones in '%s' namespace", c.lcmConfig.RookNamespace)}
	}
	if len(zonesList.Items) == 0 {
		return nil, nil
	}
	zonesStatus := map[string]*cephv1.Status{}
	issues := make([]string, 0)
	for _, zone := range zonesList.Items {
		c.healthConfig.rgwOpts.multisite = true
		zonesStatus[zone.Name] = zone.Status
		if zone.Status == nil {
			issues = append(issues, fmt.Sprintf("cephobjectzone '%s/%s' status is not available yet", c.lcmConfig.RookNamespace, zone.Name))
		} else if zone.Status.Phase != "Ready" {
			issues = append(issues, fmt.Sprintf("cephobjectzone '%s/%s' is not ready", c.lcmConfig.RookNamespace, zone.Name))
		}
	}
	return zonesStatus, issues
}
