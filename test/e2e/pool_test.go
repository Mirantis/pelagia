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

package test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

func TestAddRemovePool(t *testing.T) {
	t.Logf("e2e test: add new pool.")
	defer f.SetupTeardown(t)()
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	poolDefaultClass := f.GetDefaultPoolDeviceClass(cd)
	if poolDefaultClass == "" {
		t.Fatal("failed to find default pool")
	}

	f.Step(t, "Adding new Ceph Pool")
	poolName := "test-pool-new-" + fmt.Sprintf("%d", time.Now().Unix())
	newPool := f.GetNewPool(poolName, false, false, 2, "", "", poolDefaultClass)
	cd.Spec.Pools = append(cd.Spec.Pools, newPool)
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Validate a new pool")
	pool, err := f.TF.ManagedCluster.GetCephBlockPool(fmt.Sprintf("%s-%s", poolName, poolDefaultClass))
	if err != nil {
		t.Fatal(err)
	}
	err = cephv1.ValidateCephBlockPool(pool)
	assert.Nil(t, err)

	cd, err = f.TF.ManagedCluster.GetCephDeployment(cd.Name)
	if err != nil {
		t.Fatal(err)
	}
	f.Step(t, "Removing newly added pool")
	for idx, pool := range cd.Spec.Pools {
		if pool.Name == poolName {
			cd.Spec.Pools = append(cd.Spec.Pools[:idx], cd.Spec.Pools[idx+1:]...)
			break
		}
	}
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Validate if the pool was removed")
	_, err = f.TF.ManagedCluster.GetCephBlockPool(fmt.Sprintf("%s-%s", poolName, poolDefaultClass))
	if err == nil {
		t.Fatal("Pool still exists")
	} else {
		t.Logf("Pool not found: %v", err)
	}
	t.Logf("Test %s complete sucessfully", t.Name())
}

func TestVolumesBackendPool(t *testing.T) {
	t.Logf("e2e test: add new pool with volumes-backend role and verify it is available from openstack")
	defer f.SetupTeardown(t)()

	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	if !lcmcommon.IsOpenStackPoolsPresent(cd.Spec.Pools) {
		t.Skip("There are no openstack pools therefore could not proceed the test")
	}
	poolDefaultClass := f.GetDefaultPoolDeviceClass(cd)
	if poolDefaultClass == "" {
		t.Fatal("failed to find default pool")
	}

	f.Step(t, "Build volumes-backend role pool spec")
	name := "test-volumes-backend-" + fmt.Sprintf("%d", time.Now().Unix())
	newPool := f.GetNewPool(name, true, false, 2, "volumes", "", poolDefaultClass)

	cd.Spec.Pools = append(cd.Spec.Pools, newPool)
	f.Step(t, "Create Ceph Pool with volumes-backend role")
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Validate volumes-backend pool created")
	pool, err := f.TF.ManagedCluster.GetCephBlockPool(name)
	if err != nil {
		t.Fatal(err)
	}
	err = cephv1.ValidateCephBlockPool(pool)
	assert.Nil(t, err)

	f.Step(t, "Wait for Ceph Clients updated with volumes-backend pool")
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 3*time.Minute, true, func(_ context.Context) (bool, error) {
		cinderClient, err := f.TF.ManagedCluster.GetCephClient("cinder")
		if err != nil {
			f.TF.Log.Error().Err(err).Msg("")
			return false, nil
		}
		novaClient, err := f.TF.ManagedCluster.GetCephClient("nova")
		if err != nil {
			f.TF.Log.Error().Err(err).Msg("")
			return false, nil
		}
		found := false
		for _, caps := range cinderClient.Spec.Caps {
			if strings.Contains(caps, name) {
				found = true
			}
		}
		if !found {
			f.TF.Log.Error().Msgf("Pool %s capabilities not found on cinder CephClient, waiting", name)
			return false, nil
		}
		found = false
		for _, caps := range novaClient.Spec.Caps {
			if strings.Contains(caps, name) {
				found = true
			}
		}
		if !found {
			f.TF.Log.Error().Msgf("Pool %s capabilities not found on nova CephClient, waiting", name)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait for nova/cinder CephClient updated with pool %s capabilities", name)
	}

	f.Step(t, "Verify openstack-ceph-keys secret updated with volumes-backend pool")
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 3*time.Minute, true, func(_ context.Context) (bool, error) {
		ock, err := f.TF.ManagedCluster.GetSecret("openstack-ceph-keys", f.TF.ManagedCluster.LcmConfig.DeployParams.OpenstackCephSharedNamespace)
		if err != nil {
			t.Fatalf("Failed to get %s/openstack-ceph-keys secret: %v", f.TF.ManagedCluster.LcmConfig.DeployParams.OpenstackCephSharedNamespace, err)
		}
		found := strings.Contains(string(ock.Data["cinder"]), name) && strings.Contains(string(ock.Data["nova"]), name)
		return found, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait %s/openstack-ceph-keys secret updated", f.TF.ManagedCluster.LcmConfig.DeployParams.OpenstackCephSharedNamespace)
	}

	f.Step(t, "Wait for cinder-volume statefulset updated")
	cinderVolumeSts, err := f.TF.ManagedCluster.GetStatefulset("cinder-volume", "openstack")
	if err != nil {
		t.Fatalf("failed to get openstack/cinder-volume statefulset: %v", err)
	}
	curGeneration := cinderVolumeSts.Generation
	newGeneration := cinderVolumeSts.Generation
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 15*time.Second, 15*time.Minute, true, func(_ context.Context) (done bool, err error) {
		sts, err := f.TF.ManagedCluster.GetStatefulset("cinder-volume", "openstack")
		if err != nil {
			f.TF.Log.Error().Msgf("failed to get openstack/cinder-volume statefulset to get generation: %v", err)
			return false, nil
		}
		newGeneration = sts.Generation
		return sts.Generation > curGeneration, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait for openstack/cinder-volume statefulset updated (generation: expected=%v, actual=%v): %v", curGeneration, newGeneration, err)
	}

	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 15*time.Second, 20*time.Minute, true, func(_ context.Context) (done bool, err error) {
		sts, err := f.TF.ManagedCluster.GetStatefulset("cinder-volume", "openstack")
		if err != nil {
			f.TF.Log.Error().Msgf("failed to get openstack/cinder-volume statefulset to verify readiness: %v", err)
			return false, nil
		}
		f.TF.Log.Info().Msgf("openstack/cinder-volume statefulset %v replica readiness: current=%v, updated=%v, ready=%v",
			sts.Status.Replicas, sts.Status.CurrentReplicas, sts.Status.UpdatedReplicas, sts.Status.ReadyReplicas,
		)
		return f.IsStatefulsetReady(sts), nil
	})
	if err != nil {
		t.Fatalf("Failed to wait for openstack/cinder-volume statefulset readiness: %v", err)
	}

	f.Step(t, "Wait for %s pool is accessible from cinder-volume pods", name)
	cinderVolumePods, err := f.TF.ManagedCluster.ListPods("openstack", "application=cinder,component=volume")
	if err != nil {
		t.Fatalf("Failed to list cinder-volume pods: %v", err)
	}
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 15*time.Second, 30*time.Minute, true, func(_ context.Context) (done bool, err error) {
		for _, pod := range cinderVolumePods {
			command := fmt.Sprintf("rbd -n client.cinder ls -p %s", name)
			_, _, err := f.TF.ManagedCluster.RunPodCommand(command, pod.Spec.Containers[0].Name, pod.DeepCopy())
			if err != nil {
				f.TF.Log.Error().Err(err).Msgf("Failed to access pool data with client.cinder from %s pod", pod.Name)
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait cinder-volume pods have an access to %s pool: %v", name, err)
	}

	f.Step(t, "Obtain keystone-client pod with openstack CLI")
	keystoneClientPod, err := f.TF.ManagedCluster.GetPodByLabel("openstack", "application=keystone,component=client")
	if err != nil {
		t.Fatalf("Failed to get keystone-client pod: %v", err)
	}
	keystoneClientContainer := keystoneClientPod.Spec.Containers[0].Name

	f.Step(t, "Wait for volumes-backend pool appears as a cinder volume type")
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 15*time.Second, 15*time.Minute, true, func(_ context.Context) (done bool, err error) {
		command := "openstack volume type list -f value"
		stdout, _, err := f.TF.ManagedCluster.RunPodCommand(command, keystoneClientContainer, keystoneClientPod)
		if err != nil {
			f.TF.Log.Error().Err(err).Msgf("Failed to create cinder volume on %s pool", name)
			return false, nil
		}
		return strings.Contains(stdout, name), nil
	})
	if err != nil {
		t.Fatalf("failed to wait for volumes-backend pool becomes a cinder volume type: %v", err)
	}

	f.Step(t, "Find keystone pod")
	err = f.TF.ManagedCluster.OpenstackClientSet()
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Create cinder volume on %s pool", name)
	cinderVolumeName := fmt.Sprintf("backend-volume-%d", time.Now().Unix())
	_, err = f.TF.ManagedCluster.CinderVolumeCreate(cinderVolumeName, 1, name, true)
	if err != nil {
		t.Fatalf("failed to create volume: %v", err)
	}

	f.Step(t, "Remove cinder volume on %s pool", name)
	err = f.TF.ManagedCluster.CinderVolumeDelete(cinderVolumeName, true)
	if err != nil {
		t.Fatalf("failed to wait for cinder volume on %s pool removed: %v", name, err)
	}

	t.Logf("Test %s complete sucessfully", t.Name())
}

func TestVolumeExpansionPool(t *testing.T) {
	t.Log("#### e2e test: create volume expansion pool and expand size for corresponding pv")

	// specific defer for this test
	volExpPVCName := fmt.Sprintf("volume-expansion-pvc-%d", time.Now().Unix())
	volExpDeployName := fmt.Sprintf("volume-expansion-pods-%d", time.Now().Unix())
	f.SetupTeardown(t)()
	defer deferVolumeExpansionTest(volExpPVCName, volExpDeployName, t)()

	f.Step(t, "Build volume expansion pool spec")
	name := "test-volumes-expansion-" + fmt.Sprintf("%d", time.Now().Unix())
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	poolDefaultClass := f.GetDefaultPoolDeviceClass(cd)
	if poolDefaultClass == "" {
		t.Fatal("failed to find default pool")
	}
	newPool := f.GetNewPool(name, true, true, 2, "", "", poolDefaultClass)
	cd.Spec.Pools = append(cd.Spec.Pools, newPool)
	f.Step(t, "Create new pool %s", name)
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Validate pool %s exists", name)
	pool, err := f.TF.ManagedCluster.GetCephBlockPool(name)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Validate pool's %s storage class supports volume expansion", name)
	var poolSc *storagev1.StorageClass
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 5*time.Minute, true, func(_ context.Context) (bool, error) {
		poolSc, err = f.TF.ManagedCluster.GetStorageClass(name)
		if err != nil {
			t.Logf("storageclass %s still not available: %v", name, err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("failed to wait for storageclass %s: %v", name, err)
	}

	assert.True(t, *poolSc.AllowVolumeExpansion)
	assert.Equal(t, "rook-csi-rbd-provisioner", poolSc.Parameters["csi.storage.k8s.io/controller-expand-secret-name"])
	assert.Equal(t, pool.Namespace, poolSc.Parameters["csi.storage.k8s.io/controller-expand-secret-namespace"])
	assert.Equal(t, "ext4", poolSc.Parameters["csi.storage.k8s.io/fstype"])

	f.Step(t, "Build PVC with %s storageClass", name)
	q, err := resource.ParseQuantity("10Gi")
	if err != nil {
		t.Fatal(err)
	}

	volExpPVC := v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      volExpPVCName,
			Namespace: "default",
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.VolumeResourceRequirements{
				Requests: map[v1.ResourceName]resource.Quantity{
					v1.ResourceStorage: q,
				},
			},
			StorageClassName: &name,
		},
	}

	f.Step(t, "Create PVC %s/%s with %s storageClass", volExpPVC.Namespace, volExpPVC.Name, name)
	err = f.TF.ManagedCluster.CreatePVC(&volExpPVC)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		f.Step(t, "Clean up test PVC")
		err = f.TF.ManagedCluster.DeletePVC(volExpPVC.Name, volExpPVC.Namespace)
		if err != nil {
			t.Fatal(err)
		}
	}()

	f.Step(t, "Validate PVC %s/%s exists and valid", volExpPVC.Namespace, volExpPVC.Name)
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 3*time.Minute, true, func(_ context.Context) (bool, error) {
		pvc, err := f.TF.ManagedCluster.GetPVC(volExpPVC.Name, volExpPVC.Namespace)
		if err != nil {
			t.Logf("Retrying due to fail to get PVC %s/%s: %v", volExpPVC.Namespace, volExpPVC.Name, err)
			return false, nil
		}
		assert.Equal(t, name, *pvc.Spec.StorageClassName)
		return true, nil
	})
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to wait PVC %s/%s created", volExpPVC.Namespace, volExpPVC.Name))
	}

	f.Step(t, "Get test deployment image from Rook Ceph Operator")
	rco, err := f.TF.ManagedCluster.GetDeployment("rook-ceph-operator", f.TF.ManagedCluster.LcmConfig.RookNamespace)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get deployment %s/rook-ceph-operator", f.TF.ManagedCluster.LcmConfig.RookNamespace))
	}
	image := rco.Spec.Template.Spec.Containers[0].Image

	f.Step(t, "Build Deployment with volume expansion PVC bound")
	volExpDeploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      volExpDeployName,
			Namespace: "default",
			Labels: map[string]string{
				"app": "vol-exp-deploy",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "vol-exp-deploy",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      volExpDeployName,
					Namespace: "default",
					Labels: map[string]string{
						"app": "vol-exp-deploy",
					},
				},
				Spec: v1.PodSpec{
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "vol_exp_test",
												Operator: "In",
												Values: []string{
													"true",
												},
											},
										},
									},
								},
							},
						},
					},
					DNSPolicy: "ClusterFirstWithHostNet",
					Containers: []v1.Container{
						{
							Name:  "vol-exp-container",
							Image: image,
							Command: []string{
								"/bin/sleep", "3650d",
							},
							ImagePullPolicy: "IfNotPresent",
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "vol-exp-disk",
									MountPath: "/mnt/disk0",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "vol-exp-disk",
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: volExpPVCName},
							},
						},
					},
				},
			},
		},
	}

	f.Step(t, "Create Deployment %s/%s with volume expansion PVC bound", volExpDeploy.Namespace, volExpDeploy.Name)
	nodes, err := f.TF.ManagedCluster.ListNodes()
	if err != nil {
		t.Fatalf("failed to list nodes to label for test deployment: %v", err)
	}
	for _, node := range nodes {
		if len(node.Spec.Taints) == 0 {
			node.Labels["vol_exp_test"] = "true"
			err = f.TF.ManagedCluster.UpdateNode(node.DeepCopy())
			if err != nil {
				t.Fatalf("failed to update node labels: %v", err)
			}
			break
		}
	}
	err = f.TF.ManagedCluster.CreateDeployment(volExpDeploy)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		f.Step(t, "Clean up test Deployment")
		err = f.TF.ManagedCluster.DeleteDeployment(volExpDeploy.Name, volExpDeploy.Namespace)
		if err != nil {
			t.Fatal(err)
		}
	}()

	f.Step(t, "Verify Deployment %s/%s exists and Running", volExpDeploy.Namespace, volExpDeploy.Name)
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 3*time.Minute, true, func(_ context.Context) (bool, error) {
		deploy, err := f.TF.ManagedCluster.GetDeployment(volExpDeploy.Name, volExpDeploy.Namespace)
		if err != nil {
			t.Logf("Retrying due to fail to get Deployment %s/%s: %v", volExpDeploy.Namespace, volExpDeploy.Name, err)
			return false, nil
		}
		return lcmcommon.IsDeploymentReady(deploy), nil
	})
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to wait Deployment %s/%s running", volExpDeploy.Namespace, volExpDeploy.Name))
	}

	f.Step(t, "Verify size of mounted PV on Deployment %s/%s", volExpDeploy.Namespace, volExpDeploy.Name)
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 3*time.Minute, true, func(_ context.Context) (done bool, err error) {
		pods, _ := f.TF.ManagedCluster.ListPods("default", "")
		for _, pod := range pods {
			if pod.Labels != nil && pod.Labels["app"] == "vol-exp-deploy" {
				return pod.Status.Phase == v1.PodRunning, nil
			}
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait Deployment pod: %v", err)
	}
	stdout, _, err := f.TF.ManagedCluster.RunCommand("df -h /mnt/disk0", "default", "app=vol-exp-deploy")
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to produce df command for mounted volume"))
	}
	// Parse and verify volume size
	lines := strings.Split(stdout, "\n")
	dirtyColumns := strings.Split(lines[1], " ")
	columns := []string{}
	for _, item := range dirtyColumns {
		if item == "" {
			continue
		}
		columns = append(columns, item)
	}
	t.Logf("column %v", columns[1])
	size, err := resource.ParseQuantity(columns[1])
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to parse mounted volume size"))
	}
	_ = size.RoundUp(1)
	t.Logf("PVC size is %v", q.Value())
	t.Logf("Deployment /mnt/disk0 size if %v", size.Value())
	assert.True(t, q.Value()-size.Value() < 1000000000)

	f.Step(t, "Expand PVC %s/%s storage size", volExpPVC.Namespace, volExpPVC.Name)
	updatedQ, err := resource.ParseQuantity("12Gi")
	if err != nil {
		t.Fatal(err)
	}
	pvc, err := f.TF.ManagedCluster.GetPVC(volExpPVC.Name, volExpPVC.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	pvc.Spec.Resources.Requests[v1.ResourceStorage] = updatedQ
	err = f.TF.ManagedCluster.UpdatePVC(pvc)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Wait for PVC %s/%s expanding storage size", volExpPVC.Namespace, volExpPVC.Name)
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 3*time.Minute, true, func(_ context.Context) (bool, error) {
		pvc, err = f.TF.ManagedCluster.GetPVC(volExpPVC.Name, volExpPVC.Namespace)
		if err != nil {
			t.Logf("Retrying due to fail to get PVC %s/%s: %v", volExpPVC.Namespace, volExpPVC.Name, err)
			return false, nil
		}
		cond := pvc.Status.Conditions
		firstCond := cond != nil && cond[len(cond)-1].Type == v1.PersistentVolumeClaimFileSystemResizePending
		secondCond := pvc.Status.Capacity.Storage().Cmp(updatedQ) == 0
		return firstCond || secondCond, nil
	})
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to wait PVC %s/%s updated", volExpPVC.Namespace, volExpPVC.Name))
	}

	f.Step(t, "Restart Deployment %s/%s pod", volExpDeploy.Namespace, volExpDeploy.Name)
	t.Logf("Scaling %s deployment to 0 replicas", volExpDeployName)
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 3*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		_, err = f.TF.ManagedCluster.KubeClient.AppsV1().Deployments("default").UpdateScale(ctx, volExpDeployName, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      volExpDeployName,
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: 0,
			},
		}, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to scale default/%s deployment to %d", volExpDeployName, 0)
		}
		isScaled, err := f.TF.ManagedCluster.IsDeploymentScaled("default", volExpDeployName, 0)
		if err != nil {
			return false, errors.Wrapf(err, "failed to wait scale default/%s deployment to %d", volExpDeployName, 0)
		}
		return isScaled, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait Deployment pod deleted: %v", err)
	}
	t.Log("Wait for 3 minutes giving CSI a time to unmap rbd device")
	time.Sleep(3 * time.Minute)

	t.Logf("Scaling %s deployment to 1 replica", volExpDeployName)
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 3*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		_, err = f.TF.ManagedCluster.KubeClient.AppsV1().Deployments("default").UpdateScale(ctx, volExpDeployName, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      volExpDeployName,
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: 1,
			},
		}, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to scale default/%s deployment to %d", volExpDeployName, 1)
		}
		isScaled, err := f.TF.ManagedCluster.IsDeploymentScaled("default", volExpDeployName, 1)
		if err != nil {
			return false, errors.Wrapf(err, "failed to wait scale default/%s deployment to %d", volExpDeployName, 1)
		}
		return isScaled, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait Deployment pod deleted: %v", err)
	}

	f.Step(t, "Verify size of volume on Deployment %s/%s expanded", volExpDeploy.Namespace, volExpDeploy.Name)
	stdout, _, err = f.TF.ManagedCluster.RunCommand("df -h /mnt/disk0", "default", "app=vol-exp-deploy")
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to produce df command for mounted volume with error"))
	}
	// Parse and verify volume size
	lines = strings.Split(stdout, "\n")
	dirtyColumns = strings.Split(lines[1], " ")
	columns = []string{}
	for _, item := range dirtyColumns {
		if item == "" {
			continue
		}
		columns = append(columns, item)
	}
	t.Logf("size: %v", columns[1])
	size, err = resource.ParseQuantity(columns[1])
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to parse mounted volume size"))
	}
	_ = size.RoundUp(1)
	t.Logf("PVC size is %v", updatedQ.Value())
	t.Logf("Deployment /mnt/disk0 size is %v", size.Value())
	assert.True(t, updatedQ.Value()-size.Value() < 1000000000)

	t.Logf("#### Test %s complete sucessfully", t.Name())
}

func deferVolumeExpansionTest(pvcName, deployName string, t *testing.T) func() {
	return func() {
		t.Logf("Clean up PVC if exists")
		err := wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 3*time.Minute, true, func(_ context.Context) (done bool, err error) {
			pvc, err := f.TF.ManagedCluster.GetPVC(pvcName, "default")
			if k8serrors.IsNotFound(err) {
				return true, nil
			} else if err != nil {
				return false, nil
			}
			err = f.TF.ManagedCluster.DeletePVC(pvc.Name, pvc.Namespace)
			if k8serrors.IsNotFound(err) {
				return true, nil
			} else if err != nil {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			t.Fatalf("Failed to delete PVC default/%s: %v", pvcName, err)
		}
		t.Logf("Clean up Deployment if exists")
		err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 3*time.Minute, true, func(_ context.Context) (done bool, err error) {
			deploy, err := f.TF.ManagedCluster.GetDeployment(deployName, "default")
			if k8serrors.IsNotFound(err) {
				return true, nil
			} else if err != nil {
				return false, nil
			}
			err = f.TF.ManagedCluster.DeleteDeployment(deploy.Name, deploy.Namespace)
			if k8serrors.IsNotFound(err) {
				return true, nil
			} else if err != nil {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			t.Fatalf("Failed to delete Deployment default/%s: %v", deployName, err)
		}
		t.Log("Teardown started..")
		err = f.Teardown()
		if err != nil {
			t.Logf("Teardown failed: %v", err)
			t.Fatal(err)
		}
		t.Log("Teardown successfully done")
	}
}
