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

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

func (c *cephDeploymentConfig) ensureSharedFilesystem() (bool, error) {
	c.log.Debug().Msg("ensure Ceph shared filesystems")
	facedErrors := make([]error, 0)
	changed := false
	if c.cdConfig.cephDpl.Spec.SharedFilesystem != nil {
		cephfsChanged, err := c.ensureCephFS()
		if err != nil {
			c.log.Error().Err(err).Msg("failed to ensure shared filesystem")
			facedErrors = append(facedErrors, err)
		}
		changed = cephfsChanged
	} else {
		removed, err := c.deleteSharedFilesystems()
		if err != nil {
			c.log.Error().Err(err).Msg("failed to remove shared filesystem")
			facedErrors = append(facedErrors, err)
		}
		changed = !removed
	}
	if len(facedErrors) > 0 {
		msg := "errors faced during Ceph shared filesystems ensure"
		return false, errors.New(msg)
	}
	return changed, nil
}

func (c *cephDeploymentConfig) ensureCephFS() (bool, error) {
	c.log.Debug().Msg("ensure CephFS")
	cephFsList, err := c.api.Rookclientset.CephV1().CephFilesystems(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to get CephFS list")
	}
	dropFS := map[string]bool{}
	for _, cephFs := range cephFsList.Items {
		dropFS[cephFs.Name] = true
	}
	fsErrors := make([]error, 0)
	changed := false
	for _, cephDplCephFS := range c.cdConfig.cephDpl.Spec.SharedFilesystem.CephFS {
		delete(dropFS, cephDplCephFS.Name)
		createInProgress := false
		cephFsResource := generateCephFS(cephDplCephFS, c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Spec.HyperConverge)
		cephFs, err := c.api.Rookclientset.CephV1().CephFilesystems(c.lcmConfig.RookNamespace).Get(c.context, cephDplCephFS.Name, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				c.log.Error().Err(err).Msg("failed to get CephFS")
				fsErrors = append(fsErrors, errors.Wrapf(err, "failed to get CephFS %s/%s", c.lcmConfig.RookNamespace, cephDplCephFS.Name))
				continue
			}
			c.log.Info().Msgf("creating CephFS %s/%s", c.lcmConfig.RookNamespace, cephDplCephFS.Name)
			_, err := c.api.Rookclientset.CephV1().CephFilesystems(c.lcmConfig.RookNamespace).Create(c.context, cephFsResource, metav1.CreateOptions{})
			if err != nil {
				c.log.Error().Err(err).Msg("failed to create CephFS")
				fsErrors = append(fsErrors, errors.Wrapf(err, "failed to create CephFS %s/%s", c.lcmConfig.RookNamespace, cephDplCephFS.Name))
				continue
			}
			changed = true
			createInProgress = true
		}
		// if some resource created - go to next cephfs
		if createInProgress {
			continue
		}
		if !reflect.DeepEqual(cephFsResource.Spec, cephFs.Spec) {
			c.log.Info().Msgf("updating CephFS %s/%s", c.lcmConfig.RookNamespace, cephDplCephFS.Name)
			lcmcommon.ShowObjectDiff(*c.log, cephFs.Spec, cephFsResource.Spec)
			cephFs.Spec = cephFsResource.Spec
			_, err := c.api.Rookclientset.CephV1().CephFilesystems(c.lcmConfig.RookNamespace).Update(c.context, cephFs, metav1.UpdateOptions{})
			if err != nil {
				c.log.Error().Err(err).Msg("failed to update CephFS")
				fsErrors = append(fsErrors, errors.Wrapf(err, "failed to update CephFS %s/%s", c.lcmConfig.RookNamespace, cephDplCephFS.Name))
				continue
			}
			changed = true
		}
		subvolumegroup, err := c.cephFSSubvolumegroupCommand("ls", cephDplCephFS.Name)
		if err != nil {
			errMsg := errors.Wrapf(err, "failed to list CephFS %s subvolumegroup", cephDplCephFS.Name)
			c.log.Error().Err(errMsg)
			fsErrors = append(fsErrors, errMsg)
		} else if subvolumegroup == "" {
			c.log.Info().Msgf("creating CephFS %s/%s subvolumegroup for CSI", c.lcmConfig.RookNamespace, cephDplCephFS.Name)
			_, err = c.cephFSSubvolumegroupCommand("create", cephDplCephFS.Name)
			if err != nil {
				errMsg := errors.Wrapf(err, "failed to create CephFS %s subvolumegroup", cephDplCephFS.Name)
				c.log.Error().Err(errMsg)
				fsErrors = append(fsErrors, errMsg)
				continue
			}
			changed = true
		}
	}
	for fsName := range dropFS {
		subvolumegroup, err := c.cephFSSubvolumegroupCommand("ls", fsName)
		if err != nil {
			errMsg := errors.Wrapf(err, "failed to list CephFS %s subvolumegroup", fsName)
			c.log.Error().Err(errMsg)
			fsErrors = append(fsErrors, errMsg)
			continue
		} else if subvolumegroup != "" {
			c.log.Info().Msgf("removing CephFS %s/%s subvolumegroup for CSI", c.lcmConfig.RookNamespace, fsName)
			_, err = c.cephFSSubvolumegroupCommand("rm", fsName)
			if err != nil {
				errMsg := errors.Wrapf(err, "failed to remove CephFS %s subvolumegroup", fsName)
				c.log.Error().Err(errMsg)
				fsErrors = append(fsErrors, errMsg)
				continue
			}
		}
		c.log.Info().Msgf("removing CephFS %s/%s", c.lcmConfig.RookNamespace, fsName)
		err = c.api.Rookclientset.CephV1().CephFilesystems(c.lcmConfig.RookNamespace).Delete(c.context, fsName, metav1.DeleteOptions{})
		if err != nil {
			c.log.Error().Err(err).Msg("failed to remove CephFS")
			if apierrors.IsNotFound(err) {
				continue
			}
			fsErrors = append(fsErrors, errors.Wrapf(err, "failed to delete CephFS %s/%s", c.lcmConfig.RookNamespace, fsName))
		}
		changed = true
	}
	if len(fsErrors) > 0 {
		if len(fsErrors) > 1 {
			msg := "multiple errors during cephFS ensure"
			return false, errors.New(msg)
		}
		return false, fsErrors[0]
	}
	return changed, nil
}

func (c *cephDeploymentConfig) deleteSharedFilesystems() (bool, error) {
	cephFsList, err := c.api.Rookclientset.CephV1().CephFilesystems(c.lcmConfig.RookNamespace).List(c.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to get cephFS list")
	}
	if len(cephFsList.Items) == 0 {
		delete(resourceUpdateTimestamps.cephConfigMap, "mds")
		return true, nil
	}
	issues := 0
	for _, cephFs := range cephFsList.Items {
		c.log.Info().Msgf("removing CephFS %s/%s subvolumegroup for CSI", c.lcmConfig.RookNamespace, cephFs.Name)
		subvolumegroup, err := c.cephFSSubvolumegroupCommand("ls", cephFs.Name)
		if err != nil {
			c.log.Error().Err(err).Msgf("failed to list CephFS %s subvolumegroup", cephFs.Name)
			issues++
			continue
		} else if subvolumegroup != "" {
			_, err = c.cephFSSubvolumegroupCommand("rm", cephFs.Name)
			if err != nil {
				c.log.Error().Err(err).Msgf("failed to remove CephFS %s subvolumegroup", cephFs.Name)
				issues++
				continue
			}
		}

		c.log.Info().Msgf("removing CephFS %s/%s", c.lcmConfig.RookNamespace, cephFs.Name)
		err = c.api.Rookclientset.CephV1().CephFilesystems(c.lcmConfig.RookNamespace).Delete(c.context, cephFs.Name, metav1.DeleteOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			c.log.Error().Err(err).Msgf("failed to remove CephFS %s/%s", c.lcmConfig.RookNamespace, cephFs.Name)
			issues++
		}
		delete(resourceUpdateTimestamps.cephConfigMap, fmt.Sprintf("mds.%s", cephFs.Name))
	}
	if issues > 0 {
		return false, errors.New("some CephFS failed to delete")
	}
	return false, nil
}

func generateCephFS(cephDplCephFS cephlcmv1alpha1.CephFS, namespace string, hyperconverge *cephlcmv1alpha1.CephDeploymentHyperConverge) *cephv1.CephFilesystem {
	label := cephNodeLabels["mds"]
	cephFS := &cephv1.CephFilesystem{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cephDplCephFS.Name,
			Namespace: namespace,
		},
	}
	cephFS.Spec.MetadataPool = cephv1.NamedPoolSpec{
		PoolSpec: *generatePoolSpec(&cephDplCephFS.MetadataPool, "mds metadata"),
	}
	cephFsDataPools := make([]cephv1.NamedPoolSpec, len(cephDplCephFS.DataPools))
	for idx, dataPool := range cephDplCephFS.DataPools {
		cephFsDataPools[idx] = cephv1.NamedPoolSpec{
			Name:     dataPool.Name,
			PoolSpec: *generatePoolSpec(&dataPool.CephPoolSpec, "mds data"),
		}
	}
	cephFS.Spec.DataPools = cephFsDataPools
	cephFS.Spec.PreserveFilesystemOnDelete = cephDplCephFS.PreserveFilesystemOnDelete
	cephFS.Spec.MetadataServer = cephv1.MetadataServerSpec{
		ActiveCount:   cephDplCephFS.MetadataServer.ActiveCount,
		ActiveStandby: cephDplCephFS.MetadataServer.ActiveStandby,
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
			PodAntiAffinity: &v1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "rook_file_system",
									Operator: "In",
									Values: []string{
										cephDplCephFS.Name,
									},
								},
							},
						},
						TopologyKey: "kubernetes.io/hostname",
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
	}
	if hyperconverge != nil {
		if v, ok := hyperconverge.Tolerations["mds"]; ok {
			cephFS.Spec.MetadataServer.Placement.Tolerations = append(cephFS.Spec.MetadataServer.Placement.Tolerations, v.Rules...)
		}
		if res, ok := hyperconverge.Resources["mds"]; ok {
			cephFS.Spec.MetadataServer.Resources = res
		}
	}
	if cephDplCephFS.MetadataServer.Resources != nil {
		cephFS.Spec.MetadataServer.Resources = *cephDplCephFS.MetadataServer.Resources
	}
	if cephDplCephFS.MetadataServer.HealthCheck != nil {
		cephFS.Spec.MetadataServer.LivenessProbe = cephDplCephFS.MetadataServer.HealthCheck.LivenessProbe
		cephFS.Spec.MetadataServer.StartupProbe = cephDplCephFS.MetadataServer.HealthCheck.StartupProbe
	}
	if cephFS.Spec.MetadataServer.LivenessProbe == nil {
		cephFS.Spec.MetadataServer.LivenessProbe = &cephv1.ProbeSpec{Probe: defaultCephProbe}
	}
	// if config is updated, need to restart mds daemons, since config may have some changes to cephfs
	cephFS.Spec.MetadataServer.Annotations = map[string]string{
		fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, "global"): resourceUpdateTimestamps.cephConfigMap["global"],
	}
	if resourceUpdateTimestamps.cephConfigMap["mds"] != "" {
		cephFS.Spec.MetadataServer.Annotations[fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, "mds")] = resourceUpdateTimestamps.cephConfigMap["mds"]
	}
	mdsDaemon := fmt.Sprintf("mds.%s", cephDplCephFS.Name)
	if resourceUpdateTimestamps.cephConfigMap[mdsDaemon] != "" {
		cephFS.Spec.MetadataServer.Annotations[fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, mdsDaemon)] = resourceUpdateTimestamps.cephConfigMap[mdsDaemon]
	}
	return cephFS
}
