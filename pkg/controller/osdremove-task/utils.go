/*
Copyright 2025 The Mirantis Authors.

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

package osdremove

import (
	"fmt"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephOsdRemoveConfig) tryToGetNodeOsdsReportOrIssues(host string) (*lcmcommon.DiskDaemonOsdsReport, []string) {
	nodeReport, err := c.getNodeReport(host)
	if err != nil {
		return nil, []string{fmt.Sprintf("[node '%s'] failed to get node osds report: %v", host, err)}
	}
	if len(nodeReport.Issues) > 0 {
		issues := []string{}
		for _, issue := range nodeReport.Issues {
			issues = append(issues, fmt.Sprintf("[node '%s'] %s", host, issue))
		}
		return nil, issues
	}
	if nodeReport.OsdsReport == nil {
		return nil, []string{fmt.Sprintf("[node '%s'] node osds report is not available, check daemon logs on related node", host)}
	}
	return nodeReport.OsdsReport, nil
}

func (c *cephOsdRemoveConfig) getNodeReport(hostName string) (*lcmcommon.DiskDaemonReport, error) {
	cmd := fmt.Sprintf("%s --osd-report --port %d", lcmcommon.PelagiaDiskDaemon, c.lcmConfig.DiskDaemonPort)
	nodeReportRes, err := lcmcommon.RunFuncWithRetry(retriesForFailedCommand, commandRetryRunTimeout, func() (interface{}, error) {
		var report *lcmcommon.DiskDaemonReport
		daemonErr := lcmcommon.RunAndParseDiskDaemonCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.task.Namespace, hostName, cmd, &report)
		if daemonErr != nil {
			c.log.Error().Err(daemonErr).Msg("")
			return nil, daemonErr
		}
		if report.State == lcmcommon.DiskDaemonStateInProgress {
			c.log.Warn().Msgf("report from node '%s' is preparing, waiting...", hostName)
			return nil, errors.New("node report is not prepared yet")
		}
		return report, nil
	})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, err
	}
	return nodeReportRes.(*lcmcommon.DiskDaemonReport), nil
}

func (c *cephOsdRemoveConfig) getOsdHostsFromCluster() (map[string][]int, error) {
	var osdTree lcmcommon.OsdTree
	cmd := "ceph osd tree -f json"
	err := lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.cephCluster.Namespace, cmd, &osdTree)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return nil, err
	}
	hosts := map[string][]int{}
	for _, node := range osdTree.Nodes {
		if node.Type == "host" {
			hosts[node.Name] = node.Children
		}
	}
	return hosts, nil
}

func (c *cephOsdRemoveConfig) getOsdInfo(osdID string) (lcmcommon.OsdInfo, error) {
	var osdInfo lcmcommon.OsdInfo
	cmd := fmt.Sprintf("ceph osd info %s --format json", osdID)
	err := lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.cephCluster.Namespace, cmd, &osdInfo)
	if err != nil {
		c.log.Error().Err(err).Msg("")
	}
	return osdInfo, err
}

func (c *cephOsdRemoveConfig) checkPgsForOsd(osdID string) (bool, error) {
	var pgsByOsd map[string]interface{}
	cmd := fmt.Sprintf("ceph pg ls-by-osd %s --format json", osdID)
	err := lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.taskConfig.cephCluster.Namespace, cmd, &pgsByOsd)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return false, err
	}
	pgStats := pgsByOsd["pg_stats"]
	if pgStats != nil && len(pgStats.([]interface{})) > 0 {
		return true, nil
	}
	return false, nil
}

func (c *cephOsdRemoveConfig) checkOperatorStopped() bool {
	deploy, err := c.api.Kubeclientset.AppsV1().Deployments(c.taskConfig.cephCluster.Namespace).Get(c.context, lcmcommon.RookCephOperatorName, metav1.GetOptions{})
	if err != nil {
		err = errors.Wrapf(err, "failed to check rook-operator '%s/%s' is stopped", c.taskConfig.cephCluster.Namespace, lcmcommon.RookCephOperatorName)
		c.log.Error().Err(err).Msg("")
		return false
	}
	return deploy.Status.Replicas == 0 && deploy.Status.ReadyReplicas == 0 && deploy.Status.AvailableReplicas == 0
}

func (c *cephOsdRemoveConfig) removeOsdDeployment(deployName string) error {
	err := c.api.Kubeclientset.AppsV1().Deployments(c.taskConfig.cephCluster.Namespace).Delete(c.context, deployName, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		c.log.Error().Err(err).Msg("")
		return errors.Wrapf(err, "failed to delete osd deployment '%s'", deployName)
	}
	return nil
}
