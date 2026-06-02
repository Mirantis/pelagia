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
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

var (
	versionCheckPollInterval = 20 * time.Second
	versionCheckPollTimeout  = 5 * time.Minute
)

func (c *cephDeploymentConfig) verifyCephVersions() (*lcmcommon.CephVersion, string, string, error) {
	if c.lcmConfig.DeployParams.CephImage == "" {
		return nil, "", "", errors.New("Pelagia lcmconfig has no required 'DEPLOYMENT_CEPH_IMAGE' parameter set")
	}
	currentCephImage := ""
	cephVersionStatus := ""
	var currentCephVersion *lcmcommon.CephVersion
	cephCluster, cephClusterErr := c.api.Rookclientset.CephV1().CephClusters(c.lcmConfig.RookNamespace).Get(c.context, c.cdConfig.cephDpl.Name, metav1.GetOptions{})
	if cephClusterErr != nil {
		if !apierrors.IsNotFound(cephClusterErr) {
			return nil, "", "", errors.Wrapf(cephClusterErr, "failed to get %s/%s cephcluster", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
		}
	} else {
		currentCephImage = cephCluster.Spec.CephVersion.Image
		if isCephDeployed(c.context, *c.log, c.api.Kubeclientset, c.lcmConfig.RookNamespace) {
			if lcmcommon.IsCephToolboxCLIAvailable(c.context, c.api.Kubeclientset, c.lcmConfig.RookNamespace) {
				var err error
				currentCephVersion, cephVersionStatus, err = c.getClusterCephVersion()
				if err != nil {
					return nil, "", "", err
				}
			} else {
				return nil, "", "", errors.Errorf("Pelagia toolbox deployment '%s/%s' is not ready, waiting before proceed any actions",
					c.lcmConfig.RookNamespace, lcmcommon.PelagiaToolBox)
			}
		}
	}
	if (currentCephVersion != nil && currentCephImage != c.lcmConfig.DeployParams.CephImage) || currentCephImage == "" {
		// fully new deployment or image upgrade on live env
		expectedCephVersion, releaseErr := lcmcommon.GetCephVersionByReleaseName(c.lcmConfig.DeployParams.CephRelease)
		if releaseErr != nil {
			return nil, "", "", releaseErr
		}
		// check ceph version from CLI if image changed
		newCephVersion, versionErr := c.getCephVersionFromImage(c.lcmConfig.DeployParams.CephImage)
		if versionErr != nil {
			return nil, "", "", errors.Wrapf(versionErr, "failed to check 'ceph --version' for provided image '%s'", c.lcmConfig.DeployParams.CephImage)
		}
		if newCephVersion.Name != expectedCephVersion.Name {
			return nil, "", "", errors.Errorf("expected Ceph release %s '%s' version, but specified %s '%s' version (image: %s)",
				expectedCephVersion.Name, expectedCephVersion.MajorVersion, newCephVersion.Name, newCephVersion.MajorVersion, c.lcmConfig.DeployParams.CephImage)
		}
		if currentCephVersion == nil {
			currentCephVersion = newCephVersion
			currentCephImage = c.lcmConfig.DeployParams.CephImage
		} else {
			if newCephVersion.Order < currentCephVersion.Order {
				return nil, "", "", errors.Errorf("detected Ceph version downgrade from '%s.%s' to '%s.%s': major downgrade is not possible",
					currentCephVersion.MajorVersion, currentCephVersion.MinorVersion, newCephVersion.MajorVersion, newCephVersion.MinorVersion)
			}
			if newCephVersion.Order-currentCephVersion.Order > 1 {
				return nil, "", "", errors.Errorf("detected Ceph version upgrade from '%s.%s' to '%s.%s': upgrade with step over one major version is not possible",
					currentCephVersion.MajorVersion, currentCephVersion.MinorVersion, newCephVersion.MajorVersion, newCephVersion.MinorVersion)
			}
			// no major/minor Ceph version - no Ceph version change -> use new image
			// otherwise check version change is allowed
			if newCephVersion.Order != currentCephVersion.Order || newCephVersion.MinorVersion != currentCephVersion.MinorVersion {
				c.log.Info().Msgf("detected Ceph version change: current is '%s.%s', new '%s.%s'",
					currentCephVersion.MajorVersion, currentCephVersion.MinorVersion, newCephVersion.MajorVersion, newCephVersion.MinorVersion)
				upgradeAllowed, upgradeErr := c.cephUpgradeAllowed()
				if upgradeErr != nil {
					return nil, "", "", errors.Wrap(upgradeErr, "failed to check is Ceph upgrade allowed")
				}
				if upgradeAllowed {
					// updating current image to use and wait until it is updated in cephcluster
					return currentCephVersion, c.lcmConfig.DeployParams.CephImage, cephVersionStatus, nil
				}
			} else {
				currentCephImage = c.lcmConfig.DeployParams.CephImage
			}
		}
	} else if currentCephVersion == nil {
		// cephcluster is present, but not yet provisioned due to some
		// reasons, so get current version from its image and continue
		// check ceph version from CLI if image changed
		// deployment should be already there - so must be quick
		newCephVersion, versionErr := c.getCephVersionFromImage(currentCephImage)
		if versionErr != nil {
			return nil, "", "", errors.Wrapf(versionErr, "failed to check 'ceph --version' for used in cluster image '%s'", currentCephImage)
		}
		currentCephVersion = newCephVersion
	} else {
		// all versions are available, drop deployment if present
		_, err := c.deleteVersionCheckDeployment()
		if err != nil {
			c.log.Error().Err(err).Msg("")
		}
	}
	return currentCephVersion, currentCephImage, cephVersionStatus, nil
}

func (c *cephDeploymentConfig) prepareVersionCheckDeployment(targetImage string) error {
	ownerRefs, err := lcmcommon.GetObjectOwnerRef(c.cdConfig.cephDpl, c.api.Scheme)
	if err != nil {
		msg := fmt.Sprintf("failed to prepare ownerRefs for CephDeployment '%s/%s'", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
		c.log.Error().Err(err).Msg(msg)
		return err
	}

	versionDpl := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            pelagiaVersionCheckDpl,
			Namespace:       c.cdConfig.cephDpl.Namespace,
			Labels:          lcmcommon.ExtendLabels(map[string]string{"app": pelagiaVersionCheckDpl}, baseResourceLabels),
			OwnerReferences: ownerRefs,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": pelagiaVersionCheckDpl},
			},
			Replicas:                lcmcommon.PtrTo(int32(1)),
			RevisionHistoryLimit:    lcmcommon.PtrTo(int32(2)),
			ProgressDeadlineSeconds: lcmcommon.PtrTo(int32(5)),
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": pelagiaVersionCheckDpl},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    pelagiaVersionCheckDpl,
							Image:   targetImage,
							Command: []string{"tail", "-f", "/dev/null"},
							SecurityContext: &corev1.SecurityContext{
								Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
								AllowPrivilegeEscalation: lcmcommon.PtrTo(false),
								ReadOnlyRootFilesystem:   lcmcommon.PtrTo(true),
							},
							ImagePullPolicy:          "IfNotPresent",
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: "File",
						},
					},
					SecurityContext:               &corev1.PodSecurityContext{},
					DNSPolicy:                     "ClusterFirstWithHostNet",
					RestartPolicy:                 "Always",
					TerminationGracePeriodSeconds: lcmcommon.PtrTo(int64(30)),
				},
			},
		},
	}
	checkDpl, err := c.api.Kubeclientset.AppsV1().Deployments(versionDpl.Namespace).Get(c.context, versionDpl.Name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			c.log.Error().Err(err).Msg("")
			return errors.Wrapf(err, "failed to get '%s/%s' deployment", versionDpl.Namespace, versionDpl.Name)
		}
		c.log.Info().Msgf("running Ceph version check for image '%s'", targetImage)
		_, err = c.api.Kubeclientset.AppsV1().Deployments(versionDpl.Namespace).Create(c.context, versionDpl, metav1.CreateOptions{})
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return errors.Wrapf(err, "failed to create '%s/%s' deployment", versionDpl.Namespace, versionDpl.Name)
		}
	} else {
		versionDpl.Spec.Template.Spec.SchedulerName = checkDpl.Spec.Template.Spec.SchedulerName
		if !reflect.DeepEqual(checkDpl.Spec, versionDpl.Spec) {
			c.log.Info().Msgf("restarting Ceph version check for image '%s'", targetImage)
			lcmcommon.ShowObjectDiff(*c.log, checkDpl.Spec, versionDpl.Spec)
			checkDpl.Spec = versionDpl.Spec
			_, err = c.api.Kubeclientset.AppsV1().Deployments(c.lcmConfig.RookNamespace).Update(c.context, checkDpl, metav1.UpdateOptions{})
			if err != nil {
				c.log.Error().Err(err).Msg("")
				return errors.Wrapf(err, "failed to update '%s/%s' deployment", versionDpl.Namespace, versionDpl.Name)
			}
		}
	}
	err = wait.PollUntilContextTimeout(c.context, versionCheckPollInterval, versionCheckPollTimeout, true, func(_ context.Context) (bool, error) {
		checkDpl, err := c.api.Kubeclientset.AppsV1().Deployments(versionDpl.Namespace).Get(c.context, versionDpl.Name, metav1.GetOptions{})
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return false, nil
		}
		if !lcmcommon.IsDeploymentReady(checkDpl) {
			c.log.Info().Msgf("waiting version-check deployment '%s/%s' readiness...", versionDpl.Namespace, versionDpl.Name)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrap(err, "timeout reached for waiting version-check deployment ready")
	}
	return err
}

func (c *cephDeploymentConfig) deleteVersionCheckDeployment() (bool, error) {
	err := c.api.Kubeclientset.AppsV1().Deployments(c.cdConfig.cephDpl.Namespace).Delete(c.context, pelagiaVersionCheckDpl, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, errors.Wrapf(err, "failed to delete '%s/%s' deployment", c.cdConfig.cephDpl.Namespace, pelagiaVersionCheckDpl)
	}
	c.log.Info().Msgf("removing version-check deployment '%s/%s'", c.cdConfig.cephDpl.Namespace, pelagiaVersionCheckDpl)
	return true, nil
}

func (c *cephDeploymentConfig) getCephVersionFromImage(image string) (*lcmcommon.CephVersion, error) {
	dplErr := c.prepareVersionCheckDeployment(image)
	if dplErr != nil {
		return nil, errors.Wrap(dplErr, "failed to prepare version-check deployment")
	}
	cephVersionCLI, err := c.getCephVersion()
	if err != nil {
		return nil, err
	}
	newCephVersion, err := lcmcommon.ParseCephVersion(cephVersionCLI)
	if err != nil {
		return nil, err
	}
	return newCephVersion, nil
}

func (c *cephDeploymentConfig) getClusterCephVersion() (*lcmcommon.CephVersion, string, error) {
	cephVersions, err := c.getCephVersions()
	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to get current Ceph versions")
	}
	versionsForStatus := []string{}
	var clusterCephVersion *lcmcommon.CephVersion
	for version := range cephVersions.Overall {
		cephVersion, err := lcmcommon.ParseCephVersion(version)
		if err != nil {
			return nil, "", errors.Wrap(err, "failed to verify Ceph version in cluster")
		}
		versionsForStatus = append(versionsForStatus, fmt.Sprintf("%s.%s", cephVersion.MajorVersion, cephVersion.MinorVersion))
		if clusterCephVersion == nil || clusterCephVersion.Order > cephVersion.Order {
			clusterCephVersion = cephVersion
		}
	}
	// since we may have right now upgrade and multiple versions set
	// first version is ALWAYS must be lower version
	sort.Strings(versionsForStatus)
	return clusterCephVersion, strings.Join(versionsForStatus, ","), nil
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
	c.log.Debug().Msg("ensure rook image version is consistent with the current ceph version")
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
