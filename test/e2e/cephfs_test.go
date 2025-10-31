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
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

var (
	deploymentContainersName = "simple-container"
	mountPathForDeployment   = "/mnt"
)

func prepareTestDeployment(cephFsTestDeploymentName, cephFsPVCName, storageClassName, testImage, namespace string) error {
	f.TF.Log.Info().Msgf("Creating testing PVC '%s'", cephFsPVCName)
	testPvc := v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cephFsPVCName,
			Namespace: namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteMany,
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
		return errors.Wrapf(err, "failed to create PVC %s/%s", namespace, cephFsPVCName)
	}

	f.TF.Log.Info().Msgf("Creating testing deployment '%s'", cephFsTestDeploymentName)
	replicas := int32(3)
	testDeploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cephFsTestDeploymentName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": cephFsTestDeploymentName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": cephFsTestDeploymentName,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": cephFsTestDeploymentName,
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
							Name:  deploymentContainersName,
							Image: testImage,
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      cephFsPVCName,
									MountPath: mountPathForDeployment,
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
							Name: cephFsPVCName,
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									ClaimName: cephFsPVCName,
									ReadOnly:  false,
								},
							},
						},
					},
				},
			},
		},
	}
	err = f.TF.ManagedCluster.CreateDeployment(&testDeploy)
	if err != nil {
		return errors.Wrapf(err, "failed to create deployment %s/%s", namespace, cephFsTestDeploymentName)
	}
	err = f.TF.ManagedCluster.WaitDeploymentReady(testDeploy.Name, testDeploy.Namespace)
	if err != nil {
		return errors.Wrapf(err, "deployment %s/%s is not ready", namespace, cephFsTestDeploymentName)
	}
	return nil
}

func writeContent(cephFsTestDeploymentName, namespace string) (string, error) {
	label := "app=" + cephFsTestDeploymentName
	pods, err := f.TF.ManagedCluster.ListPods(namespace, label)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get pods with label '%s' in ns '%s'", label, namespace)
	}
	content := ""
	for idx, pod := range pods {
		cmd := fmt.Sprintf("touch %s/%d", mountPathForDeployment, idx)
		_, _, err := f.TF.ManagedCluster.RunPodCommand(cmd, deploymentContainersName, &pod)
		if err != nil {
			f.TF.Log.Error().Msgf("Failed to run cmd: '%s', in pod: '%s', in container: '%s'", cmd, pod.Name, deploymentContainersName)
			return "", err
		}
		content += fmt.Sprintf("%d\n", idx)
	}
	return content, nil
}

func getContent(cephFsTestDeploymentName, namespace string) ([]string, error) {
	label := "app=" + cephFsTestDeploymentName
	pods, err := f.TF.ManagedCluster.ListPods(namespace, label)
	if err != nil {
		return make([]string, 0), errors.Wrapf(err, "failed to get pods with label '%s' in ns '%s'", label, namespace)
	}
	content := make([]string, len(pods))
	for idx, pod := range pods {
		cmd := fmt.Sprintf("ls -1 %s", mountPathForDeployment)
		stdOut, _, err := f.TF.ManagedCluster.RunPodCommand(cmd, deploymentContainersName, &pod)
		if err != nil {
			f.TF.Log.Error().Err(err).Msgf("Failed to run cmd: '%s', in pod: '%s', in container: '%s'", cmd, pod.Name, deploymentContainersName)
			return make([]string, 0), err
		}
		content[idx] = stdOut
	}
	return content, nil
}

func TestCephFS(t *testing.T) {
	t.Log("e2e test: cephfs using in cluster")
	defer f.SetupTeardown(t)()

	f.Step(t, "Get test config options")
	testConfig := f.GetConfigForTestCase(t)
	if _, ok := testConfig["cephFsConfig"]; !ok {
		t.Fatal("Test config does not contain 'cephFsConfig' key")
	}
	storageClasses, sharedFS, err := f.ReadSharedFilesystemConfig(testConfig["cephFsConfig"])
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Verify whether CephFS deployed on cluster or not")
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	toUpdate := false
	if cd.Spec.SharedFilesystem != nil {
		for _, newCephFS := range sharedFS.CephFS {
			found := false
			for idx, cephFS := range cd.Spec.SharedFilesystem.CephFS {
				if cephFS.Name == newCephFS.Name {
					f.TF.Log.Warn().Msgf("found present CephFS with the same name as for e2e test: %s", cephFS.Name)
					found = true
					// check data pools required for tests present
					for _, dataPool := range newCephFS.DataPools {
						foundPool := false
						for _, presentDataPool := range cephFS.DataPools {
							if dataPool.Name == presentDataPool.Name {
								foundPool = true
							}
						}
						if !foundPool {
							toUpdate = true
							cd.Spec.SharedFilesystem.CephFS[idx].DataPools = append(cd.Spec.SharedFilesystem.CephFS[idx].DataPools, dataPool)
						}
					}
					break
				}
			}
			if !found {
				toUpdate = true
				cd.Spec.SharedFilesystem.CephFS = append(cd.Spec.SharedFilesystem.CephFS, newCephFS)
			}
		}
	} else {
		cd.Spec.SharedFilesystem = sharedFS
		toUpdate = true
	}
	for _, node := range cd.Spec.Nodes {
		if lcmcommon.Contains(node.Roles, "mon") && !lcmcommon.Contains(node.Roles, "mds") {
			node.Roles = append(node.Roles, "mds")
			toUpdate = true
		}
	}
	if toUpdate {
		f.Step(t, "update cephdeployment with cephFS changes")
		err = f.UpdateCephDeploymentSpec(cd, true)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		f.Step(t, "update cephdeployment is not required (cephFS and mds roles are present)")
	}

	testImage := f.TF.ManagedCluster.LcmConfig.DeployParams.CephImage
	for _, storageClass := range storageClasses {
		f.Step(t, "Prepare test deployment and PVC for storage class '%s'", storageClass)
		cephFsTestDeploymentName := fmt.Sprintf("cephfs-test-deployment-%v", time.Now().Unix())
		cephFsPVCName := fmt.Sprintf("cephfs-pvc-%v", time.Now().Unix())

		defer func() {
			if !f.TF.TestConfig.Settings.KeepAfter {
				f.Step(t, "Droping testing deployment and PVC for storage class '%s'", storageClass)
				err = f.TF.ManagedCluster.DeleteDeployment(cephFsTestDeploymentName, f.TF.TestConfig.Settings.Namespace)
				if err != nil {
					t.Fatal(err)
				}
				err = f.TF.ManagedCluster.DeletePVC(cephFsPVCName, f.TF.TestConfig.Settings.Namespace)
				if err != nil {
					t.Fatal(err)
				}
			}
		}()

		err = prepareTestDeployment(cephFsTestDeploymentName, cephFsPVCName, storageClass, testImage, f.TF.TestConfig.Settings.Namespace)
		if err != nil {
			t.Fatal(err)
		}

		f.Step(t, "Write test content for deployment '%s'", cephFsTestDeploymentName)
		content, err := writeContent(cephFsTestDeploymentName, f.TF.TestConfig.Settings.Namespace)
		if err != nil {
			t.Fatal(err)
		}

		newReplicasCount := 5
		f.Step(t, "Scale up test deployment '%s' to %d replicas", cephFsTestDeploymentName, newReplicasCount)
		err = f.TF.ManagedCluster.ScaleDeployment(cephFsTestDeploymentName, f.TF.TestConfig.Settings.Namespace, int32(newReplicasCount))
		if err != nil {
			t.Fatal(err)
		}
		err = f.TF.ManagedCluster.WaitDeploymentReady(cephFsTestDeploymentName, f.TF.TestConfig.Settings.Namespace)
		if err != nil {
			t.Fatal(err)
		}
		f.Step(t, "Check test content for deployment '%s'", cephFsTestDeploymentName)
		contents, err := getContent(cephFsTestDeploymentName, f.TF.TestConfig.Settings.Namespace)
		if err != nil {
			t.Fatal(err)
		}

		t.Run("check content from pods", func(t *testing.T) {
			for _, contentFromPod := range contents {
				assert.Equal(t, content, contentFromPod)
			}
			assert.Equal(t, newReplicasCount, len(contents))
		})
	}

	t.Log("e2e test: cephfs using in cluster completed")
}

func TestCephFSManila(t *testing.T) {
	err := f.BaseSetup(t)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Get test config options")
	testConfig := f.GetConfigForTestCase(t)
	if _, ok := testConfig["serverPrivateKey"]; !ok {
		t.Fatal("Test config does not contain 'serverPrivateKey' key")
	}
	serverPrivateKey := testConfig["serverPrivateKey"]

	var ok bool
	var defaults string
	if defaults, ok = testConfig["stackDefaultsRaw"]; !ok || defaults == "" {
		var defaultsPath string
		if defaultsPath, ok = testConfig["stackDefaultsPath"]; !ok || defaultsPath == "" {
			t.Fatal("Test config does not contain neither 'stackDefaultsRaw' nor 'stackDefaultsPath' keys")
		}
		defaultsByte, err := os.ReadFile(defaultsPath)
		if err != nil {
			t.Fatalf("failed to read file from %s path: %v", defaultsPath, err)
		}
		defaults = string(defaultsByte)
	}
	var template string
	if template, ok = testConfig["stackTemplateRaw"]; !ok || template == "" {
		var templatePath string
		if templatePath, ok = testConfig["stackTemplatePath"]; !ok || templatePath == "" {
			t.Fatal("Test config does not contain neither 'stackTemplateRaw' nor 'stackTemplatePath' keys")
		}
		templateByte, err := os.ReadFile(templatePath)
		if err != nil {
			t.Fatalf("failed to read file from %s path: %v", templatePath, err)
		}
		template = string(templateByte)
	}

	// set names and params
	privKeySecretName := fmt.Sprintf("test-vm-private-key-%d", time.Now().Unix())
	manilaShareName := fmt.Sprintf("test-share-%d", time.Now().Unix())
	stackName := fmt.Sprintf("test-stack-%d", time.Now().Unix())
	sshPodName := fmt.Sprintf("ssh-pod-%d", time.Now().Unix())
	skipOsdplTeardown := true

	f.Step(t, "Get osdpl object")
	osdpl, err := f.TF.ManagedCluster.GetOpenstackDeployment()
	if err != nil {
		terr := f.Teardown()
		if terr != nil {
			t.Logf("Teardown failed with error: %v", terr)
		}
		t.Fatal(err)
	}
	orgObject := osdpl.DeepCopy().Object

	defer f.CustomTeardown(t, func() error {
		err = f.TF.ManagedCluster.DeleteManilaShare(manilaShareName)
		if err != nil {
			f.TF.Log.Error().Err(err).Msgf("failed to delete manila share %s", manilaShareName)
		}
		err = f.TF.ManagedCluster.DeleteManilaShareType()
		if err != nil {
			f.TF.Log.Error().Err(err).Msg("failed to delete manila share type")
		}
		err = f.TF.ManagedCluster.DeleteSecret(privKeySecretName, f.TF.ManagedCluster.LcmNamespace)
		if err != nil {
			f.TF.Log.Error().Err(err).Msgf("failed to delete %s/%s secret", f.TF.ManagedCluster.LcmNamespace, privKeySecretName)
		}
		err = f.TF.ManagedCluster.DeleteSSHPod(sshPodName)
		if err != nil {
			f.TF.Log.Error().Err(err).Msgf("failed to delete ssh pod %s", sshPodName)
		}
		err = f.TF.ManagedCluster.HeatStackDelete(stackName)
		if err != nil {
			f.TF.Log.Error().Err(err).Msgf("failed to delete stack %s", stackName)
		}
		if !skipOsdplTeardown {
			err = f.TF.ManagedCluster.UpdateOpenstackDeployment(orgObject, true)
			if err != nil {
				return errors.Wrap(err, "failed to restore osdpl object")
			}
		}
		return nil
	})

	f.Step(t, "Prepare environment for test: Create ssh pod private key secret")
	privKeySecret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: f.TF.ManagedCluster.LcmNamespace,
			Name:      privKeySecretName,
		},
		Data: map[string][]byte{
			"key": []byte(serverPrivateKey),
		},
		Type: v1.SecretTypeOpaque,
	}
	err = f.TF.ManagedCluster.CreateSecret(&privKeySecret)
	if err != nil {
		t.Fatalf("failed to create private key secret: %v", err)
	}

	f.Step(t, "Prepare environment for test: Port forward from ceph public network to openstack instances")
	sshPod, err := f.TF.ManagedCluster.NewSSHPod(sshPodName, privKeySecretName)
	if err != nil {
		t.Fatalf("failed to create ssh pod %s/%s: %v", f.TF.ManagedCluster.LcmNamespace, sshPodName, err)
	}
	// find vs node
	vsIP := ""
	nodes, err := f.TF.ManagedCluster.ListNodes()
	if err != nil {
		t.Fatalf("failed to list nodes for vs node: %v", err)
	}
	for _, node := range nodes {
		if strings.Contains(node.Name, "-vs-") {
			for _, addr := range node.Status.Addresses {
				if addr.Type == v1.NodeInternalIP {
					vsIP = addr.Address
					break
				}
			}
			break
		}
	}
	if vsIP == "" {
		t.Fatal("failed to find IP address for vs node")
	}
	// masquerade
	iface, err := f.TF.ManagedCluster.ExecSSHPod(vsIP, "ip -br -4 a sh | grep 10.12.1 | awk '{print $1}'", sshPod)
	if err != nil {
		t.Fatalf("failed to \"ip -br -4 a sh | grep 10.12.1 | awk '{print $1}'\" inside vs node: %v", err)
	}
	_, err = f.TF.ManagedCluster.ExecSSHPod(vsIP, fmt.Sprintf("iptables -t nat -A POSTROUTING -o %s -j MASQUERADE", iface), sshPod)
	if err != nil {
		t.Fatalf("failed to 'iptables -t nat -A POSTROUTING -o %s -j MASQUERADE' inside vs node: %v", iface, err)
	}
	err = f.TF.ManagedCluster.DeleteSSHPod(sshPodName)
	if err != nil {
		t.Fatalf("failed to delete ssh pod %s: %v", sshPodName, err)
	}
	sshPodName = fmt.Sprintf("ssh-pod-%d", time.Now().Unix())

	f.Step(t, "Prepare environment for test: find keystone pod")
	err = f.TF.ManagedCluster.OpenstackClientSet()
	if err != nil {
		t.Fatal(err)
	}
	f.Step(t, "Prepare environment for test: Copy defaults.env and template.yaml inside keystone pod")
	_, stderr, err := f.TF.ManagedCluster.RunPodCommandWithContent("tee /tmp/defaults.env", f.TF.ManagedCluster.OpenstackClient.Container, f.TF.ManagedCluster.OpenstackClient.KeystonePod, defaults)
	if err != nil {
		errMsg := "failed to update file on path /tmp/defaults.env inside keystone pod"
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %v)", stderr)
		}
		t.Fatalf("%s: %v", err, errMsg)
	}
	_, stderr, err = f.TF.ManagedCluster.RunPodCommandWithContent("tee /tmp/template.yaml", f.TF.ManagedCluster.OpenstackClient.Container, f.TF.ManagedCluster.OpenstackClient.KeystonePod, template)
	if err != nil {
		errMsg := "failed to update file on path /tmp/template.yaml inside keystone pod"
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %v)", stderr)
		}
		t.Fatalf("%s: %v", err, errMsg)
	}

	f.Step(t, "Prepare environment for test: Create test-stack with 2 servers")
	err = f.TF.ManagedCluster.HeatStackCreate(stackName, "/tmp/defaults.env", "/tmp/template.yaml")
	if err != nil {
		t.Fatalf("failed to create stack %s: %v", stackName, err)
	}

	f.Step(t, "Prepare environment for test: Obtain servers IP addresses")
	server1, err := f.TF.ManagedCluster.HeatStackOutputShow(stackName, "instance1_ip")
	if err != nil {
		t.Fatal("failed to obtain server 1 ip address")
	}
	server1IP := server1.Value
	server2, err := f.TF.ManagedCluster.HeatStackOutputShow(stackName, "instance2_ip")
	if err != nil {
		t.Fatal("failed to obtain server 2 ip address")
	}
	server2IP := server2.Value

	f.Step(t, "Verify whether CephFS deployed on cluster or not")
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range cd.Spec.Nodes {
		if lcmcommon.Contains(node.Roles, "mon") && !lcmcommon.Contains(node.Roles, "mds") {
			node.Roles = append(node.Roles, "mds")
		}
	}
	f.Step(t, "Enable CephFS for Manila in Ceph cluster")
	cephFSName := fmt.Sprintf("shared-cephfs-%d", time.Now().Unix())
	cephFS := cephlcmv1alpha1.CephFS{
		Name: cephFSName,
		DataPools: []cephlcmv1alpha1.CephFSPool{
			{
				Name: "data-pool",
				CephPoolSpec: cephlcmv1alpha1.CephPoolSpec{
					DeviceClass:   "hdd",
					FailureDomain: "host",
					Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
						Size: 2,
					},
				},
			},
		},
		MetadataPool: cephlcmv1alpha1.CephPoolSpec{
			DeviceClass:   "hdd",
			FailureDomain: "host",
			Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
				Size: 2,
			},
		},
		MetadataServer: cephlcmv1alpha1.CephMetadataServer{
			ActiveCount: 1,
		},
	}
	if cd.Spec.SharedFilesystem != nil && len(cd.Spec.SharedFilesystem.CephFS) > 0 {
		cd.Spec.SharedFilesystem.CephFS = append(cd.Spec.SharedFilesystem.CephFS, cephFS)
	} else {
		cd.Spec.SharedFilesystem = &cephlcmv1alpha1.CephSharedFilesystem{CephFS: []cephlcmv1alpha1.CephFS{cephFS}}
	}
	f.Step(t, "update CephDeployment with Manila CephFS changes")
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Enable snapshot support in CephFS")
	_, err = f.TF.ManagedCluster.RunCephToolsCommand(fmt.Sprintf("ceph fs set %s allow_new_snaps true", cephFSName))
	if err != nil {
		t.Fatalf("failed to exec enabling snapshot support: %v", err)
	}

	/* Now we are using official Rockoon manila installation
	   Comment but not remove for further potential usage
	f.Step(t, "Obtain Manila client keyring")
	manilaClientName := "manila"
	manilaKey := ""
	err = wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
		manilaKey, stderr, err = f.TF.ManagedCluster.RunRookPodCommand("", fmt.Sprintf("ceph auth get-key client.%s", manilaClientName), f.CephClusterNamespace)
		if err != nil {
			f.TF.Log.Error().Err(err).Msg("")
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("failed to obtain manila client keyring: %v", err)
	}
	*/

	f.Step(t, "Obtain Ceph mon endpoints")
	monMap, err := f.TF.ManagedCluster.GetConfigMap(lcmcommon.MonMapConfigMapName, f.TF.ManagedCluster.LcmConfig.RookNamespace)
	if err != nil {
		t.Fatalf("failed to obtain monmap configmap: %v", err)
	}

	monmapData := monMap.Data
	reg, _ := regexp.Compile("([a-z]=)")
	monmapString := reg.ReplaceAllString(string(monmapData["data"]), "")
	monIPs := strings.Split(monmapString, ",")
	sort.Strings(monIPs)
	monEndpoints := strings.Join(monIPs, ",")

	/* Now we are using official Rockoon manila installation
	       Comment but not remove for further potential usage
		f.Step(t, "Update osdpl with manila-cephfs config")
		manilaConfig := &f.ManilaConfigParams{
			CephFsName:       cephFSName,
			CephFsClientName: manilaClientName,
			CephFsClientKey:  manilaKey,
			MonEndpoints:     monEndpoints,
		}
		newData, err := f.BuildManilaCephFSDriver(osdpl, manilaConfig)
		if err != nil {
			t.Fatal(err)
		}
		skipOsdplTeardown = false
		err = f.UpdateOpenstackDeployment(newData, true)
		if err != nil {
			t.Fatal(err)
		}

		f.Step(t, "Wait for manila-share sts become Ready")
		err = wait.PollImmediate(15*time.Second, 15*time.Minute, func() (done bool, err error) {
			sts, err := f.TF.ManagedCluster.GetStatefulset("manila-share-cephfs", "openstack")
			if err != nil {
				f.TF.Log.Error().Err(err).Msgf("failed to get openstack/manila-share-cephfs-0 sts")
				return false, nil
			}
			return f.IsStatefulsetReady(sts), nil
		})
		if err != nil {
			t.Fatalf("Failed to wait for openstack/manila-share-cephfs-0 sts becomes Ready: %v", err)
		}
	*/

	f.Step(t, "Verify manila-share in share service list")
	stdout, stderr, err := f.TF.ManagedCluster.RunOpenstackCommand("openstack share service list")
	if err != nil {
		errMsg := "failed to exec 'openstack share service list'"
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %s)", stderr)
		}
		t.Fatalf("%s: %v", errMsg, err)
	}

	if !strings.Contains(stdout, "manila-share") {
		t.Fatal("expected manila-share in share service list not found")
	}

	f.Step(t, "Create Manila share")
	shareClientName := fmt.Sprintf("manila-cephfs-%d", time.Now().Unix())
	err = f.TF.ManagedCluster.CreateManilaShareType()
	if err != nil {
		t.Fatalf("failed to create manila share type: %v", err)
	}
	location, shareClientKey, err := f.TF.ManagedCluster.CreateManilaShare(manilaShareName, shareClientName)
	if err != nil {
		t.Fatalf("failed to create manila share: %v", err)
	}

	f.Step(t, "Create SSH pod %s/%s", f.TF.ManagedCluster.LcmNamespace, sshPodName)
	sshPod, err = f.TF.ManagedCluster.NewSSHPod(sshPodName, privKeySecretName)
	if err != nil {
		t.Fatalf("failed to create ssh pod %s/%s: %v", f.TF.ManagedCluster.LcmNamespace, sshPodName, err)
	}

	for _, server := range []string{server1IP, server2IP} {
		f.Step(t, "Write ceph config files to server '%s'", server)
		_, err = f.TF.ManagedCluster.ExecSSHPod(server, "mkdir -p /etc/ceph", sshPod)
		if err != nil {
			t.Fatalf("failed to create /etc/ceph dir inside VM: %v", err)
		}

		keyringFile := fmt.Sprintf(`[client.%s]
    key = %s
`, shareClientName, shareClientKey)
		err = f.TF.ManagedCluster.UpdateFileSSHPod(server, fmt.Sprintf("/etc/ceph/ceph.client.%s.keyring", shareClientName), keyringFile, sshPod)
		if err != nil {
			t.Fatalf("failed to create keyring file inside VM: %v", err)
		}

		cephConfFile := fmt.Sprintf(`[client]
    client quota = true
    mon host = %s
`, monEndpoints)
		err = f.TF.ManagedCluster.UpdateFileSSHPod(server, "/etc/ceph/ceph.conf", cephConfFile, sshPod)
		if err != nil {
			t.Fatalf("failed to create ceph.conf file inside VM: %v", err)
		}

		f.Step(t, "Mount share inside server '%s'", server)
		_, err = f.TF.ManagedCluster.ExecSSHPod(server, "mkdir -p /nfs/test", sshPod)
		if err != nil {
			t.Fatalf("failed to create /nfs/test dir inside VM: %v", err)
		}

		mountCmd := fmt.Sprintf(
			"sudo ceph-fuse /nfs/test --id=%s --conf=/etc/ceph/ceph.conf --keyring=/etc/ceph/ceph.client.%s.keyring --client-mountpoint=%s",
			shareClientName, shareClientName, location)
		_, err = f.TF.ManagedCluster.ExecSSHPod(server, mountCmd, sshPod)
		if err != nil {
			t.Fatalf("failed to mount share to /nfs/test dir inside VM: %v", err)
		}
		t.Logf("Manila share successfully mounted to /nfs/test inside VM")
	}

	for idx, server := range []string{server1IP, server2IP} {
		f.Step(t, "Verify share mount is accessible for WRITE operations at server %s '%s'", fmt.Sprintf("%d", idx+1), server)
		_, err = f.TF.ManagedCluster.ExecSSHPod(server, fmt.Sprintf("touch /nfs/test/server%d-file", idx+1), sshPod)
		if err != nil {
			t.Fatalf("failed to create /nfs/test/server%d-file file inside VM: %v", idx+1, err)
		}
		testFile := fmt.Sprintf("server%d test string", idx+1)
		err = f.TF.ManagedCluster.UpdateFileSSHPod(server, fmt.Sprintf("touch /nfs/test/server%d-file", idx+1), testFile, sshPod)
		if err != nil {
			t.Fatalf("failed to create /nfs/test/server%d-file file inside VM: %v", idx+1, err)
		}
		t.Logf("Test file /nfs/test/server%d-file created inside VM '%s'", idx+1, server)
	}

	for idx, server := range []string{server1IP, server2IP} {
		f.Step(t, "Verify share mount is accessible for READ operations at server %s '%s'", fmt.Sprintf("%d", idx+1), server)
		output, err := f.TF.ManagedCluster.ExecSSHPod(server, fmt.Sprintf("cat /nfs/test/server%d-file", ((idx+1)%2)+1), sshPod)
		if err != nil {
			t.Fatalf("failed to 'cat cat /nfs/test/server%d-file' inside VM: %v", ((idx+1)%2)+1, err)
		}
		expected := fmt.Sprintf("server%d test string", ((idx+1)%2)+1)
		if expected != output {
			t.Fatalf("expected output '%s' not equals to actual output '%s'", expected, output)
		}
		t.Logf("Test file /nfs/test/server%d-file output equals to expected '%s' inside VM '%s'", ((idx+1)%2)+1, expected, server)
	}

	for idx, server := range []string{server1IP, server2IP} {
		f.Step(t, "Cleanup share mount from server %s '%s'", fmt.Sprintf("%d", idx+1), server)
		_, err = f.TF.ManagedCluster.ExecSSHPod(server, "umount /nfs/test", sshPod)
		if err != nil {
			t.Fatalf("failed to umount /nfs/test from VM: %v", err)
		}
		_, err = f.TF.ManagedCluster.ExecSSHPod(server, "rm -r /nfs/test", sshPod)
		if err != nil {
			t.Fatalf("failed to rm -r /nfs/test from VM: %v", err)
		}
		_, err = f.TF.ManagedCluster.ExecSSHPod(server, "rm -r /etc/ceph", sshPod)
		if err != nil {
			t.Fatalf("failed to rm -r /etc/ceph from VM: %v", err)
		}
		t.Logf("Manila share successfully cleaned up from VM")
	}

	t.Logf("Test %v successfully passed", t.Name())
}
