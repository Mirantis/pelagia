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
	"os"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

const (
	// Env vars important for ceph maintenance
	controllerClusterReleaseVar = "CEPH_CONTROLLER_CLUSTER_RELEASE"
)

// CephUpgradeAllowed stands for upgrade order if Ceph is integrated with OpenStack delivered with
// Mirantis/rockoon project
func (c *cephDeploymentConfig) cephUpgradeAllowed() (bool, error) {
	c.log.Debug().Msg("checking if Ceph version can be updated for CephCluster")
	clusterRelease, found := os.LookupEnv(controllerClusterReleaseVar)
	if !found {
		return false, errors.Errorf("required env variable '%s' is not set", controllerClusterReleaseVar)
	}

	// verify osdplst state and release and rely allowing ceph upgrade on
	// these values
	osState, osRelease, err := c.getOpenstackDeploymentStatus()
	if err != nil {
		return false, errors.Wrap(err, "failed to get openstackdeploymentstatus state and release")
	}

	// Do not check osdpl if there is no MOSK on cluster
	if osRelease == "" && osState == "" {
		return true, nil
	}

	if osRelease != clusterRelease {
		c.log.Info().Msgf("current Openstack deployment release version is '%v', but Pelagia has '%v', Ceph version upgrade is postponed", osRelease, clusterRelease)
		return false, nil
	}
	if osState != osDplReadyState {
		c.log.Info().Msgf("Openstack deployment is not in '%v' state, Ceph version upgrade is postponed", osDplReadyState)
		return false, nil
	}
	return true, nil
}

func (c *cephDeploymentConfig) getOpenstackDeploymentStatus() (string, string, error) {
	u := &unstructured.UnstructuredList{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "lcm.mirantis.com",
		Kind:    "OpenStackDeploymentStatusList",
		Version: "v1alpha1",
	})
	err := c.api.Client.List(c.context, u, &client.ListOptions{Namespace: "openstack"})
	if client.IgnoreNotFound(err) != nil {
		if !meta.IsNoMatchError(err) && !runtime.IsNotRegisteredError(err) {
			return "", "", errors.Wrap(err, "failed to list openstackdeploymentstatus objects")
		}
		return "", "", nil
	}
	if len(u.Items) == 0 {
		c.log.Info().Msg("OpenstackDeploymentStatus object not found, skipping openstack release wait")
		return "", "", nil
	}
	if len(u.Items) != 1 {
		return "", "", errors.Errorf("expected number of openstackdeploymentstatus objects is 1, but found %d", len(u.Items))
	}
	// osdplst valuable example:
	// ---
	// apiVersion: lcm.mirantis.com/v1alpha1
	// kind: OpenStackDeploymentStatus
	// metadata:
	//   name: osh-dev
	//   namespace: openstack
	// status:
	//   osdpl:
	//    cause: update
	//    changes: (('change', ('ceph', 'secret', 'hash'), '1e9e7bd75f202378e81a90161452fb31e1370ed4c6e38a5967f948290a2b29a0',
	//      'debca4bdd4aa077d47ba5bcb4eeabe48a130e91075762edd1280c3367146f986'),)
	//    controller_version: 0.13.1.dev44
	//    fingerprint: 562ea8c37a50692075a0340ab4784fe82a12e9cd9851137026a2bc1d1aa46ba5
	//    openstack_version: victoria
	//    release: 11.7.3+3.5.7
	//    state: APPLIED
	//    timestamp: "2023-05-16 12:53:18.759886"
	uObj := u.Items[0].Object
	if uObj["status"] != nil && uObj["status"].(map[string]interface{})["osdpl"] != nil {
		osdplst := uObj["status"].(map[string]interface{})["osdpl"].(map[string]interface{})
		state := ""
		release := ""
		if osdplst["state"] != nil {
			state = osdplst["state"].(string)
		}
		if osdplst["release"] != nil {
			release = osdplst["release"].(string)
		}
		c.log.Info().Msgf("OpenstackDeploymentStatus object found with state '%s' of release '%s'", state, release)
		return state, release, nil
	}

	return "", "", errors.Errorf("OpenstackDeploymentStatus required values in status.osdpl not found")
}

func (c *cephDeploymentConfig) isMaintenanceActing() (bool, error) {
	return lcmcommon.IsClusterMaintenanceActing(c.context, c.api.CephLcmclientset, c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
}
