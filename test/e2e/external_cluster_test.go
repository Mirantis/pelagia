/*
Copyright 2026 Mirantis IT.

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
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	bktv1alpha1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/v3/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	f "github.com/Mirantis/pelagia/v3/test/e2e/framework"
)

func prepareExternalPVC(cluster *f.ManagedConfig, storageClassName string, accessMode v1.PersistentVolumeAccessMode) (string, error) {
	pvcName := fmt.Sprintf("test-external-pvc-%v", time.Now().Unix())
	testPvc := v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: "default",
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes:      []v1.PersistentVolumeAccessMode{accessMode},
			StorageClassName: &storageClassName,
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
	_, err := cluster.KubeClient.CoreV1().PersistentVolumeClaims("default").Create(cluster.Context, &testPvc, metav1.CreateOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "failed to create PVC %s/%s", "default", pvcName)
	}
	return pvcName, nil
}

func prepareExternalOBC(cluster *f.ManagedConfig, storageClassName string) (string, error) {
	obcName := fmt.Sprintf("test-external-bucket-%v", time.Now().Unix())
	testObc := bktv1alpha1.ObjectBucketClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obcName,
			Namespace: "default",
		},
		Spec: bktv1alpha1.ObjectBucketClaimSpec{
			GenerateBucketName: "external-bucket",
			StorageClassName:   storageClassName,
		},
	}
	_, err := cluster.BucketClientSet.ObjectbucketV1alpha1().ObjectBucketClaims("default").Create(cluster.Context, &testObc, metav1.CreateOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "failed to create OBC %s/%s", "default", obcName)
	}
	return obcName, nil
}

func prepareExternalTestDeployment(cluster *f.ManagedConfig, claimType, refName, imageName string, replicas int32) (string, error) {
	deployName := fmt.Sprintf("test-external-deployment-%v", time.Now().Unix())
	testDeploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployName,
			Namespace: "default",
			Labels:    map[string]string{"app": deployName},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": deployName},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": deployName},
				},
				Spec: v1.PodSpec{
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
											{
												Key:      "node-role.kubernetes.io/control-plane",
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
							Name:  "simple-container",
							Image: imageName,
							Command: []string{
								"/bin/sh",
								"-c",
								"sleep 60m",
							},
						},
					},
				},
			},
		},
	}
	if claimType == "pvc" {
		testDeploy.Spec.Template.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{
			{
				Name:      refName,
				MountPath: "/mnt",
			},
		}
		testDeploy.Spec.Template.Spec.Volumes = []v1.Volume{
			{
				Name: refName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: refName,
						ReadOnly:  false,
					},
				},
			},
		}
	} else {
		testDeploy.Spec.Template.Spec.Containers[0].EnvFrom = []v1.EnvFromSource{
			{ConfigMapRef: &v1.ConfigMapEnvSource{Name: refName}},
			{SecretRef: &v1.SecretEnvSource{Name: refName}},
		}
	}
	_, err := cluster.KubeClient.AppsV1().Deployments("default").Create(cluster.Context, &testDeploy, metav1.CreateOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "failed to create deployment default/%s", deployName)
	}
	return deployName, nil
}

func TestExternalCephCluster(t *testing.T) {
	runExternalClusterTest(t, true)
}

func TestExternalCephClusterNonAdmin(t *testing.T) {
	runExternalClusterTest(t, false)
}

func runExternalClusterTest(t *testing.T, isAdmin bool) {
	t.Logf("#### e2e test: test sharing ceph cluster across region")
	defer f.SetupTeardown(t)()
	f.Step(t, "Obtain test case configuration")
	caseConfig := f.GetConfigForTestCase(t)
	externalClusterKubeconfigPath, ok := caseConfig["externalClusterKubeconfigPath"]
	if !ok {
		t.Fatal("Could not obtain test case externalClusterKubeconfigPath configuration")
	}
	if !path.IsAbs(externalClusterKubeconfigPath) {
		externalClusterKubeconfigPath, _ = filepath.Abs(externalClusterKubeconfigPath)
	}
	externalConfig, err := clientcmd.BuildConfigFromFlags("", externalClusterKubeconfigPath)
	if err != nil {
		t.Fatalf("Cannot build kube from external kubeconfig: %v", err)
	}
	externalClusterNamespace := caseConfig["externalClusterNamespace"]
	if externalClusterNamespace == "" {
		t.Log("### deployment namespace var for external cluster 'externalClusterNamespace' is not set, using same to master cluster")
		externalClusterNamespace = f.TF.ManagedCluster.LcmNamespace
	}
	externalClusterKeepAfter := caseConfig["externalClusterKeepAfter"] != ""
	externalCluster, err := f.NewManagedCluster(externalClusterNamespace, externalConfig)
	if err != nil {
		t.Fatalf("Cannot initialize external cluster clients: %v", err)
	}
	var rbdPoolsForShare, cephfsPoolsForShare []string
	rbdPoolsForShareStr := caseConfig["rbdPoolsForShare"]
	if rbdPoolsForShareStr != "" {
		rbdPoolsForShare = strings.Split(rbdPoolsForShareStr, ",")
	}
	cephfsPoolsForShareStr := caseConfig["cephfsPoolsForShare"]
	if cephfsPoolsForShareStr != "" {
		cephfsPoolsForShare = strings.Split(cephfsPoolsForShareStr, ",")
	}
	rgwObjectStoreForShare := caseConfig["rgwObjectStoreForShare"]
	if len(rbdPoolsForShare) == 0 && len(cephfsPoolsForShare) == 0 && rgwObjectStoreForShare == "" {
		t.Fatal("At least one parameter cephfsPoolsForShare or rbdPoolsForShare or rgwObjectStoreForShare should be non-empty")
	}

	f.Step(t, "Check main cluster configuration")
	connectionStringArgs := []string{"pelagia-connector"}
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	err = f.TF.ManagedCluster.WaitForCephDeploymentReady(cd.Name)
	if err != nil {
		t.Fatal(err)
	}
	err = f.TF.ManagedCluster.WaitForCephDeploymentHealthReady(cd.Name)
	if err != nil {
		t.Fatal(err)
	}
	poolsSection := make([]cephlcmv1alpha1.CephPool, len(rbdPoolsForShare))
	for idx, sharePool := range rbdPoolsForShare {
		poolFound := false
		for _, pool := range cd.Spec.BlockStorage.Pools {
			castedPool, _ := pool.GetSpec()
			if pool.Name == sharePool && pool.UseAsFullName || fmt.Sprintf("%s-%s", pool.Name, castedPool.DeviceClass) == sharePool {
				poolFound = true
				poolsSection[idx] = pool
				break
			}
		}
		if !poolFound {
			t.Fatalf("Could not find RBD pool '%s' in source Ceph cluster, but required for testing", sharePool)
		}
		connectionStringArgs = append(connectionStringArgs, "--use-rbd")
	}
	cephFSSection := &cephlcmv1alpha1.CephSharedFilesystem{
		Filesystems: []cephlcmv1alpha1.CephFilesystem{},
	}
	if len(cephfsPoolsForShare) > 0 {
		if cd.Spec.SharedFilesystem == nil {
			t.Fatal("CephFS is not found in source Ceph cluster, but required for testing")
		}
		for _, shareCephFsPool := range cephfsPoolsForShare {
			poolFound := false
			for _, cephFS := range cd.Spec.SharedFilesystem.Filesystems {
				castedFsSpec, _ := cephFS.GetSpec()
				for _, dataPool := range castedFsSpec.DataPools {
					if fmt.Sprintf("%s-%s", cephFS.Name, dataPool.Name) == shareCephFsPool {
						poolFound = true
						added := false
						for idx, externalCephFs := range cephFSSection.Filesystems {
							if cephFS.Name == externalCephFs.Name {
								added = true
								castedExternalFs, _ := externalCephFs.GetSpec()
								castedExternalFs.DataPools = append(castedExternalFs.DataPools, dataPool)
								fsSpecRaw, err := cephlcmv1alpha1.DecodeStructToRaw(castedExternalFs)
								if err != nil {
									t.Fatal(err)
								}
								cephFSSection.Filesystems[idx].FsSpec.Raw = fsSpecRaw
								break
							}
						}
						if !added {
							copyCephFs := cephFS.DeepCopy()
							casted, _ := copyCephFs.GetSpec()
							casted.DataPools = []cephv1.NamedPoolSpec{dataPool}
							fsSpecRaw, err := cephlcmv1alpha1.DecodeStructToRaw(casted)
							if err != nil {
								t.Fatal(err)
							}
							copyCephFs.FsSpec.Raw = fsSpecRaw
							cephFSSection.Filesystems = append(cephFSSection.Filesystems, *copyCephFs)
						}
						break
					}
				}
				if poolFound {
					break
				}
			}
			if !poolFound {
				t.Fatalf("Could not find CephFS datapool '%s' in source Ceph cluster, but required for testing", shareCephFsPool)
			}
		}
		connectionStringArgs = append(connectionStringArgs, "--use-cephfs")
	}
	rgwExternalSection := &cephlcmv1alpha1.CephObjectStorage{}
	if rgwObjectStoreForShare != "" {
		if cd.Spec.ObjectStorage == nil || len(cd.Spec.ObjectStorage.Rgws) == 0 {
			t.Fatal("Could not find RGW ObjectStorage in source Ceph cluster, but required for testing")
		}
		endpoints, err := f.GetRgwPublicEndpoints(cd.Name, rgwObjectStoreForShare)
		if err != nil {
			t.Fatal(err)
		}
		if len(endpoints) == 0 {
			t.Fatal("failed to find public rgw endpoint in source Ceph cluster, which is required")
		}
		// TODO: currently using only plain http connection
		v := strings.Split(endpoints[0], "://")
		endpointWithPort := v[0]
		if len(v) > 1 {
			endpointWithPort = v[1]
		}
		conn := strings.Split(endpointWithPort, ":")
		endpoint := conn[0]
		externalRgw, err := cephlcmv1alpha1.DecodeStructToRaw(
			cephv1.ObjectStoreSpec{
				Gateway: cephv1.GatewaySpec{
					Port:                 80,
					ExternalRgwEndpoints: []cephv1.EndpointAddress{{IP: endpoint}},
				},
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		rgwExternalSection = &cephlcmv1alpha1.CephObjectStorage{
			Rgws: []cephlcmv1alpha1.CephObjectStore{
				{
					Name: "external-rgw",
					Spec: runtime.RawExtension{Raw: externalRgw},
				},
			},
		}
		connectionStringArgs = append(connectionStringArgs, "--use-rgw")
	}

	if isAdmin {
		connectionStringArgs = append(connectionStringArgs, "--client-name admin")
	} else {
		f.Step(t, "Create non-admin user in source Ceph cluster")
		testClientName := fmt.Sprintf("non-admin-client-%v", time.Now().Unix())
		connectionStringArgs = append(connectionStringArgs, fmt.Sprintf("--client-name %s", testClientName))
		osdCaps := []string{}
		for _, pool := range rbdPoolsForShare {
			osdCaps = append(osdCaps, "profile rbd pool="+pool)
		}
		if len(cephfsPoolsForShare) > 0 {
			osdCaps = append(osdCaps, "allow rw tag cephfs *=*")
		}
		clientSpec := cephv1.ClientSpec{
			Name: testClientName,
			Caps: map[string]string{
				"mgr": "allow r",
				"mon": "allow r, profile role-definer",
				"osd": strings.Join(osdCaps, ", "),
			},
		}
		if len(cephfsPoolsForShare) > 0 {
			clientSpec.Caps["mds"] = "allow rw"
		}
		clientRaw, _ := cephlcmv1alpha1.DecodeStructToRaw(clientSpec)
		client := cephlcmv1alpha1.CephClient{ClientSpec: runtime.RawExtension{Raw: clientRaw}}
		if len(cd.Spec.Clients) > 0 {
			cd.Spec.Clients = append(cd.Spec.Clients, client)
		} else {
			cd.Spec.Clients = []cephlcmv1alpha1.CephClient{client}
		}
		err = f.UpdateCephDeploymentSpec(cd, true)
		if err != nil {
			t.Fatal(err)
		}
	}

	f.Step(t, "Prepare connection string for external")
	connectionStringCmd := strings.Join(connectionStringArgs, " ")
	connectionString, _, err := f.TF.ManagedCluster.RunCommand(connectionStringCmd, f.TF.ManagedCluster.LcmNamespace, "app=pelagia-deployment-controller")
	if err != nil {
		t.Fatalf("failed to generate connection string with error: %v", err)
	}
	assert.NotEmpty(t, connectionString)
	t.Logf("Connection string is: %v", connectionString)

	f.Step(t, "Creating external connection secret for external cluster")
	externalConnectionSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pelagia-external-connection",
			Namespace: externalClusterNamespace,
		},
		Data: map[string][]byte{
			"connection": []byte(connectionString),
		},
	}
	err = externalCluster.CreateSecret(externalConnectionSecret)
	if err != nil {
		t.Fatalf("failed to create secret with external connection info: %v", err)
	}

	f.Step(t, "Build and create external Ceph cluster")
	externalRaw, err := cephlcmv1alpha1.DecodeStructToRaw(
		cephv1.ClusterSpec{External: cephv1.ExternalSpec{Enable: true}},
	)
	if err != nil {
		t.Fatal(err)
	}
	externalCephCluster := &cephlcmv1alpha1.CephDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-ceph",
			Namespace: externalClusterNamespace,
		},
		Spec: cephlcmv1alpha1.CephDeploymentSpec{
			Cluster: cephlcmv1alpha1.CephCluster{
				RawExtension: runtime.RawExtension{Raw: externalRaw},
			},
			BlockStorage: &cephlcmv1alpha1.CephBlockStorage{
				Pools: poolsSection,
			},
			Nodes:            []cephlcmv1alpha1.CephDeploymentNode{},
			SharedFilesystem: cephFSSection,
			ObjectStorage:    rgwExternalSection,
		},
	}

	f.Step(t, "Creating external CephDeployment and wait its ready")
	err = externalCluster.CreateCephDeployment(externalCephCluster)
	if err != nil {
		t.Fatalf("failed to create external CephDeployment: %v", err)
	}
	err = externalCluster.WaitForCephDeploymentReady(externalCephCluster.Name)
	if err != nil {
		t.Fatal(err)
	}
	err = externalCluster.WaitForCephDeploymentHealthReady(externalCephCluster.Name)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Get storageclasses for pools")
	scList, err := externalCluster.ListStorageClass()
	if err != nil {
		t.Fatal(err)
	}
	scMap := map[string][]string{
		"rbd":    {},
		"cephfs": {},
	}
	for _, sc := range scList.Items {
		for _, rbdPool := range rbdPoolsForShare {
			if sc.Parameters["pool"] == rbdPool {
				scMap["rbd"] = append(scMap["rbd"], sc.Name)
				break
			}
		}
		for _, cephFsPool := range cephfsPoolsForShare {
			if sc.Parameters["pool"] == cephFsPool {
				scMap["cephfs"] = append(scMap["cephfs"], sc.Name)
				break
			}
		}
	}

	errMsgs := []string{}
	cleanupPvc := func(pvcName string) {
		t.Logf("Clean up test PVC default/%s", pvcName)
		err = externalCluster.KubeClient.CoreV1().PersistentVolumeClaims("default").Delete(externalCluster.Context, pvcName, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			errMsg := fmt.Sprintf("failed to remove 'default/%s' test pvc: %v", pvcName, err)
			t.Log(errMsg)
			errMsgs = append(errMsgs, errMsg)
		}
	}

	cleanupDeployment := func(deployName string) {
		t.Logf("Clean up test deployment default/%s", deployName)
		err = externalCluster.KubeClient.AppsV1().Deployments("default").Delete(externalCluster.Context, deployName, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			errMsg := fmt.Sprintf("failed to remove 'default/%s' test deployment: %v", deployName, err)
			t.Log(errMsg)
			errMsgs = append(errMsgs, errMsg)
		}
	}

	if len(rbdPoolsForShare) > 0 {
		f.Step(t, "test RBD pools' StorageClasses for external cluster")
		for idx, rbdSc := range scMap["rbd"] {
			func() {
				t.Logf("creating testing external PVC for RBD storageclass %s", rbdSc)
				pvcName, err := prepareExternalPVC(externalCluster, rbdSc, v1.ReadWriteOnce)
				if err != nil {
					msg := fmt.Sprintf("testing '%s' storageclass: failed to prepare test pvc: %v", rbdSc, err)
					t.Log(msg)
					errMsgs = append(errMsgs, msg)
					return
				}
				defer cleanupPvc(pvcName)

				testDeployment := func(newData bool) error {
					deployName, err := prepareExternalTestDeployment(externalCluster, "pvc", pvcName, externalCluster.LcmConfig.DeployParams.CephImage, 1)
					if err != nil {
						return errors.Wrapf(err, "testing '%s' storageclass: failed to prepare test deployment", rbdSc)
					}
					defer cleanupDeployment(deployName)

					err = externalCluster.WaitDeploymentReady(deployName, "default")
					if err != nil {
						return errors.Wrapf(err, "deployment 'default/%s' is not ready", deployName)
					}
					podLabel := fmt.Sprintf("app=%s", deployName)

					if newData {
						t.Logf("writing content to a PV based on '%s' storageclass for 'default/%s' deployment", rbdSc, deployName)
						_, _, err := externalCluster.RunCommand(fmt.Sprintf("touch /mnt/%d", idx), "default", podLabel)
						if err != nil {
							return errors.Wrapf(err, "testing '%s' storageclass: failed to write content to 'default/%s' deployment", rbdSc, deployName)
						}
					}
					t.Logf("checking content from PV based on '%s' storageclass from 'default/%s' deployment", rbdSc, deployName)
					_, _, err = externalCluster.RunCommand(fmt.Sprintf("test -f /mnt/%d", idx), "default", podLabel)
					if err != nil {
						return errors.Wrapf(err, "testing '%s' storageclass: failed to get content from 'default/%s' deployment", rbdSc, deployName)
					}
					return nil
				}

				t.Logf("creating testing deployment for PVC %s", pvcName)
				err = testDeployment(true)
				if err != nil {
					t.Log(err)
					errMsgs = append(errMsgs, err.Error())
					return
				}
				t.Logf("re-create test deployment and re-attach existing PVC %s", pvcName)
				err = testDeployment(false)
				if err != nil {
					t.Log(err)
					errMsgs = append(errMsgs, err.Error())
				}
			}()
		}
	}

	if len(cephfsPoolsForShare) > 0 {
		f.Step(t, "Test CephFS pools' StorageClasses for external cluster")
		for idx, cephFSSc := range scMap["cephfs"] {
			func() {
				t.Logf("creating testing external PVC for CephFS storageclass %s", cephFSSc)
				pvcName, err := prepareExternalPVC(externalCluster, cephFSSc, v1.ReadWriteMany)
				if err != nil {
					msg := fmt.Sprintf("testing '%s' storageclass: failed to prepare test pvc: %v", cephFSSc, err)
					t.Log(msg)
					errMsgs = append(errMsgs, msg)
					return
				}
				defer cleanupPvc(pvcName)

				testDeployment := func(newData bool) error {
					deployName, err := prepareExternalTestDeployment(externalCluster, "pvc", pvcName, externalCluster.LcmConfig.DeployParams.CephImage, 2)
					if err != nil {
						return errors.Wrapf(err, "testing '%s' storageclass: failed to prepare test deployment", cephFSSc)
					}
					defer cleanupDeployment(deployName)

					err = externalCluster.WaitDeploymentReady(deployName, "default")
					if err != nil {
						return errors.Wrapf(err, "deployment 'default/%s' is not ready", deployName)
					}
					podLabel := fmt.Sprintf("app=%s", deployName)

					deployPodList, err := externalCluster.ListPods("default", podLabel)
					if err != nil {
						return errors.Wrapf(err, "%s", fmt.Sprintf("testing '%s' storageclass: failed to find test deployment pods", cephFSSc))
					}

					if newData {
						for jdx, pod := range deployPodList {
							t.Logf("writing content to a PV based on '%s' storageclass for 'default/%s' pod", cephFSSc, pod.Name)
							_, _, err := externalCluster.RunPodCommandWithContent(fmt.Sprintf("tee -a /mnt/%d", idx), pod.Spec.Containers[0].Name, &pod, fmt.Sprintf("%d\n", jdx))
							if err != nil {
								return errors.Wrapf(err, "testing '%s' storageclass: failed to write content to 'default/%s' pod", cephFSSc, pod.Name)
							}
						}
					}

					for _, pod := range deployPodList {
						t.Logf("checking content from PV based on '%s' storageclass from 'default/%s' pod", cephFSSc, pod.Name)
						stdout, _, err := externalCluster.RunPodCommand(fmt.Sprintf("cat /mnt/%d", idx), pod.Spec.Containers[0].Name, &pod)
						if err != nil {
							return errors.Wrapf(err, "testing '%s' storageclass: failed to get content from 'default/%s' pod", cephFSSc, pod.Name)
						}
						contentStr := strings.Trim(stdout, "\n")
						content := strings.Split(contentStr, "\n")
						if len(content) != len(deployPodList) {
							return errors.Errorf("testing '%s' storageclass: could not find expected content from 'default/%s' pod: expected='%d', found='%d'",
								cephFSSc, pod.Name, len(deployPodList), len(content))
						}
						for jdx, item := range content {
							if item != fmt.Sprintf("%d", jdx) {
								return errors.Errorf("testing '%s' storageclass: unexpected content from 'default/%s' pod: '%s'", cephFSSc, pod.Name, item)
							}
						}
					}
					return nil
				}

				t.Logf("creating testing deployment for PVC %s", pvcName)
				err = testDeployment(true)
				if err != nil {
					t.Log(err)
					errMsgs = append(errMsgs, err.Error())
					return
				}
				t.Logf("re-create test deployment and re-attach existing PVC %s", pvcName)
				err = testDeployment(false)
				if err != nil {
					t.Log(err)
					errMsgs = append(errMsgs, err.Error())
				}
			}()
		}
	}

	if rgwObjectStoreForShare != "" {
		f.Step(t, "Test RGW OBC for external cluster")

		cleanupOBC := func(obcName string) {
			t.Logf("Clean up test OBC default/%s", obcName)
			err = externalCluster.BucketClientSet.ObjectbucketV1alpha1().ObjectBucketClaims("default").Delete(externalCluster.Context, obcName, metav1.DeleteOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				errMsg := fmt.Sprintf("failed to remove 'default/%s' test obc: %v", obcName, err)
				t.Log(errMsg)
				errMsgs = append(errMsgs, errMsg)
			}
		}

		testDeployment := func() error {
			t.Logf("creating testing external OBC for RGW %s", externalCephCluster.Spec.ObjectStorage.Rgws[0].Name)
			storageClassName := fmt.Sprintf("%s-bucket", externalCephCluster.Spec.ObjectStorage.Rgws[0].Name)
			obcName, err := prepareExternalOBC(externalCluster, storageClassName)
			if err != nil {
				return errors.Wrapf(err, "testing '%s' storageclass: failed to prepare test obc", storageClassName)
			}
			defer cleanupOBC(obcName)

			t.Logf("creating testing deployment for OBC %s", obcName)
			deployName, err := prepareExternalTestDeployment(externalCluster, "obc", obcName, f.TF.E2eImage, 1)
			if err != nil {
				return errors.Wrapf(err, "testing '%s' storageclass: failed to prepare test deployment", storageClassName)
			}
			defer cleanupDeployment(deployName)

			err = externalCluster.WaitDeploymentReady(deployName, "default")
			if err != nil {
				return errors.Wrapf(err, "deployment 'default/%s' is not ready", deployName)
			}
			podLabel := fmt.Sprintf("app=%s", deployName)

			deployPodList, err := externalCluster.ListPods("default", podLabel)
			if err != nil {
				return errors.Wrapf(err, "%s", fmt.Sprintf("testing '%s' storageclass: failed to find test deployment pods", storageClassName))
			}
			pod := deployPodList[0]
			t.Logf("writing content to a bucket based on '%s' storageclass for 'default/%s' pod", storageClassName, pod.Name)
			_, _, err = externalCluster.RunPodCommand(`sh -c "aws s3 cp /etc/hostname s3://$BUCKET_NAME/test.txt --endpoint-url http://$BUCKET_HOST:$BUCKET_PORT"`, pod.Spec.Containers[0].Name, &pod)
			if err != nil {
				return errors.Wrapf(err, "testing '%s' storageclass: failed to write content to 'default/%s' pod", storageClassName, pod.Name)
			}
			t.Logf("checking content from PV based on '%s' storageclass from 'default/%s' pod", storageClassName, pod.Name)
			_, _, err = externalCluster.RunPodCommand(`sh -c "aws s3 cp s3://$BUCKET_NAME/test.txt /tmp/test.txt --endpoint-url http://$BUCKET_HOST:$BUCKET_PORT"`, pod.Spec.Containers[0].Name, &pod)
			if err != nil {
				return errors.Wrapf(err, "testing '%s' storageclass: failed to get content from 'default/%s' pod", storageClassName, pod.Name)
			}
			stdoutTarget, _, err := externalCluster.RunPodCommand("md5sum -z --tag /tmp/test.txt", pod.Spec.Containers[0].Name, &pod)
			if err != nil {
				return errors.Wrapf(err, "testing '%s' storageclass: failed to get content from 'default/%s' pod", storageClassName, pod.Name)
			}
			stdoutSource, _, err := externalCluster.RunPodCommand("md5sum -z --tag /etc/hostname", pod.Spec.Containers[0].Name, &pod)
			if err != nil {
				return errors.Wrapf(err, "testing '%s' storageclass: failed to get content from 'default/%s' pod", storageClassName, pod.Name)
			}
			if len(stdoutSource) != len(stdoutTarget) {
				return errors.Errorf("testing '%s' storageclass: could not find expected content from 'default/%s' pod: expected='%s', found='%s'",
					storageClassName, pod.Name, stdoutSource, stdoutTarget)
			}
			return nil
		}

		err = testDeployment()
		if err != nil {
			t.Log(err)
			errMsgs = append(errMsgs, err.Error())
		}
	}

	if !externalClusterKeepAfter {
		f.Step(t, "Remove external CephDeployment")
		err = externalCluster.RemoveCephDeployment(externalCephCluster.Name)
		if err != nil {
			t.Fatalf("failed to remove external CephDeployment: %v", err)
		}
	}
	if len(errMsgs) > 0 {
		t.Fatalf("The following error(s) raised during StorageClass testing: %v", strings.Join(errMsgs, "; "))
	}
	t.Logf("Test %s complete sucessfully", t.Name())
}
