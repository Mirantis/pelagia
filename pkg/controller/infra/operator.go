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

package infra

import (
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *cephDeploymentInfraConfig) checkRookOperatorReplicas() error {
	desiredReplicas := int32(1)
	rookOperator, err := c.api.Kubeclientset.AppsV1().Deployments(c.lcmConfig.RookNamespace).Get(c.context, lcmcommon.RookCephOperatorName, metav1.GetOptions{})
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return errors.Wrapf(err, "failed to check replicas for rook operator '%s/%s'", c.lcmConfig.RookNamespace, lcmcommon.RookCephOperatorName)
	}
	currentReplicas := int32(1)
	if rookOperator.Spec.Replicas != nil {
		currentReplicas = *rookOperator.Spec.Replicas
	}
	scaleReason := "detected stopped rook operator and no reason to have it stopped"
	maintenance, err := lcmcommon.IsClusterMaintenanceActing(c.context, c.api.Lcmclientset, c.infraConfig.namespace, c.infraConfig.name)
	if err != nil {
		c.log.Error().Err(err).Msg("")
		return errors.Wrap(err, "failed to check CephDeploymentMaintenance state")
	}
	if maintenance {
		scaleReason = "set maintenance mode"
		desiredReplicas = int32(0)
	} else if !c.infraConfig.externalCeph {
		taskList, err := c.api.Lcmclientset.LcmV1alpha1().CephOsdRemoveTasks(c.infraConfig.namespace).List(c.context, metav1.ListOptions{})
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return errors.Wrap(err, "failed to check CephOsdRemoveTasks")
		}
		if len(taskList.Items) > 0 {
			downScale := false
			for _, task := range taskList.Items {
				if task.Status != nil {
					if task.Status.Phase == lcmv1alpha1.TaskPhaseWaitingOperator || task.Status.Phase == lcmv1alpha1.TaskPhaseProcessing {
						scaleReason = fmt.Sprintf("found CephOsdRemoveTask in phase '%s'", task.Status.Phase)
						downScale = true
						break
					}
					if task.Status.Phase == lcmv1alpha1.TaskPhaseFailed && (task.Spec == nil || !task.Spec.Resolved) {
						scaleReason = fmt.Sprintf("found CephOsdRemoveTask in phase '%s' and it does not have resolve flag", task.Status.Phase)
						downScale = true
						break
					}
				}
			}
			if downScale {
				desiredReplicas = int32(0)
			}
		}
	}
	if currentReplicas != desiredReplicas {
		c.log.Info().Msgf("scaling rook operator to %d replicas, since %s", desiredReplicas, scaleReason)
		err := lcmcommon.ScaleDeployment(c.context, c.api.Kubeclientset, lcmcommon.RookCephOperatorName, c.lcmConfig.RookNamespace, desiredReplicas)
		if err != nil {
			c.log.Error().Err(err).Msg("")
			return errors.Wrapf(err, "failed to scale rook operator '%s/%s'", c.lcmConfig.RookNamespace, lcmcommon.RookCephOperatorName)
		}
	}
	return nil
}
