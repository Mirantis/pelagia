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

package framework

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const sshContainerName = "ssh-pod"

func (c *ManagedConfig) NewSSHPod(name string, privKeySecretName string) (*corev1.Pod, error) {
	privileged := true
	runAsUser := int64(0)
	privKeyMode := int32(0600)
	sshPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.LcmNamespace,
			Labels: map[string]string{
				"app": "ssh-pod",
			},
		},
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "node-role.kubernetes.io/master",
										Operator: corev1.NodeSelectorOpDoesNotExist,
									},
								},
							},
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name: sshContainerName,
					// TODO: make it configurable from current go runtime version
					Image: "docker-dev-kaas-local.docker.mirantis.net/mirantis/ceph/golang:1.21.13",
					Command: []string{
						"/bin/sleep", "3650d",
					},
					ImagePullPolicy: "IfNotPresent",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "ssh-priv-key",
							MountPath: "/priv-key",
						},
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
						RunAsUser:  &runAsUser,
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "ssh-priv-key",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  privKeySecretName,
							DefaultMode: &privKeyMode,
						},
					},
				},
			},
		},
	}

	err := c.CreatePod(sshPod)
	if err != nil {
		return nil, err
	}
	var pod *corev1.Pod
	err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 5*time.Minute, true, func(_ context.Context) (bool, error) {
		pod, err = c.GetPodByLabel(c.LcmNamespace, "app=ssh-pod")
		if err != nil {
			return false, nil
		}
		return pod.Status.Phase == corev1.PodRunning, nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to wait pod %s/%s running", c.LcmNamespace, name)
	}
	return pod, nil
}

func (c *ManagedConfig) ExecSSHPod(ip, cmd string, sshPod *corev1.Pod) (string, error) {
	command := fmt.Sprintf(
		"ssh"+
			" -o LogLevel=ERROR -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"+
			" -i /priv-key/key ubuntu@%s sudo %v", ip, cmd)
	stdout, stderr, err := c.RunPodCommand(command, sshContainerName, sshPod)
	if err != nil {
		errMsg := fmt.Sprintf("failed to run command '%v' on server IP %s", cmd, ip)
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %v)", stderr)
		}
		return "", errors.Wrap(err, errMsg)
	}
	return stdout, nil
}

func (c *ManagedConfig) UpdateFileSSHPod(ip, path, content string, sshPod *corev1.Pod) error {
	command := fmt.Sprintf(
		"ssh"+
			" -o LogLevel=ERROR -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"+
			" -i /priv-key/key ubuntu@%s sudo tee %s", ip, path)
	_, stderr, err := c.RunPodCommandWithContent(command, sshContainerName, sshPod, content)
	if err != nil {
		errMsg := fmt.Sprintf("failed to update file on path %s with content '%v' on server IP %s", path, content, ip)
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %v)", stderr)
		}
		return errors.Wrap(err, errMsg)
	}
	return nil
}

func (c *ManagedConfig) DeleteSSHPod(name string) error {
	err := c.DeletePod(name, c.LcmNamespace)
	if err != nil {
		return errors.Wrapf(err, "failed to delete SSH pod %s/%s: %v", TF.ManagedCluster.LcmNamespace, name, err)
	}
	return nil
}
