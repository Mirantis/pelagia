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
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

func (c *cephDeploymentConfig) ensureCluster() (bool, error) {
	c.log.Info().Msg("ensure ceph cluster")
	var err error
	changed := false
	// Get current cephCluster
	cephClusterFound := true
	cephCluster, err := c.api.Rookclientset.CephV1().CephClusters(c.lcmConfig.RookNamespace).Get(c.context, c.cdConfig.cephDpl.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			cephClusterFound = false
		} else {
			return false, errors.Wrapf(err, "failed to get %s/%s cephcluster", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
		}
	}
	// prepare resources required for external case and pass cephcluster for owner refs
	if c.cdConfig.cephDpl.Spec.External {
		var ownerRefs []metav1.OwnerReference
		if cephClusterFound {
			ownerRefs, err = lcmcommon.GetObjectOwnerRef(cephCluster, c.api.Scheme)
			if err != nil {
				return false, errors.Wrapf(err, "failed to prepare ownerRefs for CephCluster '%s/%s' related external resources", cephCluster.Namespace, cephCluster.Name)
			}
		}
		changed, err = c.addExternalResources(ownerRefs)
		if err != nil {
			c.log.Error().Err(err).Msg("failed to ensure external cluster configuration")
			return false, errors.Wrapf(err, "unable to create external cluster configuration")
		}
	}

	// in case of migration, we could not have mon deployed yet but cephcluster could be
	// already here so we should check rook-ceph-mon-endpoint cm existence separately
	cephDeployed := isCephDeployed(c.context, *c.log, c.api.Kubeclientset, c.lcmConfig.RookNamespace)

	if !c.cdConfig.cephDpl.Spec.External {
		configChanged, err := c.ensureCephConfig(cephClusterFound && cephDeployed)
		if err != nil {
			return false, errors.Wrapf(err, "failed to ensure ceph config for %s/%s cephcluster", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
		}
		changed = changed || configChanged
	}

	// If ceph cluster is not ready/created/healthy - skip ensure
	if err = c.statusCluster(); err != nil {
		return false, errors.Wrapf(err, "failed to ensure cephcluster %s/%s", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
	}

	// Generate new ceph cluster spec
	generatedClusterSpec := generateCephClusterSpec(c.cdConfig.cephDpl, c.cdConfig.currentCephImage, c.cdConfig.nodesListExpanded)

	// Create/Update/Skip ceph cluster
	if !cephClusterFound {
		newCluster := &cephv1.CephCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.cdConfig.cephDpl.Name,
				Namespace: c.lcmConfig.RookNamespace,
			},
			Spec: generatedClusterSpec,
		}
		c.log.Info().Msgf("creating cephcluster %s/%s", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
		_, err := c.api.Rookclientset.CephV1().CephClusters(c.lcmConfig.RookNamespace).Create(c.context, newCluster, metav1.CreateOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to create cephcluster %s/%s", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
		}
		return true, nil
	}

	// since osd params usually updated at runtime, no restart is required,
	// but if operator specified manually reason for restart - set it
	// keep annotations in cephcluster to control changes
	if c.cdConfig.cephDpl.Spec.ExtraOpts != nil && c.cdConfig.cephDpl.Spec.ExtraOpts.OsdRestartReason != "" {
		if cephCluster.Annotations == nil {
			cephCluster.Annotations = map[string]string{}
		}
		if cephCluster.Annotations[cephRestartOsdLabel] != c.cdConfig.cephDpl.Spec.ExtraOpts.OsdRestartReason {
			cephCluster.Annotations[cephRestartOsdLabel] = c.cdConfig.cephDpl.Spec.ExtraOpts.OsdRestartReason
			cephCluster.Annotations[cephRestartOsdTimestampLabel] = lcmcommon.GetCurrentTimeString()
		}
	}
	if _, ok := cephCluster.Annotations[cephRestartOsdLabel]; ok {
		generatedClusterSpec.Annotations[cephv1.KeyOSD] = map[string]string{
			cephRestartOsdLabel:          cephCluster.Annotations[cephRestartOsdLabel],
			cephRestartOsdTimestampLabel: cephCluster.Annotations[cephRestartOsdTimestampLabel],
		}
	}

	if !reflect.DeepEqual(cephCluster.Spec, generatedClusterSpec) {
		lcmcommon.ShowObjectDiff(*c.log, cephCluster.Spec, generatedClusterSpec)
		c.log.Info().Msgf("updating cephcluster %s/%s", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
		cephCluster.Spec = generatedClusterSpec
		_, err := c.api.Rookclientset.CephV1().CephClusters(c.lcmConfig.RookNamespace).Update(c.context, cephCluster, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to update cephcluster %s/%s", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
		}
		changed = true
	}

	return changed, nil
}

func (c *cephDeploymentConfig) deleteCluster() (bool, error) {
	cluster, err := c.api.Rookclientset.CephV1().CephClusters(c.lcmConfig.RookNamespace).Get(c.context, c.cdConfig.cephDpl.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// reset whole timestamps var
			unsetTimestampsVar()
			return true, nil
		}
		return false, errors.Wrap(err, "failed to get ceph cluster")
	}
	cluster.Spec.CleanupPolicy.Confirmation = "yes-really-destroy-data"
	cluster.Spec.CleanupPolicy.AllowUninstallWithVolumes = true
	c.log.Info().Msgf("setting Ceph cluster %s/%s CleanupPolicy before remove", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
	_, err = c.api.Rookclientset.CephV1().CephClusters(c.lcmConfig.RookNamespace).Update(c.context, cluster, metav1.UpdateOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to update ceph cluster with cleanupPolicy")
	}
	c.log.Info().Msgf("removing Ceph cluster %s/%s", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
	err = c.api.Rookclientset.CephV1().CephClusters(c.lcmConfig.RookNamespace).Delete(c.context, cluster.Name, metav1.DeleteOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to delete ceph cluster")
	}
	return false, nil
}

func (c *cephDeploymentConfig) statusCluster() error {
	cluster, err := c.api.Rookclientset.CephV1().CephClusters(c.lcmConfig.RookNamespace).Get(c.context, c.cdConfig.cephDpl.Name, metav1.GetOptions{})
	if err != nil {
		c.log.Error().Err(err).Msgf("failed to get ceph cluster %s/%s", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
		return nil
	}
	clusterStatus := &cluster.Status
	if clusterStatus == nil || clusterStatus.CephStatus == nil {
		c.log.Error().Err(err).Msgf("Ceph cluster %s/%s has no cluster status", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
		return nil
	}
	isHealthy := c.healthCluster(clusterStatus.CephStatus)
	isStateOk := isStateReadyToUpdate(clusterStatus.State)
	isPhaseOk := isTypeReadyToUpdate(clusterStatus.Phase)

	// Don't check cluster health - update it anyway
	if !isStateOk || !isPhaseOk {
		msg := fmt.Sprintf("ceph cluster %s/%s is not ready to be updated: "+
			"cluster healthy = %v, "+
			"cluster state = '%v', "+
			"cluster phase = '%v'", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name, isHealthy, clusterStatus.State, clusterStatus.Phase,
		)
		return errors.New(msg)
	}
	return nil
}

func (c *cephDeploymentConfig) healthCluster(cephStatus *cephv1.CephStatus) bool {
	clusterhealth, _ := json.Marshal(cephStatus.Health)
	healthdetails := cephStatus.Details

	if string(clusterhealth) == "\"HEALTH_OK\"" {
		c.log.Info().Msgf("Cluster health: %v", string(clusterhealth))
		return true
	}
	if string(clusterhealth) == "\"HEALTH_WARN\"" {
		c.log.Warn().Msgf("Cluster health: %v", string(clusterhealth))
	} else if string(clusterhealth) == "\"HEALTH_ERR\"" {
		c.log.Error().Msgf("Cluster health: %v", string(clusterhealth))
	}
	c.log.Info().Msgf("allowed issues: %v", strings.Join(cephIgnoredHealthWarnings, " "))
	doNotIgnoreIssues := []string{}
	for key, message := range healthdetails {
		if lcmcommon.Contains(cephIgnoredHealthWarnings, key) {
			c.log.Info().Msgf("found issue %s: %s", key, message)
		} else {
			c.log.Warn().Msgf("found issue %s: %s", key, message)
			doNotIgnoreIssues = append(doNotIgnoreIssues, key)
		}
	}
	if len(doNotIgnoreIssues) > 0 {
		c.log.Warn().Msgf("found issues, which can't be ignored: %s", strings.Join(doNotIgnoreIssues, " "))
		return false
	}
	return true
}

func generateCephClusterSpec(cephDpl *cephlcmv1alpha1.CephDeployment, image string, nodesExpanded []cephlcmv1alpha1.CephDeploymentNode) cephv1.ClusterSpec {
	clusterSpec := cephv1.ClusterSpec{}
	clusterSpec.CephVersion.Image = image

	if len(nodesExpanded) <= 3 {
		// We need to WA upgrade checks due to Ceph Octopus upgrade failure
		// the corresponding issue in rook: https://github.com/rook/rook/issues/5337
		// this is a common issue and could not be fixed
		clusterSpec.SkipUpgradeChecks = true
		clusterSpec.ContinueUpgradeAfterChecksEvenIfNotHealthy = true
	}

	if cephDpl.Spec.DataDirHostPath == "" {
		clusterSpec.DataDirHostPath = lcmcommon.DefaultDataDirHostPath
	} else {
		clusterSpec.DataDirHostPath = cephDpl.Spec.DataDirHostPath
	}

	if cephDpl.Spec.External {
		clusterSpec.External.Enable = true
		return clusterSpec
	}

	// if config map changed - mark cluster spec annotations for mon and mgr to restart
	// also control global value, since for that all daemons will be restarted, except osd
	clusterSpec.Annotations = map[cephv1.KeyType]cephv1.Annotations{
		cephv1.KeyMon: map[string]string{
			fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, "global"): resourceUpdateTimestamps.cephConfigMap["global"],
		},
		cephv1.KeyMgr: map[string]string{
			fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, "global"): resourceUpdateTimestamps.cephConfigMap["global"],
		},
	}
	if resourceUpdateTimestamps.cephConfigMap["mon"] != "" {
		clusterSpec.Annotations[cephv1.KeyMon][fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, "mon")] = resourceUpdateTimestamps.cephConfigMap["mon"]
	}
	if resourceUpdateTimestamps.cephConfigMap["mgr"] != "" {
		clusterSpec.Annotations[cephv1.KeyMgr][fmt.Sprintf(cephConfigParametersUpdateTimestampLabel, "mgr")] = resourceUpdateTimestamps.cephConfigMap["mgr"]
	}

	clusterSpec.Mon.Count = 0
	clusterSpec.Mgr.Count = 0
	for _, node := range nodesExpanded {
		if lcmcommon.Contains(node.Roles, "mon") {
			clusterSpec.Mon.Count = clusterSpec.Mon.Count + 1
		}
		// we have to limit mgr count by two, since rook can't have
		// more than two Mgrs simultaneously
		if lcmcommon.Contains(node.Roles, "mgr") && clusterSpec.Mgr.Count < 2 {
			clusterSpec.Mgr.Count = clusterSpec.Mgr.Count + 1
		}
	}

	clusterSpec.Dashboard.Enabled = cephDpl.Spec.DashboardEnabled

	defaultModules := []string{"balancer", "pg_autoscaler"}
	defaultModulesFound := map[string]bool{
		"pg_autoscaler": false,
		"balancer":      false,
	}

	if cephDpl.Spec.Mgr != nil && cephDpl.Spec.Mgr.MgrModules != nil {
		for _, module := range cephDpl.Spec.Mgr.MgrModules {
			cephModule := cephv1.Module{
				Name:    module.Name,
				Enabled: module.Enabled,
			}
			if module.Settings != nil {
				cephModule.Settings = cephv1.ModuleSettings{BalancerMode: module.Settings.BalancerMode}
			}
			clusterSpec.Mgr.Modules = append(clusterSpec.Mgr.Modules, cephModule)
			if _, ok := defaultModulesFound[module.Name]; ok {
				defaultModulesFound[module.Name] = true
			}
		}
	}
	for _, module := range defaultModules {
		if !defaultModulesFound[module] {
			clusterSpec.Mgr.Modules = append(clusterSpec.Mgr.Modules, cephv1.Module{
				Name:    module,
				Enabled: true,
			})
		}
	}

	useAllDevices := false
	clusterSpec.Storage = cephv1.StorageScopeSpec{
		UseAllNodes: false,
		Selection: cephv1.Selection{
			UseAllDevices: &useAllDevices,
		},
	}

	clusterSpec.Storage.Nodes = buildStorageNodes(nodesExpanded)
	switch cephDpl.Spec.Network.Provider {
	case "", "host":
		clusterSpec.Network.Provider = "host"
		if cephDpl.Spec.Network.MonOnPublicNet {
			publics := strings.Split(cephDpl.Spec.Network.PublicNet, ",")
			publicCIDRs := cephv1.CIDRList{}
			for _, pubNet := range publics {
				publicCIDRs = append(publicCIDRs, cephv1.CIDR(pubNet))
			}

			clusters := strings.Split(cephDpl.Spec.Network.ClusterNet, ",")
			clusterCIDRs := cephv1.CIDRList{}
			for _, clusterNet := range clusters {
				clusterCIDRs = append(clusterCIDRs, cephv1.CIDR(clusterNet))
			}

			clusterSpec.Network.AddressRanges = &cephv1.AddressRangesSpec{
				Public:  publicCIDRs,
				Cluster: clusterCIDRs,
			}
		}
	case "multus":
		clusterSpec.Network.Provider = "multus"
		clusterSpec.Network.Selectors = cephDpl.Spec.Network.Selector
	}

	if cephDpl.Spec.HyperConverge == nil {
		cephDpl.Spec.HyperConverge = &cephlcmv1alpha1.CephDeploymentHyperConverge{}
	}

	clusterSpec.Placement = cephv1.PlacementSpec{
		cephv1.KeyMon: cephv1.Placement{
			NodeAffinity: &v1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
					NodeSelectorTerms: []v1.NodeSelectorTerm{
						{
							MatchExpressions: []v1.NodeSelectorRequirement{
								{
									Key:      cephNodeLabels["mon"],
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
			PodAffinity:     &v1.PodAffinity{},
			PodAntiAffinity: &v1.PodAntiAffinity{},
			Tolerations: append([]v1.Toleration{
				{
					Key:      cephNodeLabels["mon"],
					Operator: "Exists",
				},
			},
				cephDpl.Spec.HyperConverge.Tolerations[string(cephv1.KeyMon)].Rules...),
		},
		cephv1.KeyMgr: cephv1.Placement{
			NodeAffinity: &v1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
					NodeSelectorTerms: []v1.NodeSelectorTerm{
						{
							MatchExpressions: []v1.NodeSelectorRequirement{
								{
									Key:      cephNodeLabels["mgr"],
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
			PodAffinity:     &v1.PodAffinity{},
			PodAntiAffinity: &v1.PodAntiAffinity{},
			Tolerations: append([]v1.Toleration{
				{
					Key:      cephNodeLabels["mgr"],
					Operator: "Exists",
				},
			},
				cephDpl.Spec.HyperConverge.Tolerations[string(cephv1.KeyMgr)].Rules...),
		},
	}
	if v, ok := cephDpl.Spec.HyperConverge.Tolerations[string(cephv1.KeyAll)]; ok {
		clusterSpec.Placement[cephv1.KeyAll] = cephv1.Placement{
			Tolerations: v.Rules,
		}
	}
	osdTolerations := cephDpl.Spec.HyperConverge.Tolerations[string(cephv1.KeyOSD)].Rules
	if len(osdTolerations) > 0 {
		clusterSpec.Placement[cephv1.KeyOSD] = cephv1.Placement{
			Tolerations: osdTolerations,
		}
		// We have to set same tolerations which specified for osd, because
		// it could not be neither prepared nor cleaned otherwise.
		clusterSpec.Placement[cephv1.KeyOSDPrepare] = cephv1.Placement{
			Tolerations: osdTolerations,
		}
		clusterSpec.Placement[cephv1.KeyCleanup] = cephv1.Placement{
			Tolerations: osdTolerations,
		}
	}

	if len(cephDpl.Spec.HyperConverge.Resources) > 0 {
		clusterSpec.Resources = cephDpl.Spec.HyperConverge.Resources
	}

	if cephDpl.Spec.HealthCheck != nil {
		clusterSpec.HealthCheck = cephv1.CephClusterHealthCheckSpec{
			DaemonHealth:  cephDpl.Spec.HealthCheck.DaemonHealth,
			LivenessProbe: cephDpl.Spec.HealthCheck.LivenessProbe,
			StartupProbe:  cephDpl.Spec.HealthCheck.StartupProbe,
		}
	}
	cephDaemons := []cephv1.KeyType{"mgr", "mon", "osd"}
	if clusterSpec.HealthCheck.LivenessProbe == nil {
		clusterSpec.HealthCheck.LivenessProbe = map[cephv1.KeyType]*cephv1.ProbeSpec{}
	}
	for _, daemon := range cephDaemons {
		if _, ok := clusterSpec.HealthCheck.LivenessProbe[daemon]; !ok {
			clusterSpec.HealthCheck.LivenessProbe[daemon] = &cephv1.ProbeSpec{
				Probe: defaultCephProbe,
			}
		}
	}

	return clusterSpec
}

func (c *cephDeploymentConfig) checkStorageSpecIsAligned() (bool, error) {
	cluster, err := c.api.Rookclientset.CephV1().CephClusters(c.lcmConfig.RookNamespace).Get(c.context, c.cdConfig.cephDpl.Name, metav1.GetOptions{})
	if err != nil {
		c.log.Error().Err(err).Msgf("failed to get ceph cluster %s/%s", c.lcmConfig.RookNamespace, c.cdConfig.cephDpl.Name)
		return false, err
	}
	newNodes := buildStorageNodes(c.cdConfig.nodesListExpanded)
	equal := reflect.DeepEqual(cluster.Spec.Storage.Nodes, newNodes)
	if !equal {
		lcmcommon.ShowObjectDiff(*c.log, cluster.Spec.Storage.Nodes, newNodes)
	}
	return equal, nil
}

func buildStorageNodes(cephDplNodes []cephlcmv1alpha1.CephDeploymentNode) []cephv1.Node {
	// CephCluster sorts nodes entries lexicographically by node's name
	nodeNames := make([]string, 0)
	nodeMap := map[string]int{}
	for idx, node := range cephDplNodes {
		if isCephOsdNode(node.Node) {
			nodeNames = append(nodeNames, node.Name)
			nodeMap[node.Name] = idx
		}
	}
	sort.Strings(nodeNames)
	nodes := []cephv1.Node{}
	for _, nodeName := range nodeNames {
		node := cephDplNodes[nodeMap[nodeName]]
		newNode := node.Node
		if len(node.Devices) > 0 {
			devices := []cephv1.Device{}
			for _, dev := range node.Devices {
				// by-id is put into name, not full path field
				if strings.HasPrefix(dev.FullPath, "/dev/disk/by-id") {
					devices = append(devices, cephv1.Device{
						Name:   dev.FullPath,
						Config: dev.Config,
					})
				} else {
					devices = append(devices, cephv1.Device{
						Name:     dev.Name,
						FullPath: dev.FullPath,
						Config:   dev.Config,
					})
				}
			}
			if len(devices) > 0 {
				newNode.Devices = devices
			}
		}
		nodes = append(nodes, newNode)
	}
	return nodes
}
