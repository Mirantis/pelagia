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
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentHealthConfig) cephDeploymentVerification() (*lcmv1alpha1.CephDeploymentHealthReport, []string) {
	healthIssues := []string{}
	newHealthReport := &lcmv1alpha1.CephDeploymentHealthReport{
		RookOperator: c.checkRookOperator(),
	}
	if len(newHealthReport.RookOperator.Issues) > 0 {
		healthIssues = append(healthIssues, newHealthReport.RookOperator.Issues...)
	}

	rookObjectsReport, rookObjectsIssues := c.rookObjectsVerification()
	if len(rookObjectsIssues) > 0 {
		healthIssues = append(healthIssues, rookObjectsIssues...)
	}
	newHealthReport.RookCephObjects = rookObjectsReport

	// if cephstatus is not present, no need to check everything else
	if rookObjectsReport == nil {
		return newHealthReport, healthIssues
	}

	daemonsStatus, daemonsIssues := c.daemonsStatusVerification()
	newHealthReport.CephDaemons = daemonsStatus
	if len(daemonsIssues) > 0 {
		healthIssues = append(healthIssues, daemonsIssues...)
	}

	clusterDetailsInfo, detailsIssues := c.getClusterDetailsInfo()
	newHealthReport.ClusterDetails = clusterDetailsInfo
	if len(detailsIssues) > 0 {
		healthIssues = append(healthIssues, detailsIssues...)
	}

	specAnalysisStatus, specIssues := c.getSpecAnalysisStatus()
	newHealthReport.OsdAnalysis = specAnalysisStatus
	if len(specIssues) > 0 {
		healthIssues = append(healthIssues, specIssues...)
	}

	sort.Strings(healthIssues)
	return newHealthReport, healthIssues
}

func (c *cephDeploymentHealthConfig) checkRookOperator() lcmv1alpha1.DaemonStatus {
	rookOperatorStatus := lcmv1alpha1.DaemonStatus{}
	deployment, err := c.api.Kubeclientset.AppsV1().Deployments(c.lcmConfig.RookNamespace).Get(c.context, "rook-ceph-operator", metav1.GetOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		rookOperatorStatus.Issues = []string{fmt.Sprintf("failed to get 'rook-ceph-operator' deployment in '%s' namespace", c.lcmConfig.RookNamespace)}
		rookOperatorStatus.Status = lcmv1alpha1.DaemonStateFailed
		return rookOperatorStatus
	}
	if lcmcommon.IsDeploymentReady(deployment) {
		rookOperatorStatus.Status = lcmv1alpha1.DaemonStateOk
	} else {
		rookOperatorStatus.Status = lcmv1alpha1.DaemonStateFailed
		maintenance, err := lcmcommon.IsClusterMaintenanceActing(c.context, c.api.Lcmclientset, c.healthConfig.namespace, c.healthConfig.name)
		if err != nil {
			c.log.Error().Err(err).Msg("")
			rookOperatorStatus.Issues = []string{"failed to check CephDeploymentMaintenance state"}
			return rookOperatorStatus
		}
		if maintenance {
			rookOperatorStatus.Messages = []string{"deployment 'rook-ceph-operator' is scaled down due to maintenance mode"}
		} else {
			rookOperatorStatus.Issues = []string{"deployment 'rook-ceph-operator' is not ready"}
		}
	}
	return rookOperatorStatus
}
