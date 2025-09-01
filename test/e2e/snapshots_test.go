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
	"fmt"
	"strings"
	"testing"
	"time"

	vsapi "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	vsclient "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

func createPVC(pvcName, namespace, storageClassName string) error {
	f.TF.Log.Info().Msgf("Creating testing PVC with storage class '%s'", storageClassName)
	testPvc := v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			StorageClassName: &storageClassName,
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
	err := f.TF.ManagedCluster.CreatePVC(&testPvc)
	if err != nil {
		return errors.Wrapf(err, "failed to create PVC %s/%s", testPvc.Namespace, testPvc.Name)
	}
	return nil
}

func createTestDeployment(pvcName, mountPath, testImage, deploymentName, namespace string) error {
	f.TF.Log.Info().Msgf("Creating testing '%s/%s' deployment", namespace, deploymentName)
	replicas := int32(1)
	testDeploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": pvcName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": pvcName,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": pvcName,
					},
				},
				Spec: v1.PodSpec{
					// Avoid spawning on UCP Master node
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "node-role.kubernetes.io/master",
												Operator: v1.NodeSelectorOpDoesNotExist,
											},
										},
									},
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:  "basic",
							Image: testImage,
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      pvcName,
									MountPath: mountPath,
								},
							},
							Command: []string{
								"/bin/sh",
								"-c",
								"sleep 60m",
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: pvcName,
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
									ReadOnly:  false,
								},
							},
						},
					},
				},
			},
		},
	}
	err := f.TF.ManagedCluster.CreateDeployment(&testDeploy)
	if err != nil {
		return errors.Wrapf(err, "failed to create deployment %s/%s", testDeploy.Namespace, testDeploy.Name)
	}
	err = f.TF.ManagedCluster.WaitDeploymentReady(testDeploy.Name, testDeploy.Namespace)
	if err != nil {
		return errors.Wrapf(err, "deployment %s/%s is not ready", testDeploy.Namespace, testDeploy.Name)
	}
	return nil
}

func createPVCFromSnapshot(pvcName, namespace, pvcSnapshotname, storageClassName string) error {
	f.TF.Log.Info().Msgf("Restoring PVC snapshot for PVC based on '%s' snapshot", pvcSnapshotname)
	testPvc := v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			StorageClassName: &storageClassName,
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
			DataSource: &v1.TypedLocalObjectReference{
				APIGroup: &[]string{"snapshot.storage.k8s.io"}[0],
				Kind:     "VolumeSnapshot",
				Name:     pvcSnapshotname,
			},
		},
	}
	err := f.TF.ManagedCluster.CreatePVC(&testPvc)
	if err != nil {
		return errors.Wrapf(err, "failed to create PVC %s/%s", testPvc.Namespace, testPvc.Name)
	}
	return nil
}

func cleanupObjects(t *testing.T, vsClient *vsclient.Clientset, namespace, basePvcName, baseDeployName, pvcSnapshotName, snapshotPvcName, snapshotDeployName string) {
	f.TF.Log.Info().Msg("Cleaning after test...")
	err := f.TF.ManagedCluster.DeleteDeployment(baseDeployName, namespace)
	if err != nil {
		t.Fatal(err)
	}
	err = f.TF.ManagedCluster.DeletePVC(basePvcName, namespace)
	if err != nil {
		t.Fatal(err)
	}
	err = f.TF.ManagedCluster.DeleteDeployment(snapshotDeployName, namespace)
	if err != nil {
		t.Fatal(err)
	}
	err = vsClient.SnapshotV1().VolumeSnapshots(namespace).Delete(f.TF.ManagedCluster.Context, pvcSnapshotName, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		t.Fatal(err)
	}
	err = f.TF.ManagedCluster.DeletePVC(snapshotPvcName, namespace)
	if err != nil {
		t.Fatal(err)
	}
}

func runSnapshotTest(t *testing.T, volumeType string) {
	if volumeType != "rbd" && volumeType != "cephfs" {
		t.Fatalf("unknown volume type '%s'", volumeType)
	}
	vsclassDriver := fmt.Sprintf("rook-ceph.%s.csi.ceph.com", volumeType)

	vsClient, err := vsclient.NewForConfig(f.TF.ManagedCluster.KubeConfig)
	if err != nil {
		t.Fatal("failed to create vsclient from kubeconfig")
	}

	f.Step(t, "Check VSClass with driver '%s' is present", vsclassDriver)
	vsclasses, err := vsClient.SnapshotV1().VolumeSnapshotClasses().List(f.TF.ManagedCluster.Context, metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(vsclasses.Items) == 0 {
		t.Skipf("Test is actual only when VSClass with driver '%s' is present", vsclassDriver)
	}
	vsClassName := ""
	for _, vsclass := range vsclasses.Items {
		if vsclass.Driver == vsclassDriver {
			vsClassName = vsclass.Name
			break
		}
	}
	if vsClassName == "" {
		t.Skipf("Test is actual only when VSClass with driver '%s' is present", vsclassDriver)
	}

	testImage := f.TF.ManagedCluster.LcmConfig.DeployParams.CephImage
	namespaceToUse := "default"
	mountPath := "/mnt/pvc-mount"
	testFile := fmt.Sprintf("%s/test-file", mountPath)

	f.Step(t, "Find appropriate storage class name")
	scList, err := f.TF.ManagedCluster.ListStorageClass()
	if err != nil {
		t.Fatal(err)
	}
	storageClassName := ""
	for _, sc := range scList.Items {
		if sc.Provisioner == vsclassDriver {
			storageClassName = sc.Name
			break
		}
	}
	if storageClassName == "" {
		t.Fatal("failed to find required storage class name")
	}

	curtime := time.Now().Unix()
	basePvcName := fmt.Sprintf("%s-e2e-%d", volumeType, curtime)
	baseDeployName := fmt.Sprintf("%s-e2e-%d", volumeType, curtime)
	pvcSnapshotName := basePvcName
	snapshotPvcName := fmt.Sprintf("%s-snapshot-e2e-%d", volumeType, curtime)
	snapshotDeployName := fmt.Sprintf("%s-snapshot-e2e-%d", volumeType, curtime)

	if !f.TF.TestConfig.Settings.KeepAfter {
		defer cleanupObjects(t, vsClient, namespaceToUse, basePvcName, baseDeployName, pvcSnapshotName, snapshotPvcName, snapshotDeployName)
	}

	f.Step(t, "Create test PVC")
	err = createPVC(basePvcName, namespaceToUse, storageClassName)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Create test deployment")
	err = createTestDeployment(basePvcName, mountPath, testImage, baseDeployName, namespaceToUse)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Push some test data to PVC")
	podLabel := "app=" + basePvcName
	cmd := fmt.Sprintf("dd if=/dev/urandom of=%s bs=1M count=20", testFile)
	_, _, err = f.TF.ManagedCluster.RunCommand(cmd, namespaceToUse, podLabel)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to write test data"))
	}
	// sleep sometime to give Ceph sync data
	time.Sleep(1 * time.Minute)

	f.Step(t, "Create PVC snapshot")
	f.TF.Log.Info().Msgf("Creating PVC snapshot for PVC '%s' based on '%s' class", basePvcName, vsClassName)
	pvcsnapshot := vsapi.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcSnapshotName,
			Namespace: namespaceToUse,
		},
		Spec: vsapi.VolumeSnapshotSpec{
			VolumeSnapshotClassName: &vsClassName,
			Source: vsapi.VolumeSnapshotSource{
				PersistentVolumeClaimName: &basePvcName,
			},
		},
	}
	_, err = vsClient.SnapshotV1().VolumeSnapshots(namespaceToUse).Create(f.TF.ManagedCluster.Context, &pvcsnapshot, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = lcmcommon.RunFuncWithRetry(5, 30*time.Second, func() (interface{}, error) {
		snapshot, err := vsClient.SnapshotV1().VolumeSnapshots(namespaceToUse).Get(f.TF.ManagedCluster.Context, pvcsnapshot.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if snapshot.Status != nil && snapshot.Status.ReadyToUse != nil && *snapshot.Status.ReadyToUse {
			return nil, nil
		}
		msg := fmt.Sprintf("snapshot %s is not ready yet", snapshot.Name)
		if snapshot.Status != nil && snapshot.Status.Error != nil && snapshot.Status.Error.Message != nil {
			msg = fmt.Sprintf("%s, error: %s", msg, *snapshot.Status.Error.Message)
		}
		f.TF.Log.Warn().Msg(msg)
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Push extra test data to PVC")
	cmd = fmt.Sprintf("dd if=/dev/urandom of=%s bs=1M count=5", testFile)
	_, _, err = f.TF.ManagedCluster.RunCommand(cmd, namespaceToUse, podLabel)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to write test data"))
	}

	f.Step(t, "Restore PVC snapshot")
	err = createPVCFromSnapshot(snapshotPvcName, namespaceToUse, pvcSnapshotName, storageClassName)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Create deployment from snapshot")
	err = createTestDeployment(snapshotPvcName, mountPath, testImage, snapshotDeployName, namespaceToUse)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Verify test data on PVC")
	cmd = fmt.Sprintf("wc -c %s", testFile)
	output, _, err := f.TF.ManagedCluster.RunCommand(cmd, namespaceToUse, "app="+snapshotPvcName)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to read test data"))
	}
	fileSize := strings.Split(output, " ")[0]
	if fileSize != "20971520" {
		t.Fatalf("Data on snapshot PVC is not the same to before snapshot, actual data size is '%s', expected size '20971520'", fileSize)
	}
}

func TestRBDVolumeSnapshot(t *testing.T) {
	t.Log("e2e test: create and test RBD PVC snapshot.")
	defer f.SetupTeardown(t)()
	runSnapshotTest(t, "rbd")
}

func TestCephFsVolumeSnapshot(t *testing.T) {
	t.Log("e2e test: create and test CephFS PVC snapshot.")
	defer f.SetupTeardown(t)()
	runSnapshotTest(t, "cephfs")
}
