/*
Copyright 2026 Mirantis IT.

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

	csiopapi "github.com/ceph/ceph-csi-operator/api/v1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	vsapi "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/v3/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/v3/pkg/common"
)

const (
	// names are used initially from Rook
	// so need to keep them for backward compatibilty for existing clusters
	rbdVSClassName    = "csi-rbdplugin-snapclass"
	cephfsVSClassName = "csi-cephfsplugin-snapclass"
)

func (c *cephDeploymentConfig) ensureVSCResources() (bool, error) {
	if !c.lcmConfig.DeployParams.ManageVolumeSnapshotClasses {
		c.log.Warn().Msg("ensure volumesnapshotclasses resources is disabled for Pelagia")
		return false, nil
	}

	vsclasses := map[string]string{
		rbdVSClassName:    fmt.Sprintf(cephCsiDriverNameTemplate, c.lcmConfig.RookNamespace, cephlcmv1alpha1.RBDCSIDriver),
		cephfsVSClassName: fmt.Sprintf(cephCsiDriverNameTemplate, c.lcmConfig.RookNamespace, cephlcmv1alpha1.CephFSCSIDriver),
	}
	changed := false
	errs := 0
	for vsclassName, pluginName := range vsclasses {
		vsclass := vsapi.VolumeSnapshotClass{}
		vsclass.Name = vsclassName
		vsclassErr := c.api.ClientNoCache.Get(c.context, crclient.ObjectKeyFromObject(&vsclass), &vsclass)
		if vsclassErr != nil && !apierrors.IsNotFound(vsclassErr) {
			errs++
			c.log.Error().Err(vsclassErr).Msgf("failed to check VolumeSnapshotClass '%s'", vsclass.Name)
			continue
		}
		cephDriver := csiopapi.Driver{}
		cephDriver.Name = pluginName
		cephDriver.Namespace = c.lcmConfig.RookNamespace
		cephDriverErr := c.api.ClientNoCache.Get(c.context, crclient.ObjectKeyFromObject(&cephDriver), &cephDriver)
		if cephDriverErr != nil {
			if !apierrors.IsNotFound(cephDriverErr) {
				errs++
				c.log.Error().Err(cephDriverErr).Msgf("failed to check cephcsi Driver '%s/%s'", cephDriver.Namespace, cephDriver.Name)
				continue
			}
			c.log.Debug().Msgf("cephcsi Driver '%s/%s' is not found, ensure VolumeSnapshotClass '%s' not present", cephDriver.Namespace, cephDriver.Name, vsclass.Name)
			if vsclassErr == nil {
				c.log.Info().Msgf("removing VolumeSnapshotClass '%s'", vsclass.Name)
				err := c.api.ClientNoCache.Delete(c.context, &vsclass)
				if err != nil {
					errs++
					c.log.Error().Msgf("failed to remove VolumeSnapshotClass '%s'", vsclass.Name)
					continue
				}
				changed = true
			}
		} else {
			newVSClass := generateVSClass(vsclassName, c.lcmConfig.RookNamespace, pluginName)
			if vsclassErr != nil {
				c.log.Info().Msgf("creating VolumeSnapshotClass '%s'", vsclass.Name)
				err := c.api.ClientNoCache.Create(c.context, &newVSClass)
				if err != nil {
					c.log.Error().Err(err).Msg("")
					errs++
					continue
				}
				changed = true
			} else {
				updateRequired := lcmcommon.AlignBaseLabels(*c.log, "VolumeSnapshotClass", &vsclass.ObjectMeta, newVSClass.Labels)
				if updateRequired {
					c.log.Info().Msgf("updating VolumeSnapshotClass labels '%s'", vsclass.Name)
					err := c.api.ClientNoCache.Update(c.context, &vsclass)
					if err != nil {
						c.log.Error().Err(err).Msg("")
						errs++
						continue
					}
					changed = true
				}
			}
		}
	}
	if errs > 0 {
		return false, errors.New("failed to ensure VolumeSnapshotClasses")
	}
	return changed, nil
}

func (c *cephDeploymentConfig) deleteVolumeSnapshotClasses() (bool, error) {
	if !c.lcmConfig.DeployParams.ManageVolumeSnapshotClasses {
		c.log.Warn().Msg("ensure volumesnapshotclasses resources is disabled for Pelagia, skipping cleanup")
		return true, nil
	}
	vscs := &vsapi.VolumeSnapshotClassList{}
	// get only created by pelagia and remove
	err := c.api.ClientNoCache.List(c.context, vscs, &crclient.ListOptions{LabelSelector: labels.Set(baseResourceLabels).AsSelector()})
	if err != nil {
		return false, errors.Wrap(err, "failed to list VolumeSnapshotClasses")
	}
	removed := true
	for _, vsc := range vscs.Items {
		// remove all drivers in our namespace
		removed = false
		err := c.api.ClientNoCache.Delete(c.context, &vsc)
		if err != nil {
			return false, errors.Wrapf(err, "failed to delete VolumeSnapshotClass '%s'", vsc.Name)
		}
		c.log.Info().Msgf("removing VolumeSnapshotClass '%s'", vsc.Name)
	}
	return removed, nil
}

func generateVSClass(vsclassName, clusterNamespace, driverName string) vsapi.VolumeSnapshotClass {
	secretMapping := map[string]string{
		rbdVSClassName:    "rook-csi-rbd-provisioner",
		cephfsVSClassName: "rook-csi-cephfs-provisioner",
	}
	return vsapi.VolumeSnapshotClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   vsclassName,
			Labels: baseResourceLabels,
		},
		Driver: driverName,
		Parameters: map[string]string{
			"clusterID": clusterNamespace,
			"csi.storage.k8s.io/snapshotter-secret-name":      secretMapping[vsclassName],
			"csi.storage.k8s.io/snapshotter-secret-namespace": clusterNamespace,
		},
		DeletionPolicy: vsapi.VolumeSnapshotContentDelete,
	}
}
