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

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	f "github.com/Mirantis/pelagia/test/e2e/framework"
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

func prepareExternalTestDeployment(cluster *f.ManagedConfig, pvcName string, replicas int32) (string, error) {
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
							Image: cluster.LcmConfig.DeployParams.CephImage,
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      pvcName,
									MountPath: "/mnt",
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
	if len(rbdPoolsForShare) == 0 && len(cephfsPoolsForShare) == 0 {
		t.Fatal("At least one parameter cephfsPoolsForShare or rbdPoolsForShare should be non-empty")
	}

	f.Step(t, "Check main cluster configuration")
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
		for _, pool := range cd.Spec.Pools {
			if pool.Name == sharePool && pool.UseAsFullName || fmt.Sprintf("%s-%s", pool.Name, pool.DeviceClass) == sharePool {
				poolFound = true
				poolsSection[idx] = pool
				break
			}
		}
		if !poolFound {
			t.Fatalf("Could not find RBD pool '%s' in source Ceph cluster, but required for testing", sharePool)
		}
	}
	cephFSSection := &cephlcmv1alpha1.CephSharedFilesystem{
		CephFS: []cephlcmv1alpha1.CephFS{},
	}
	if len(cephfsPoolsForShare) > 0 {
		if cd.Spec.SharedFilesystem == nil {
			t.Fatal("CephFS is not found in source Ceph cluster, but required for testing")
		}
		for _, shareCephFsPool := range cephfsPoolsForShare {
			poolFound := false
			for _, cephFS := range cd.Spec.SharedFilesystem.CephFS {
				for _, dataPool := range cephFS.DataPools {
					if fmt.Sprintf("%s-%s", cephFS.Name, dataPool.Name) == shareCephFsPool {
						poolFound = true
						added := false
						for idx, externalCephFs := range cephFSSection.CephFS {
							if cephFS.Name == externalCephFs.Name {
								added = true
								cephFSSection.CephFS[idx].DataPools = append(cephFSSection.CephFS[idx].DataPools, dataPool)
								break
							}
						}
						if !added {
							copyCephFs := cephFS.DeepCopy()
							copyCephFs.DataPools = []cephlcmv1alpha1.CephFSPool{dataPool}
							cephFSSection.CephFS = append(cephFSSection.CephFS, *copyCephFs)
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
	}
	// TODO (degorenko): add support for external RGW as well
	connectionStringCmdTmp := "pelagia-connector --client-name %s --use-cephfs --use-rbd"
	connectionStringCmd := fmt.Sprintf(connectionStringCmdTmp, "admin")
	if !isAdmin {
		f.Step(t, "Create non-admin user in source Ceph cluster")
		testClientName := fmt.Sprintf("non-admin-client-%v", time.Now().Unix())
		connectionStringCmd = fmt.Sprintf(connectionStringCmdTmp, testClientName)
		osdCaps := []string{}
		for _, pool := range rbdPoolsForShare {
			osdCaps = append(osdCaps, "profile rbd pool="+pool)
		}
		if len(cephfsPoolsForShare) > 0 {
			osdCaps = append(osdCaps, "allow rw tag cephfs *=*")
		}
		client := cephlcmv1alpha1.CephClient{
			ClientSpec: cephlcmv1alpha1.ClientSpec{
				Name: testClientName,
				Caps: map[string]string{
					"mgr": "allow r",
					"mon": "allow r, profile role-definer",
					"osd": strings.Join(osdCaps, ", "),
				},
			},
		}
		if len(cephfsPoolsForShare) > 0 {
			client.Caps["mds"] = "allow rw"
		}
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
	externalCephCluster := &cephlcmv1alpha1.CephDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-ceph",
			Namespace: externalClusterNamespace,
		},
		Spec: cephlcmv1alpha1.CephDeploymentSpec{
			Network: cephlcmv1alpha1.CephNetworkSpec{
				ClusterNet: cd.Spec.Network.ClusterNet,
				PublicNet:  cd.Spec.Network.PublicNet,
			},
			External:         true,
			Nodes:            []cephlcmv1alpha1.CephDeploymentNode{},
			Pools:            poolsSection,
			SharedFilesystem: cephFSSection,
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
					deployName, err := prepareExternalTestDeployment(externalCluster, pvcName, 1)
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
					deployName, err := prepareExternalTestDeployment(externalCluster, pvcName, 2)
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

	if len(errMsgs) > 0 {
		t.Fatalf("The following error(s) raised during StorageClass testing: %v", strings.Join(errMsgs, "; "))
	}
	f.Step(t, "Remove external CephDeployment")
	err = externalCluster.RemoveCephDeployment(externalCephCluster.Name)
	if err != nil {
		t.Fatalf("failed to remove external CephDeployment: %v", err)
	}
	t.Logf("Test %s complete sucessfully", t.Name())
}
