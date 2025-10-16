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
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func IsCephToolboxCLIAvailable(ctx context.Context, kubeClient kubernetes.Interface, namespace string) bool {
	cephToolBox, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, PelagiaToolBox, metav1.GetOptions{})
	if err != nil {
		return false
	}
	return IsDeploymentReady(cephToolBox)
}

func RunCephToolboxCLI(ctx context.Context, kubeClient kubernetes.Interface, config *rest.Config, namespace, command string) (string, error) {
	e := ExecConfig{
		Context:    ctx,
		Kubeclient: kubeClient,
		Config:     config,
		Namespace:  namespace,
		Command:    command,
		Labels:     []string{fmt.Sprintf("app=%s", PelagiaToolBox)},
	}
	output, _, err := RunPodCmdAndCheckError(e)
	if err != nil {
		return output, err
	}
	return output, nil
}

func RunAndParseCephToolboxCLI(ctx context.Context, kubeClient kubernetes.Interface, config *rest.Config, namespace, command string, data any) error {
	output, err := RunCephToolboxCLI(ctx, kubeClient, config, namespace, command)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(output), data)
	if err != nil {
		return errors.Wrapf(err, "failed to parse output for command '%s'", command)
	}
	return nil
}

func RunAndParseDiskDaemonCLI(ctx context.Context, kubeClient kubernetes.Interface, config *rest.Config, namespace, nodeName, command string, data any) error {
	e := ExecConfig{
		Context:    ctx,
		Kubeclient: kubeClient,
		Config:     config,
		Namespace:  namespace,
		Command:    command,
		Nodename:   nodeName,
		Labels:     []string{fmt.Sprintf("app=%s", PelagiaDiskDaemon)},
	}
	output, _, err := RunPodCmdAndCheckError(e)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(output), data)
	if err != nil {
		return errors.Wrapf(err, "failed to parse output for command '%s'", command)
	}
	return nil
}
