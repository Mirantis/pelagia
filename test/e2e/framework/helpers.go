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
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/pkg/errors"
	rookv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

func WaitForStatusReady(name string) error {
	TF.Log.Info().Msg("waiting for cluster status become Ready")
	err := TF.ManagedCluster.WaitForCephDeploymentReady(name)
	if err != nil {
		return errors.Wrap(err, "failed to wait for cephdeployment readiness")
	}
	err = TF.ManagedCluster.WaitForCephDeploymentHealthReady(name)
	if err != nil {
		return errors.Wrap(err, "failed to wait for cephdeploymenthealth readiness")
	}
	return nil
}

func UpdateCephDeploymentSpec(cdUpdated *cephlcmv1alpha.CephDeployment, waitReadiness bool) error {
	updated, err := TF.ManagedCluster.UpdateCephDeploymentSpec(cdUpdated)
	if err != nil {
		return errors.Wrap(err, "failed to update cephdeployment")
	}
	if updated && waitReadiness {
		err := WaitForStatusReady(cdUpdated.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func ReadSharedFilesystemConfig(sharedFilesystemConfigString string) ([]string, *cephlcmv1alpha.CephSharedFilesystem, error) {
	storageClasses := []string{}
	var sharedFS cephlcmv1alpha.CephSharedFilesystem
	err := json.Unmarshal([]byte(sharedFilesystemConfigString), &sharedFS)
	if err != nil {
		return nil, nil, err
	}
	for _, cephFS := range sharedFS.CephFS {
		storageClasses = append(storageClasses, fmt.Sprintf("%s-%s", cephFS.Name, cephFS.DataPools[0].Name))
	}
	return storageClasses, &sharedFS, nil
}

func ReadNodeStorageConfig(nodesConfig string) (map[string]rookv1.Node, error) {
	var cephNodesUpdate map[string]rookv1.Node
	err := yaml.Unmarshal([]byte(nodesConfig), &cephNodesUpdate)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal nodes config")
	}
	return cephNodesUpdate, nil
}

func GenerateOsdRemoveTasks(tasksSpecString string) ([]*cephlcmv1alpha.CephOsdRemoveTask, error) {
	if tasksSpecString == "" {
		return nil, nil
	}
	curTime := time.Now().UnixNano()
	var tasksConfig []cephlcmv1alpha.CephOsdRemoveTaskSpec
	err := yaml.Unmarshal([]byte(tasksSpecString), &tasksConfig)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal tasks config")
	}
	tasksList := make([]*cephlcmv1alpha.CephOsdRemoveTask, 0)
	for idx, taskSpec := range tasksConfig {
		task := &cephlcmv1alpha.CephOsdRemoveTask{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("ceph-osd-remove-task-%d-%d", idx+1, curTime),
				Namespace: TF.ManagedCluster.LcmNamespace,
			},
			Spec: &taskSpec,
		}
		tasksList = append(tasksList, task)
		strReq, err := yaml.Marshal(task.Spec)
		if err != nil {
			TF.Log.Error().Err(err).Msg("### failed to marshal task nodes")
		} else {
			TF.Log.Info().Msgf("### task (%s/%s) spec:\n%s\n", task.Namespace, task.Name, strReq)
		}
	}
	return tasksList, nil
}

func GetNewPool(name string, useAsfullName, volumeExpansion bool, size int, role, mapOptions string, deviceClass string) cephlcmv1alpha.CephPool {
	if role == "" {
		role = "e2e-tests"
	}
	return cephlcmv1alpha.CephPool{
		Name: name,
		Role: role,
		StorageClassOpts: cephlcmv1alpha.CephStorageClassSpec{
			Default:              false,
			MapOptions:           mapOptions,
			AllowVolumeExpansion: volumeExpansion,
		},
		CephPoolSpec: cephlcmv1alpha.CephPoolSpec{
			Replicated: &cephlcmv1alpha.CephPoolReplicatedSpec{
				Size: uint(size),
			},
			DeviceClass: deviceClass,
		},
		UseAsFullName: useAsfullName,
	}
}

func GetSecretForRgwCreds(cdSecret, userName string) (*corev1.Secret, error) {
	secretName := ""
	secretNamespace := ""
	cd, err := TF.ManagedCluster.GetCephDeploymentSecret(cdSecret)
	if err != nil {
		return nil, errors.Errorf("failed to get cephdeploymentsecret: %v", err)
	}
	for _, scinfo := range cd.Status.SecretsInfo.RgwUserSecrets {
		if scinfo.ObjectName == userName {
			secretNamespace = scinfo.SecretNamespace
			secretName = scinfo.SecretName
			break
		}
	}
	if secretName == "" || secretNamespace == "" {
		return nil, errors.Errorf("failed to identify secret for rgw user '%s' (empty ref)", userName)
	}
	return TF.ManagedCluster.GetSecret(secretName, secretNamespace)
}

func GetRgwPublicEndpoint(cdhName string) (string, error) {
	cdh, err := TF.ManagedCluster.GetCephDeploymentHealth(cdhName)
	if err != nil {
		return "", err
	}
	if cdh.Status.HealthReport == nil || cdh.Status.HealthReport.ClusterDetails == nil || cdh.Status.HealthReport.ClusterDetails.RgwInfo == nil {
		return "", errors.New("empty RgwInfo status")
	}
	return cdh.Status.HealthReport.ClusterDetails.RgwInfo.PublicEndpoint, nil
}

func GetDefaultPoolDeviceClass(cd *cephlcmv1alpha.CephDeployment) string {
	poolDefaultClass := ""
	for _, pool := range cd.Spec.Pools {
		if pool.StorageClassOpts.Default {
			poolDefaultClass = pool.DeviceClass
			break
		}
	}
	return poolDefaultClass
}
