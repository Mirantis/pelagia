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
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

var (
	// pure pod command run with exec config
	RunPodCommand = runPodCommand
	// validate exec config and run pod command
	RunPodCommandWithValidation = runPodCommandWithValidation
)

type ExecConfig struct {
	Context       context.Context
	Kubeclient    kubernetes.Interface
	Config        *rest.Config
	Pod           *corev1.Pod
	Namespace     string
	Command       string
	Content       []byte
	ContainerName string
	Nodename      string
	Labels        []string
}

func RunPodCmdAndCheckError(e ExecConfig) (string, string, error) {
	stdOut, stdErr, err := RunPodCommandWithValidation(e)
	if err != nil {
		msg := fmt.Sprintf("failed to run command '%s'", e.Command)
		if e.Nodename != "" {
			msg = fmt.Sprintf("%s on node '%s'", msg, e.Nodename)
		}
		if stdErr != "" {
			msg = fmt.Sprintf("%s (stdErr: %s)", msg, stdErr)
		}
		return stdOut, stdErr, errors.Wrap(err, msg)
	}
	return stdOut, stdErr, nil
}

func runPodCommandWithValidation(e ExecConfig) (string, string, error) {
	err := e.validate()
	if err != nil {
		return "", "", err
	}
	return RunPodCommand(e)
}

func (e *ExecConfig) validate() error {
	if e.Command == "" {
		return errors.New("command is not specified")
	}
	if e.Namespace == "" {
		e.Namespace = "default"
	}
	if e.Config == nil {
		config, err := rest.InClusterConfig()
		if err != nil {
			return errors.Wrap(err, "failed to get rest config")
		}
		config.Timeout = 90 * time.Second
		e.Config = config
	}
	if e.Pod == nil {
		pod, err := e.findPod()
		if err != nil {
			return errors.Wrap(err, "failed to find pod to run command")
		}
		e.Pod = pod
	}
	return nil
}

func (e *ExecConfig) findPod() (*corev1.Pod, error) {
	podSelector := "no labels specified"
	listOptions := metav1.ListOptions{}
	if len(e.Labels) > 0 {
		listOptions.LabelSelector = strings.Join(e.Labels, ",")
		podSelector = fmt.Sprintf("label(s): '%s'", listOptions.LabelSelector)
	}
	if e.Nodename != "" {
		listOptions.FieldSelector = fmt.Sprintf("spec.nodeName=%s", e.Nodename)
		podSelector = fmt.Sprintf("%s, field: '%s'", podSelector, listOptions.FieldSelector)
	}
	podExecList, err := e.Kubeclient.CoreV1().Pods(e.Namespace).List(e.Context, listOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pods list")
	}
	if len(podExecList.Items) == 0 {
		return nil, fmt.Errorf("no pods found matching criteria (%s) in namespace '%s'", podSelector, e.Namespace)
	}
	podFound := false
	podReady := false
	for idx, pod := range podExecList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			podFound = true
			for _, podCondition := range pod.Status.Conditions {
				if podCondition.Type == corev1.PodReady && podCondition.Status == corev1.ConditionTrue {
					podReady = true
					if e.ContainerName == "" && len(pod.Spec.Containers) > 0 {
						e.ContainerName = pod.Spec.Containers[0].Name
					}
					for _, containerStatus := range pod.Status.ContainerStatuses {
						if containerStatus.Name == e.ContainerName && containerStatus.Ready {
							return &podExecList.Items[idx], nil
						}
					}
				}
			}
		}
	}
	if !podFound || !podReady {
		return nil, fmt.Errorf("no running ready pod matching criteria (%s) in namespace '%s'", podSelector, e.Namespace)
	}
	return nil, fmt.Errorf("no ready container '%s' found for pod matching criteria (%s) in namespace '%s'", e.ContainerName, podSelector, e.Namespace)
}

func runPodCommand(e ExecConfig) (string, string, error) {
	req := e.Kubeclient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(e.Pod.Name).
		Namespace(e.Pod.Namespace).
		SubResource("exec")

	splitCmd := strings.Fields(e.Command)
	if strings.Contains(e.Command, "-c \"") || strings.Contains(e.Command, "-c '") {
		beforeCmd, afterCmd, _ := strings.Cut(e.Command, "\"")
		splitCmd = strings.Fields(beforeCmd)
		splitCmd = append(splitCmd, afterCmd[:len(afterCmd)-1])
	}
	var resultCmd []string
	// add connect timeout for ceph cli
	if splitCmd[0] == "ceph" {
		resultCmd = []string{"ceph", "--connect-timeout", fmt.Sprintf("%d", RunCephCommandTimeout)}
		resultCmd = append(resultCmd, splitCmd[1:]...)
	} else {
		resultCmd = splitCmd
	}
	req.VersionedParams(&corev1.PodExecOptions{
		Command:   resultCmd,
		Container: e.ContainerName,
		Stdin:     len(e.Content) > 0,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(e.Config, "POST", req.URL())
	if err != nil {
		return "", "", errors.Wrap(err, "error while creating executor")
	}

	var stdout, stderr bytes.Buffer
	streamOptions := remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	}
	if len(e.Content) > 0 {
		stdin := bytes.NewBuffer(e.Content)
		streamOptions.Stdin = stdin
	}
	err = exec.StreamWithContext(e.Context, streamOptions)
	if err != nil {
		err = errors.Wrap(err, "error while executing command")
	}
	return stdout.String(), stderr.String(), err
}
