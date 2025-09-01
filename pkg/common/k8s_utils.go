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

package lcmcommon

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func IsDaemonSetReady(ds *appsv1.DaemonSet) bool {
	return ds.Status.NumberReady > 0 &&
		ds.Status.CurrentNumberScheduled == ds.Status.DesiredNumberScheduled &&
		ds.Status.DesiredNumberScheduled == ds.Status.NumberReady &&
		ds.Status.NumberReady == ds.Status.NumberAvailable &&
		ds.Status.NumberAvailable == ds.Status.UpdatedNumberScheduled
}

func IsDeploymentReady(deploy *appsv1.Deployment) bool {
	return deploy.Status.Replicas > 0 &&
		deploy.Status.UpdatedReplicas == deploy.Status.Replicas &&
		deploy.Status.ReadyReplicas == deploy.Status.Replicas &&
		deploy.Status.AvailableReplicas == deploy.Status.Replicas
}

func GetObjectOwnerRef(obj runtime.Object, scheme *runtime.Scheme) ([]metav1.OwnerReference, error) {
	// See https://github.com/kubernetes/kubernetes/issues/3030 - APIVersion and Kind fields may not be populated
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get GVK for object")
	}
	objMeta, err := meta.Accessor(obj)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get meta.Interface for object")
	}

	return []metav1.OwnerReference{
		{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
			Name:       objMeta.GetName(),
			UID:        objMeta.GetUID(),
		},
	}, nil
}

func GetNodeList(ctx context.Context, kubeClient kubernetes.Interface, listOptions metav1.ListOptions) (*corev1.NodeList, error) {
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func GetNode(ctx context.Context, kubeClient kubernetes.Interface, nodename string) (*corev1.Node, error) {
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodename, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return node, nil
}

func IsNodeAvailable(node corev1.Node) (bool, string) {
	if len(node.Spec.Taints) > 0 {
		for _, taint := range node.Spec.Taints {
			switch taint.Key {
			case corev1.TaintNodeUnreachable, corev1.TaintNodeUnschedulable, corev1.TaintNodeNotReady, corev1.TaintNodeOutOfService:
				return false, fmt.Sprintf("node '%s' has '%s' taint, assuming node is not available", node.Name, taint.Key)
			}
		}
	}
	return true, ""
}

func ScaleDeployment(ctx context.Context, kubeClient kubernetes.Interface, deployName, namespace string, replicas int32) error {
	scale := &autoscalingv1.Scale{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      deployName,
		},
		Spec: autoscalingv1.ScaleSpec{
			Replicas: replicas,
		},
	}
	_, err := kubeClient.AppsV1().Deployments(namespace).UpdateScale(ctx, deployName, scale, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func RestartDeployment(ctx context.Context, log zerolog.Logger, kubeclientset kubernetes.Interface, deploymentName, namespace string) error {
	log.Info().Msgf("restarting pods from %s/%s deployment", namespace, deploymentName)
	deployment, err := kubeclientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get deployment %s/%s", namespace, deploymentName)
	}
	if deployment.Spec.Paused {
		errMsg := fmt.Sprintf("can't restart paused deployment %s/%s (run rollout resume first)", namespace, deploymentName)
		return errors.New(errMsg)
	}
	if deployment.Status.Replicas == 0 {
		log.Warn().Msgf("can't restart deployment which has no replicas %s/%s", namespace, deploymentName)
		return nil
	}
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations[DeploymentRestartAnnotation] = GetCurrentTimeString()
	_, err = kubeclientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to restart deployment %s/%s", namespace, deploymentName)
	}
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 3*time.Minute, true, func(_ context.Context) (bool, error) {
		updatedDeployment, err := kubeclientset.AppsV1().Deployments(namespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
		if err != nil {
			log.Error().Err(err).Msgf("failed to get deployment %s/%s", namespace, deploymentName)
			return false, nil
		}
		// do not continue until generation changes and replicas started updating
		if updatedDeployment.Status.ObservedGeneration == deployment.Status.ObservedGeneration ||
			updatedDeployment.Status.Replicas == 0 {
			return false, nil
		}
		return IsDeploymentReady(updatedDeployment), nil
	})
	return err
}
