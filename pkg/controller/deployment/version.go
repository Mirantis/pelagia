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
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentConfig) verifyCephVersions() (*lcmcommon.CephVersion, string, string, error) {
	desiredCephVersion, err := lcmcommon.CheckExpectedCephVersion(c.lcmConfig.DeployParams.CephImage, c.lcmConfig.DeployParams.CephRelease)
	if err != nil {
		return nil, "", "", errors.Wrap(err, "failed to check desired ceph version")
	}
	cephVersionFromStatus := c.cdConfig.cephDpl.Status.ClusterVersion
	// check basic case, when no ceph version changes
	// if desired Ceph version same as in status - no matter if image changes with CVE
	if cephVersionFromStatus != "" && cephVersionFromStatus == fmt.Sprintf("%s.%s", desiredCephVersion.MajorVersion, desiredCephVersion.MinorVersion) {
		return desiredCephVersion, c.lcmConfig.DeployParams.CephImage, fmt.Sprintf("%s.%s", desiredCephVersion.MajorVersion, desiredCephVersion.MinorVersion), nil
	}
	cephCluster, cephClusterErr := c.api.Rookclientset.CephV1().CephClusters(c.lcmConfig.RookNamespace).Get(c.context, c.cdConfig.cephDpl.Name, metav1.GetOptions{})
	// if no version set in status - set it first and after that wait
	// for re-run reconcile before proceed with further checks
	if cephVersionFromStatus == "" {
		// if no version set in status - check that cephcluster present, otherwise fresh deploy
		if cephClusterErr != nil {
			if !apierrors.IsNotFound(cephClusterErr) {
				return nil, "", "", errors.Wrapf(cephClusterErr, "failed to get %s/%s cephcluster", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
			}
			return desiredCephVersion, c.lcmConfig.DeployParams.CephImage, "", nil
		}
		if isCephDeployed(c.context, *c.log, c.api.Kubeclientset, c.lcmConfig.RookNamespace) {
			if lcmcommon.IsCephToolboxCLIAvailable(c.context, c.api.Kubeclientset, c.lcmConfig.RookNamespace) {
				cephVersion, cephVersionForStatus, err := c.checkVersionsFromCli()
				if err != nil {
					return nil, "", "", err
				}
				return cephVersion, cephCluster.Spec.CephVersion.Image, cephVersionForStatus, nil
			}
			// case when tools pod/deployment is broken and can be fixed during its reconcile function
			// and we can't predict which current version is, because it may be upgrade/downgrade.
			// So if Ceph deployed - we need check versions from ceph cli for sure
			// do not fail, since it may be fixed during ceph-tools ensure, but
			// after ceph-tools ensure we need to re-run overall reconcile to set correct version
			return nil, cephCluster.Spec.CephVersion.Image, "", nil
		}
		// avoid extra parsing if image same as desired
		if cephCluster.Spec.CephVersion.Image == c.lcmConfig.DeployParams.CephImage {
			return desiredCephVersion, c.lcmConfig.DeployParams.CephImage, "", nil
		}
		cephVersion, err := lcmcommon.ParseCephVersion(lcmcommon.GetCephVersionFromImage(cephCluster.Spec.CephVersion.Image))
		if err != nil {
			return nil, "", "", errors.Wrap(err, "failed to verify Ceph version in CephCluster spec")
		}
		return cephVersion, cephCluster.Spec.CephVersion.Image, "", nil
	}
	// since version is set in status, that means ceph cluster is operational - any error
	// should be interpreted as reconcile error
	if cephClusterErr != nil {
		return nil, "", "", errors.Wrapf(cephClusterErr, "failed to get %s/%s CephCluster", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
	}
	// since we may have right now upgrade and multiple versions set
	// first version is ALWAYS must be lower version
	curVersion := strings.Split(cephVersionFromStatus, ",")[0]
	currentCephVersion, err := lcmcommon.ParseCephVersion(curVersion)
	if err != nil {
		return nil, "", "", errors.Wrap(err, "failed to verify current Ceph version")
	}
	if desiredCephVersion.Order < currentCephVersion.Order {
		return nil, "", "", errors.Errorf("detected Ceph version downgrade from '%s.%s' to '%s.%s': downgrade is not possible",
			currentCephVersion.MajorVersion, currentCephVersion.MinorVersion, desiredCephVersion.MajorVersion, desiredCephVersion.MinorVersion)
	}
	if desiredCephVersion.Order-currentCephVersion.Order > 1 {
		return nil, "", "", errors.Errorf("detected Ceph version upgrade from '%s.%s' to '%s.%s': upgrade with step over one major version is not possible",
			currentCephVersion.MajorVersion, currentCephVersion.MinorVersion, desiredCephVersion.MajorVersion, desiredCephVersion.MinorVersion)
	}
	// if image is not updated yet in cephcluster spec - check is it possible to update or not
	if cephCluster.Spec.CephVersion.Image != c.lcmConfig.DeployParams.CephImage {
		c.log.Info().Msgf("detected Ceph version change: new '%s.%s', current is '%s.%s'", desiredCephVersion.MajorVersion, desiredCephVersion.MinorVersion,
			currentCephVersion.MajorVersion, currentCephVersion.MinorVersion)
		upgradeAllowed, err := c.cephUpgradeAllowed()
		if err != nil {
			return nil, "", "", errors.Wrap(err, "failed to check is Ceph upgrade allowed")
		}
		if upgradeAllowed {
			// updating current image to use and wait until it is updated in cephcluster
			return currentCephVersion, c.lcmConfig.DeployParams.CephImage, curVersion, nil
		}
		return currentCephVersion, cephCluster.Spec.CephVersion.Image, curVersion, nil
	}
	if lcmcommon.IsCephToolboxCLIAvailable(c.context, c.api.Kubeclientset, c.lcmConfig.RookNamespace) {
		cephVersion, cephVersionForStatus, err := c.checkVersionsFromCli()
		if err != nil {
			return nil, "", "", err
		}
		return cephVersion, cephCluster.Spec.CephVersion.Image, cephVersionForStatus, nil
	}
	c.log.Warn().Msgf("can't verify current Ceph version, %s is not available, waiting", lcmcommon.PelagiaToolBox)
	// case when tools pod/deployment is broken and can be fixed during its reconcile function
	return currentCephVersion, cephCluster.Spec.CephVersion.Image, c.cdConfig.cephDpl.Status.ClusterVersion, nil
}

func (c *cephDeploymentConfig) checkVersionsFromCli() (*lcmcommon.CephVersion, string, error) {
	cephVersions, err := c.getCephVersions()
	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to get current Ceph versions")
	}
	versionsForStatus := []string{}
	for version := range cephVersions.Overall {
		versionPattern := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
		parsedVersion := versionPattern.FindString(version)
		versionsForStatus = append(versionsForStatus, fmt.Sprintf("v%s", parsedVersion))
	}
	// since we may have right now upgrade and multiple versions set
	// first version is ALWAYS must be lower version
	sort.Strings(versionsForStatus)
	cephVersion, err := lcmcommon.ParseCephVersion(versionsForStatus[0])
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to verify Ceph version in cluster")
	}
	return cephVersion, strings.Join(versionsForStatus, ","), nil
}

func (c *cephDeploymentConfig) ensureCephClusterVersion() error {
	cephCluster, err := c.api.Rookclientset.CephV1().CephClusters(c.lcmConfig.RookNamespace).Get(c.context, c.cdConfig.cephDpl.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "failed to get %s/%s CephCluster", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
	}
	if cephCluster.Spec.CephVersion.Image != c.cdConfig.currentCephImage {
		c.log.Info().Msgf("updating CephCluster image from '%s' to '%s'", cephCluster.Spec.CephVersion.Image, c.cdConfig.currentCephImage)
		cephCluster.Spec.CephVersion.Image = c.cdConfig.currentCephImage
		// Remove hostNetwork because Rook now fails validation if it is set within
		// provider=host
		cephCluster.Spec.Network.HostNetwork = false
		// drop osd restart reasons if present, because we dont want to keep them once image changed
		// if reason is not set
		if c.cdConfig.cephDpl.Spec.ExtraOpts == nil || c.cdConfig.cephDpl.Spec.ExtraOpts.OsdRestartReason == "" {
			delete(cephCluster.Annotations, cephRestartOsdLabel)
			delete(cephCluster.Annotations, cephRestartOsdTimestampLabel)
			delete(cephCluster.Spec.Annotations, cephv1.KeyOSD)
		}
		_, err := c.api.Rookclientset.CephV1().CephClusters(c.lcmConfig.RookNamespace).Update(c.context, cephCluster, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to update CephCluster %s/%s version", cephCluster.Namespace, cephCluster.Name)
		}
		return errors.Errorf("update CephCluster %s/%s version is in progress", cephCluster.Namespace, cephCluster.Name)
	}
	return nil
}

func (c *cephDeploymentConfig) ensureRookImage() error {
	c.log.Info().Msg("ensure rook image version is consistent with the current ceph version")
	operator, err := c.api.Kubeclientset.AppsV1().Deployments(c.lcmConfig.RookNamespace).Get(c.context, lcmcommon.RookCephOperatorName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get %s/%s deployment", c.lcmConfig.RookNamespace, lcmcommon.RookCephOperatorName)
	}
	if *operator.Spec.Replicas == 0 {
		c.log.Info().Msgf("skipping rook image consistency verification due to %s/%s deployment is scaled to zero", c.lcmConfig.RookNamespace, lcmcommon.RookCephOperatorName)
		return nil
	}

	if operator.Spec.Template.Spec.Containers[0].Image != c.lcmConfig.DeployParams.RookImage {
		c.log.Info().Msgf("rook image in %s/%s deployment is different from the current release, updating", c.lcmConfig.RookNamespace, lcmcommon.RookCephOperatorName)
		operator.Spec.Template.Spec.Containers[0].Image = c.lcmConfig.DeployParams.RookImage
		_, err = c.api.Kubeclientset.AppsV1().Deployments(c.lcmConfig.RookNamespace).Update(c.context, operator, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to update %s/%s deployment with new rook image", c.lcmConfig.RookNamespace, lcmcommon.RookCephOperatorName)
		}
		return errors.Errorf("deployment %s/%s rook image update is in progress", c.lcmConfig.RookNamespace, lcmcommon.RookCephOperatorName)
	} else if !lcmcommon.IsDeploymentReady(operator) {
		return errors.Errorf("deployment %s/%s rook image update still is in progress", c.lcmConfig.RookNamespace, lcmcommon.RookCephOperatorName)
	}

	if isCephDeployed(c.context, *c.log, c.api.Kubeclientset, c.lcmConfig.RookNamespace) {
		discover, err := c.api.Kubeclientset.AppsV1().DaemonSets(c.lcmConfig.RookNamespace).Get(c.context, lcmcommon.RookDiscoverName, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to get %s/%s daemonset", c.lcmConfig.RookNamespace, lcmcommon.RookDiscoverName)
		}
		if discover.Spec.Template.Spec.Containers[0].Image != c.lcmConfig.DeployParams.RookImage {
			c.log.Info().Msgf("rook image in %s/%s daemonset is different from the current release, updating", c.lcmConfig.RookNamespace, lcmcommon.RookDiscoverName)
			discover.Spec.Template.Spec.Containers[0].Image = c.lcmConfig.DeployParams.RookImage
			_, err = c.api.Kubeclientset.AppsV1().DaemonSets(c.lcmConfig.RookNamespace).Update(c.context, discover, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update %s/%s daemonset with new rook image", c.lcmConfig.RookNamespace, lcmcommon.RookDiscoverName)
			}
			return errors.Errorf("daemonset %s/%s rook image update is in progress", c.lcmConfig.RookNamespace, lcmcommon.RookDiscoverName)
		} else if !lcmcommon.IsDaemonSetReady(discover) {
			// external case, we may not have rook-discover pods at all by design
			if discover.Status.DesiredNumberScheduled == 0 {
				return nil
			}
			return errors.Errorf("daemonset %s/%s rook image update still is in progress", c.lcmConfig.RookNamespace, lcmcommon.RookDiscoverName)
		}
	}
	return nil
}
