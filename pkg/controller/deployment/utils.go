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
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	rookclient "github.com/rook/rook/pkg/client/clientset/versioned"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

const blockPoolServiceMarker = "builtin"

func buildRGWName(name, suffix string) string {
	if suffix == "" {
		return fmt.Sprintf("rook-ceph-rgw-%s", name)
	}
	return fmt.Sprintf("rook-ceph-rgw-%s-%s", name, suffix)
}

func rgwSyncDaemonName(rgwName string) string {
	return fmt.Sprintf("%s-sync", rgwName)
}

func isKubeCrush(key string) bool {
	return key == "region" || key == "zone"
}

func isSpecIngressProxyRequired(cephSpec cephlcmv1alpha1.CephDeploymentSpec) bool {
	// do not create ingress if no custom class name specified
	// with default class (openstack-nginx-proxy), but without OpenStack we don't need an Ingress proxy
	if cephSpec.IngressConfig != nil {
		if cephSpec.IngressConfig.ControllerClassName != "" {
			return true
		}
	}
	if lcmcommon.IsOpenStackPoolsPresent(cephSpec.Pools) {
		return true
	}
	return false
}

func isTypeReadyToUpdate(condition cephv1.ConditionType) bool {
	notReadyTypes := []cephv1.ConditionType{
		cephv1.ConditionConnecting,
		cephv1.ConditionProgressing,
		cephv1.ConditionDeleting,
	}
	for _, notReady := range notReadyTypes {
		if condition == notReady {
			return false
		}
	}
	return true
}

func getCephPoolName(pool cephlcmv1alpha1.CephPool) string {
	if pool.UseAsFullName {
		return pool.Name
	}
	return fmt.Sprintf("%s-%s", pool.Name, pool.DeviceClass)
}

func getBuiltinPoolName(name string) string {
	if strings.HasPrefix(name, ".") {
		return fmt.Sprintf("%s%s", blockPoolServiceMarker, strings.ReplaceAll(name, ".", "-"))
	}
	if strings.Contains(name, "_") {
		return fmt.Sprintf("%s-%s", blockPoolServiceMarker, strings.ReplaceAll(name, "_", "-"))
	}
	return name
}

func buildPoolName(pool cephlcmv1alpha1.CephPool) string {
	cephBlockPoolName := getCephPoolName(pool)
	// PRODX-37192 - allow to create pools names starting with '.' (.rgw.root, .mgr)
	if lcmcommon.Contains(builtinCephPools, cephBlockPoolName) {
		cephBlockPoolName = getBuiltinPoolName(cephBlockPoolName)
	}
	return cephBlockPoolName
}

func isCephDeployed(ctx context.Context, log zerolog.Logger, kubeclient kubernetes.Interface, namespace string) bool {
	_, err := kubeclient.CoreV1().ConfigMaps(namespace).Get(ctx, rookCephMonEndpointsMapName, metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("failed to get ConfigMap %s/%s", namespace, rookCephMonEndpointsMapName)
		return false
	}
	return true
}

func isStateReadyToUpdate(state cephv1.ClusterState) bool {
	notReadyStates := []cephv1.ClusterState{
		cephv1.ClusterStateCreating,
		cephv1.ClusterStateConnecting,
		cephv1.ClusterStateUpdating,
	}
	for _, notReady := range notReadyStates {
		if state == notReady {
			return false
		}
	}
	return true
}

func buildCephNodeAnnotations(current map[string]string, desired map[string]string) (map[string]string, bool) {
	newAnnotations := map[string]string{}
	for k, v := range current {
		newAnnotations[k] = v
	}
	changed := false
	for _, annotation := range cephNodeAnnotationKeys {
		if _, ok := desired[annotation]; !ok {
			if _, ok := newAnnotations[annotation]; ok {
				delete(newAnnotations, annotation)
				changed = true
			}
		} else if newAnnotations[annotation] != desired[annotation] {
			newAnnotations[annotation] = desired[annotation]
			changed = true
		}
	}
	return newAnnotations, changed
}

func isCephPoolReady(ctx context.Context, log zerolog.Logger, client rookclient.Interface, namespace string, poolName string) bool {
	pool, err := client.CephV1().CephBlockPools(namespace).Get(ctx, poolName, metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("failed to get %s/%s pool", namespace, poolName)
		return false
	}
	if pool.Status != nil {
		return pool.Status.Phase == cephv1.ConditionReady
	}
	return false
}

func isCephFsReady(ctx context.Context, log zerolog.Logger, client rookclient.Interface, namespace string, cephFsName string) bool {
	cephFs, err := client.CephV1().CephFilesystems(namespace).Get(ctx, cephFsName, metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("failed to get %s/%s cephFs", namespace, cephFsName)
		return false
	}
	if cephFs.Status != nil {
		return cephFs.Status.Phase == cephv1.ConditionReady
	}
	return false
}

func validateDeviceClassName(deviceClass string, extraOpts *cephlcmv1alpha1.CephDeploymentExtraOpts) error {
	customDeviceClasses := make([]string, 0)
	if extraOpts != nil && len(extraOpts.CustomDeviceClasses) > 0 {
		customDeviceClasses = extraOpts.CustomDeviceClasses
	}
	validNames := append([]string{"hdd", "nvme", "ssd"}, customDeviceClasses...)
	if deviceClass == "" {
		return fmt.Errorf("no deviceClass specified (valid options are: %v)", validNames)
	}
	for _, className := range validNames {
		if className == deviceClass {
			return nil
		}
	}
	return fmt.Errorf("unknown deviceClass '%s' (valid options are: %v)", deviceClass, validNames)
}

// CLI utils

type cephConfigOptionDump struct {
	Section string `json:"section"`
	Name    string `json:"name"`
	Value   string `json:"value"`
}

func (c *cephDeploymentConfig) getCephConfigDump() ([]cephConfigOptionDump, error) {
	var cephConfigDump []cephConfigOptionDump
	err := lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, "ceph config dump --format json", &cephConfigDump)
	if err != nil {
		return nil, err
	}
	return cephConfigDump, nil
}

func (c *cephDeploymentConfig) getCephVersions() (*lcmcommon.CephVersions, error) {
	var cephVersions lcmcommon.CephVersions
	err := lcmcommon.RunAndParseCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, "ceph versions --format json", &cephVersions)
	if err != nil {
		return nil, err
	}
	return &cephVersions, nil
}

func (c *cephDeploymentConfig) cephFSSubvolumegroupCommand(op string, cephFsName string) (string, error) {
	command := fmt.Sprintf("ceph fs subvolumegroup -f json %s %s", op, cephFsName)
	if op == "create" || op == "rm" {
		command = command + " " + subVolumeGroupName
	}
	output, err := lcmcommon.RunCephToolboxCLI(c.context, c.api.Kubeclientset, c.api.Config, c.lcmConfig.RookNamespace, command)
	if err != nil {
		return "", err
	}
	if op == "ls" {
		var subvolumegroup []map[string]string
		err = json.Unmarshal([]byte(output), &subvolumegroup)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse 'ceph fs subvolumegroup' command output")
		}
		if len(subvolumegroup) == 0 {
			return "", nil
		}
		return subvolumegroup[0]["name"], nil
	}
	return "", nil
}
