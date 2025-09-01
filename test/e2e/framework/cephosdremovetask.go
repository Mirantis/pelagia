/*
Copyright 2025 Mirantis IT.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless taskuired by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	cephlcmv1alpha "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

func (c *ManagedConfig) CreateOsdRemoveTask(task *cephlcmv1alpha.CephOsdRemoveTask) error {
	_, err := c.CephOsdRemoveTaskClient.Create(c.Context, task, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to create CephOsdRemoveTask %s/%s", task.Namespace, task.Name)
	}
	return nil
}

func (c *ManagedConfig) UpdateOsdRemoveTask(task *cephlcmv1alpha.CephOsdRemoveTask) error {
	_, err := c.CephOsdRemoveTaskClient.Update(c.Context, task, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to update CephOsdRemoveTask %s/%s", task.Namespace, task.Name)
	}
	return nil
}

func (c *ManagedConfig) GetOsdRemoveTask(taskName, taskNamespace string) (*cephlcmv1alpha.CephOsdRemoveTask, error) {
	task, err := c.CephOsdRemoveTaskClient.Get(c.Context, taskName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get CephOsdRemoveTask %s/%s", taskNamespace, taskName)
	}
	return task, nil
}

func (c *ManagedConfig) CreateAndWaitOsdRemoveTask(task *cephlcmv1alpha.CephOsdRemoveTask) error {
	err := c.CreateOsdRemoveTask(task)
	if err != nil {
		return err
	}
	return c.WaitCephOsdRemoveTaskFinished(task)
}

func (c *ManagedConfig) WaitCephOsdRemoveTaskFinished(task *cephlcmv1alpha.CephOsdRemoveTask) error {
	taskFailed := false
	var taskStatus *cephlcmv1alpha.CephOsdRemoveTaskStatus
	err := wait.PollUntilContextTimeout(c.Context, 10*time.Second, 15*time.Minute, true, func(_ context.Context) (bool, error) {
		task, err := c.GetOsdRemoveTask(task.Name, task.Namespace)
		if err != nil {
			TF.Log.Error().Err(err).Msg("")
			return false, err
		}
		if task.Status == nil {
			TF.Log.Warn().Msgf("Waiting CephOsdRemoveTask %s/%s status updated", task.Namespace, task.Name)
			return false, nil
		}
		taskStatus = task.Status
		if task.Status.Phase == cephlcmv1alpha.TaskPhaseApproveWaiting {
			TF.Log.Warn().Msgf("Found CephOsdRemoveTask %s/%s in %v phase, approving...", task.Namespace, task.Name, task.Status.Phase)
			task.Spec.Approve = true
			err := c.UpdateOsdRemoveTask(task)
			if err != nil {
				TF.Log.Error().Err(err).Msg("")
			}
			return false, err
		}
		if task.Status.Phase != cephlcmv1alpha.TaskPhaseCompleted && task.Status.Phase != cephlcmv1alpha.TaskPhaseFailed &&
			task.Status.Phase != cephlcmv1alpha.TaskPhaseCompletedWithWarnings {
			TF.Log.Warn().Msgf("Waiting CephOsdRemoveTask %s/%s finished, current phase: %v", task.Namespace, task.Name, task.Status.Phase)
			return false, nil
		}
		if task.Status.Phase == cephlcmv1alpha.TaskPhaseFailed {
			taskFailed = true
		}
		if len(task.Status.RemoveInfo.CleanupMap) == 0 {
			TF.Log.Warn().Msgf("CephOsdRemoveTask %s/%s cleanup map is empty", task.Namespace, task.Name)
			return false, errors.New("empty cleanup map")
		}
		return true, nil
	})
	strStatus, statusErr := yaml.Marshal(taskStatus)
	if statusErr != nil {
		TF.Log.Error().Err(statusErr).Msg("failed to marshal osd remove task status")
	} else {
		TF.Log.Info().Msgf("status for CephOsdRemoveTask %s/%s:\n%s\n", task.Namespace, task.Name, strStatus)
	}
	if err != nil {
		return err
	}
	if taskFailed {
		return fmt.Errorf("CephOsdRemoveTask %s/%s failed", task.Namespace, task.Name)
	}
	return nil
}

func (c *ManagedConfig) DeleteOsdRemoveTask(name string) error {
	err := c.CephOsdRemoveTaskClient.Delete(c.Context, name, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to delete CephOsdRemoveTask %s", name)
	}
	return nil
}
