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
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	v1storage "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

func (c *ManagedConfig) ListPods(namespace, label string) ([]corev1.Pod, error) {
	listOpts := metav1.ListOptions{}
	if label != "" {
		listOpts = metav1.ListOptions{LabelSelector: label}
	}
	pods, err := c.KubeClient.CoreV1().Pods(namespace).List(c.Context, listOpts)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list pods from %s namespace", namespace)
	}
	return pods.Items, nil
}

func (c *ManagedConfig) GetPodByLabel(namespace, label string) (*corev1.Pod, error) {
	pods, err := c.ListPods(namespace, label)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get pods in %s namespace with label %s", namespace, label)
	}
	if len(pods) == 0 {
		return nil, k8serrors.NewNotFound(schema.GroupResource{Resource: "pod", Group: "v1"}, "pod not found")
	}
	return &pods[0], nil
}

func (c *ManagedConfig) GetPodLogs(name string, namespace string) (string, error) {
	r := c.KubeClient.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{})
	podLogs, err := r.Stream(c.Context)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get pod's '%s/%s' logs", namespace, name)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	logText := buf.String()
	return logText, nil
}

func (c *ManagedConfig) CreatePod(pod *corev1.Pod) error {
	_, err := c.KubeClient.CoreV1().Pods(pod.Namespace).Create(c.Context, pod, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to create pod %s/%s", pod.Namespace, pod.Name)
	}
	return nil
}

func (c *ManagedConfig) DeletePod(name, namespace string) error {
	err := c.KubeClient.CoreV1().Pods(namespace).Delete(c.Context, name, metav1.DeleteOptions{})
	if k8serrors.IsNotFound(err) {
		return err
	} else if err != nil {
		return errors.Wrapf(err, "failed to delete pod %s/%s", namespace, name)
	}
	return nil
}

func (c *ManagedConfig) ListNodes() ([]corev1.Node, error) {
	nodes, err := c.KubeClient.CoreV1().Nodes().List(c.Context, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "unable to list nodes")
	}
	return nodes.Items, nil
}

func (c *ManagedConfig) UpdateNode(node *corev1.Node) error {
	_, err := c.KubeClient.CoreV1().Nodes().Update(c.Context, node, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to update %s node", node.Name)
	}
	return nil
}

func (c *ManagedConfig) GetService(name, namespace string) (*corev1.Service, error) {
	svc, err := c.KubeClient.CoreV1().Services(namespace).Get(c.Context, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get service %s/%s", namespace, name)
	}
	return svc, nil
}

func (c *ManagedConfig) GetSecret(name, namespace string) (*corev1.Secret, error) {
	secret, err := c.KubeClient.CoreV1().Secrets(namespace).Get(c.Context, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get secret %s/%s", namespace, name)
	}
	return secret, nil
}

func (c *ManagedConfig) CreateSecret(secret *corev1.Secret) error {
	_, err := c.KubeClient.CoreV1().Secrets(secret.Namespace).Create(c.Context, secret, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to create Secret %s/%s", secret.Namespace, secret.Name)
	}
	return nil
}

func (c *ManagedConfig) DeleteSecret(name, namespace string) error {
	err := c.KubeClient.CoreV1().Secrets(namespace).Delete(c.Context, name, metav1.DeleteOptions{})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to delete Secret %s/%s", namespace, name)
	}
	return nil
}

func (c *ManagedConfig) GetConfigMap(name, namespace string) (*corev1.ConfigMap, error) {
	cm, err := c.KubeClient.CoreV1().ConfigMaps(namespace).Get(c.Context, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get configmap %s/%s", namespace, name)
	}
	return cm, nil
}

func (c *ManagedConfig) CreateConfigMap(cm *corev1.ConfigMap) error {
	_, err := c.KubeClient.CoreV1().ConfigMaps(cm.Namespace).Create(c.Context, cm, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to create ConfigMap %s/%s", cm.Namespace, cm.Name)
	}
	return nil
}

func (c *ManagedConfig) DeleteConfigMap(name, namespace string) error {
	err := c.KubeClient.CoreV1().ConfigMaps(namespace).Delete(c.Context, name, metav1.DeleteOptions{})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to delete ConfigMap %s/%s", namespace, name)
	}
	return nil
}

func (c *ManagedConfig) GetIngress(name, namespace string) (*networkingv1.Ingress, error) {
	ingress, err := c.KubeClient.NetworkingV1().Ingresses(namespace).Get(c.Context, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get ingress %s/%s", namespace, name)
	}
	return ingress, nil
}

func (c *ManagedConfig) GetIngressClass(name string) (*networkingv1.IngressClass, error) {
	ingressClass, err := c.KubeClient.NetworkingV1().IngressClasses().Get(c.Context, name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get IngressClass %s", name)
	}
	return ingressClass, nil
}

func (c *ManagedConfig) GetStorageClass(name string) (*v1storage.StorageClass, error) {
	sc, err := c.KubeClient.StorageV1().StorageClasses().Get(c.Context, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get storageClass %s", name)
	}
	return sc, nil
}

func (c *ManagedConfig) ListStorageClass() (*v1storage.StorageClassList, error) {
	scList, err := c.KubeClient.StorageV1().StorageClasses().List(c.Context, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list storageClasses")
	}
	return scList, nil
}

func (c *ManagedConfig) CreatePVC(pvc *corev1.PersistentVolumeClaim) error {
	_, err := c.KubeClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(c.Context, pvc, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to create PVC %s/%s", pvc.Namespace, pvc.Name)
	}
	return nil
}

func (c *ManagedConfig) UpdatePVC(pvc *corev1.PersistentVolumeClaim) error {
	_, err := c.KubeClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(c.Context, pvc, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to update PVC %s/%s", pvc.Namespace, pvc.Name)
	}
	return nil
}

func (c *ManagedConfig) DeletePVC(name, namespace string) error {
	err := c.KubeClient.CoreV1().PersistentVolumeClaims(namespace).Delete(c.Context, name, metav1.DeleteOptions{})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to delete PVC %s/%s", namespace, name)
	}
	return nil
}

func (c *ManagedConfig) GetPVC(name, namespace string) (*corev1.PersistentVolumeClaim, error) {
	pvc, err := c.KubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(c.Context, name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, err
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get PVC %s/%s", namespace, name)
	}
	return pvc, nil
}

func (c *ManagedConfig) CreateDeployment(deploy *appsv1.Deployment) error {
	_, err := c.KubeClient.AppsV1().Deployments(deploy.Namespace).Create(c.Context, deploy, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to create Deployment %s/%s", deploy.Namespace, deploy.Name)
	}
	return nil
}

func (c *ManagedConfig) GetDeployment(name, namespace string) (*appsv1.Deployment, error) {
	deploy, err := c.KubeClient.AppsV1().Deployments(namespace).Get(c.Context, name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, err
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Deployment %s/%s", namespace, name)
	}
	return deploy, nil
}

func (c *ManagedConfig) IsDeploymentScaled(namespace, name string, replicas int32) (bool, error) {
	var deployment *appsv1.Deployment
	var err error
	if waitErr := wait.PollUntilContextTimeout(c.Context, 10*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		deployment, err = c.KubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsConflict(err) {
				return false, err
			}
			if !k8serrors.IsAlreadyExists(err) {
				TF.Log.Error().Err(err).Msg("retrying...")
				return false, nil
			}
		}
		if replicas == 0 {
			return deployment.Status.Replicas == replicas, nil
		}
		return deployment.Status.ReadyReplicas == replicas, nil
	}); waitErr != nil {
		return false, err
	}
	return true, nil
}

func (c *ManagedConfig) ListDeployments(namespace string, labels ...string) (*appsv1.DeploymentList, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: strings.Join(labels, ","),
	}
	deploys, err := c.KubeClient.AppsV1().Deployments(namespace).List(c.Context, listOptions)
	if k8serrors.IsNotFound(err) {
		return nil, err
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list Deployments in %s namespace", namespace)
	}
	return deploys, nil
}

func (c *ManagedConfig) DeleteDeployment(name, namespace string) error {
	err := c.KubeClient.AppsV1().Deployments(namespace).Delete(c.Context, name, metav1.DeleteOptions{})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to delete Deployment %s/%s", namespace, name)
	}
	return nil
}

func (c *ManagedConfig) ScaleDeployment(deployName, namespace string, replicas int32) error {
	return lcmcommon.ScaleDeployment(c.Context, c.KubeClient, deployName, namespace, replicas)
}

func (c *ManagedConfig) WaitDeploymentReady(deployName, deployNamespace string) error {
	TF.Log.Info().Msgf("Waiting deployment %s/%s readiness", deployNamespace, deployName)
	err := wait.PollUntilContextTimeout(c.Context, 15*time.Second, 30*time.Minute, true, func(_ context.Context) (bool, error) {
		deploy, err := c.GetDeployment(deployName, deployNamespace)
		if err != nil {
			return false, nil
		}
		return lcmcommon.IsDeploymentReady(deploy), nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to wait deployment %s/%s readiness", deployNamespace, deployName)
	}
	return nil
}

func (c *ManagedConfig) RunCommand(command, namespace string, labels ...string) (string, string, error) {
	e := lcmcommon.ExecConfig{
		Context:    c.Context,
		Kubeclient: c.KubeClient,
		Config:     c.KubeConfig,
		Namespace:  namespace,
		Command:    command,
		Labels:     labels,
	}
	return lcmcommon.RunPodCmdAndCheckError(e)
}

func (c *ManagedConfig) RunCephToolsCommand(command string) (string, error) {
	return lcmcommon.RunCephToolboxCLI(c.Context, c.KubeClient, c.KubeConfig, c.LcmConfig.RookNamespace, command)
}

func (c *ManagedConfig) RunPodCommand(command, containerName string, pod *corev1.Pod) (string, string, error) {
	return c.RunPodCommandWithContent(command, containerName, pod, "")
}

func (c *ManagedConfig) RunPodCommandWithContent(command, containerName string, pod *corev1.Pod, content string) (string, string, error) {
	if pod == nil {
		return "", "", errors.New("pod object is not provided, but required")
	}
	e := lcmcommon.ExecConfig{
		Context:       c.Context,
		Kubeclient:    c.KubeClient,
		Config:        c.KubeConfig,
		Command:       command,
		Namespace:     pod.Namespace,
		ContainerName: containerName,
		Pod:           pod,
	}
	if len(content) > 0 {
		e.Content = []byte(content)
	}
	return lcmcommon.RunPodCmdAndCheckError(e)
}

func (c *ManagedConfig) GetStatefulset(name, namespace string) (*appsv1.StatefulSet, error) {
	sts, err := c.KubeClient.AppsV1().StatefulSets(namespace).Get(c.Context, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %s/%s statefulset", namespace, name)
	}
	return sts, nil
}

func IsStatefulsetReady(sts *appsv1.StatefulSet) bool {
	return sts.Status.Replicas > 0 &&
		sts.Status.UpdatedReplicas == sts.Status.Replicas &&
		sts.Status.ReadyReplicas == sts.Status.Replicas &&
		sts.Status.CurrentReplicas == sts.Status.Replicas
}

func (c *ManagedConfig) CreateAWSCliDeployment(name string, label string, testImage string, awsConfig string, sslSecretName string, ingressIP string, endpoint string) (*appsv1.Deployment, error) {
	if label == "" {
		label = "awscli"
	}
	TF.Log.Info().Msgf("using image '%s' for awscli deployment '%s/%s", testImage, c.LcmConfig.RookNamespace, name)
	awscli := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.LcmConfig.RookNamespace,
			Labels: map[string]string{
				"app": label,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": label,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: c.LcmConfig.RookNamespace,
					Labels: map[string]string{
						"app": label,
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
					DNSPolicy: "ClusterFirstWithHostNet",
					Containers: []corev1.Container{
						{
							Name:  "awscli",
							Image: testImage,
							Command: []string{
								"/bin/sleep", "3650d",
							},
							ImagePullPolicy: "IfNotPresent",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "rgw-ssl-secret",
									MountPath: "/etc/rgwcerts",
								},
								{
									Name:      "rgw-user-creds",
									MountPath: "/root/.aws",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "rgw-ssl-secret",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: sslSecretName,
								},
							},
						},
						{
							Name: "rgw-user-creds",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: awsConfig},
								},
							},
						},
					},
				},
			},
		},
	}
	if ingressIP != "" && endpoint != "" {
		awscli.Spec.Template.Spec.HostAliases = []corev1.HostAlias{
			{
				IP:        ingressIP,
				Hostnames: []string{endpoint},
			},
		}
	}
	err := c.CreateDeployment(awscli)
	if err != nil {
		return nil, err
	}
	err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 15*time.Minute, true, func(_ context.Context) (bool, error) {
		TF.Log.Info().Msgf("waiting deployment '%s/%s' ready", awscli.Namespace, awscli.Name)
		deploy, err := c.GetDeployment(awscli.Name, awscli.Namespace)
		if err != nil {
			return false, nil
		}
		return lcmcommon.IsDeploymentReady(deploy), nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to wait Deployment %s/%s running", awscli.Namespace, awscli.Name)
	}

	return awscli, nil
}
