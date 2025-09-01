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
	"strconv"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

func TestCreateServerWithVolume(t *testing.T) {
	t.Log("#### e2e test: Create nova server with cinder volume attached")
	defer f.SetupTeardown(t)()

	f.Step(t, "Get testconfig for test case")
	config := f.GetConfigForTestCase(t)

	serverName := fmt.Sprintf("test-server-%d", time.Now().Unix())
	if config["serverName"] != "" {
		serverName = config["serverName"]
	}

	volumeName := fmt.Sprintf("test-volume-%d", time.Now().Unix())
	if config["volumeName"] != "" {
		volumeName = config["volumeName"]
	}

	keypairName := fmt.Sprintf("test-keypair-%d", time.Now().Unix())
	if config["keypairName"] != "" {
		keypairName = config["keypairName"]
	}

	privKeySecretName := fmt.Sprintf("test-private-key-%d", time.Now().Unix())
	if config["privKeySecretName"] != "" {
		privKeySecretName = config["privKeySecretName"]
	}

	required := []string{"flavor", "image", "network", "size"}
	for _, req := range required {
		if _, ok := config[req]; !ok {
			t.Fatalf("Testconfig '%s' config option is not set but required", req)
		}
	}

	size, err := strconv.Atoi(config["size"])
	if err != nil {
		t.Fatalf("Testconfig 'size' config value is incorrect and should be a positive integer: %v", err)
	}

	f.Step(t, "Find keystone pod")
	err = f.TF.ManagedCluster.OpenstackClientSet()
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Create keypair for server")
	privKey, err := f.TF.ManagedCluster.KeypairCreate(keypairName, fmt.Sprintf("/tmp/%s", keypairName))
	if err != nil {
		t.Fatalf("failed to create keypair %s: %v", keypairName, err)
	}
	t.Logf("Keypair %s private key is:\n%v", keypairName, privKey)

	f.Step(t, "Create secret %s/%s with keypair private key", f.TF.ManagedCluster.LcmNamespace, privKeySecretName)

	privKeySecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: f.TF.ManagedCluster.LcmNamespace,
			Name:      privKeySecretName,
		},
		Data: map[string][]byte{
			"key": []byte(privKey),
		},
		Type: corev1.SecretTypeOpaque,
	}
	err = f.TF.ManagedCluster.CreateSecret(&privKeySecret)
	if err != nil {
		t.Fatalf("failed to create private key secret: %v", err)
	}
	t.Logf("Keypair %s private key secret %s/%s created", keypairName, f.TF.ManagedCluster.LcmNamespace, privKeySecretName)

	f.Step(t, "Create nova server with waiting for its availability")
	server, err := f.TF.ManagedCluster.NovaServerCreate(serverName, config["flavor"], keypairName, config["image"], config["network"], true)
	if err != nil {
		t.Fatalf("failed to complete server %s create action: %v", serverName, err)
	}
	t.Logf("Server %s is created: %v", serverName, server)

	f.Step(t, "Create cinder volume on default ceph pool")
	volume, err := f.TF.ManagedCluster.CinderVolumeCreate(volumeName, size, "", true)
	if err != nil {
		t.Fatalf("failed to complete volume %s create action: %v", volumeName, err)
	}
	t.Logf("Volume %s is created: %v", volumeName, volume)

	f.Step(t, "Attach cinder volume to nova server")
	device, err := f.TF.ManagedCluster.NovaServerAddVolume(serverName, volumeName)
	if err != nil {
		t.Fatalf("failed to complete volume %s attach action: %v", volumeName, err)
	}
	t.Logf("Volume %s is attached to server %s to '%s' device", volumeName, serverName, device)

	t.Logf("Test %v successfully passed", t.Name())
}

func TestServerVolumeWriteOp(t *testing.T) {
	t.Log("#### e2e test: Write to server volume via established ssh connection")
	defer f.SetupTeardown(t)()

	f.Step(t, "Get testconfig for test case")
	config := f.GetConfigForTestCase(t)

	required := []string{"serverName", "privKeySecretName", "testString"}
	for _, req := range required {
		if _, ok := config[req]; !ok {
			t.Fatalf("Testconfig '%s' config option is not set but required", req)
		}
	}

	f.Step(t, "Find keystone pod")
	err := f.TF.ManagedCluster.OpenstackClientSet()
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Disable port security for nova server %s", config["serverName"])
	err = f.TF.ManagedCluster.NovaServerDisablePortSecurity(config["serverName"])
	if err != nil {
		t.Fatalf("failed to disable server %s security port: %v", config["serverName"], err)
	}

	f.Step(t, "Get nova server %s and parse its IP address", config["serverName"])
	server, err := f.TF.ManagedCluster.NovaServerShow(config["serverName"])
	if err != nil {
		t.Fatalf("failed to get nova server %s: %v", config["serverName"], err)
	}
	var serverIP string
	for addr := range server.Addresses {
		serverIP = server.Addresses[addr][0]
		break
	}

	sshPodName := fmt.Sprintf("ssh-pod-%d", time.Now().Unix())
	f.Step(t, "Create SSH pod %s/%s", f.TF.ManagedCluster.LcmNamespace, sshPodName)
	sshPod, err := f.TF.ManagedCluster.NewSSHPod(sshPodName, config["privKeySecretName"])
	if err != nil {
		t.Fatalf("failed to create ssh pod %s/%s: %v", f.TF.ManagedCluster.LcmNamespace, sshPodName, err)
	}

	f.Step(t, "Make filesystem on attached volume of server %s", config["serverName"])
	_, err = f.TF.ManagedCluster.ExecSSHPod(serverIP, "mkfs.ext4 /dev/vdb", sshPod)
	if err != nil {
		delErr := f.TF.ManagedCluster.DeleteSSHPod(sshPod.Name)
		if delErr != nil {
			f.TF.Log.Error().Err(delErr).Msg("")
		}
		t.Fatal(err.Error())
	}

	f.Step(t, "Create mount directory for attached volume of server %s", config["serverName"])
	_, err = f.TF.ManagedCluster.ExecSSHPod(serverIP, "mkdir /mnt/test", sshPod)
	if err != nil {
		delErr := f.TF.ManagedCluster.DeleteSSHPod(sshPod.Name)
		if delErr != nil {
			f.TF.Log.Error().Err(delErr).Msg("")
		}
		t.Fatal(err.Error())
	}

	f.Step(t, "Mount attached volume to created directory on server %s", config["serverName"])
	_, err = f.TF.ManagedCluster.ExecSSHPod(serverIP, "mount /dev/vdb /mnt/test", sshPod)
	if err != nil {
		delErr := f.TF.ManagedCluster.DeleteSSHPod(sshPod.Name)
		if delErr != nil {
			f.TF.Log.Error().Err(delErr).Msg("")
		}
		t.Fatal(err.Error())
	}

	f.Step(t, "Create test file on mounted directory and verify it on server %s", config["serverName"])
	err = f.TF.ManagedCluster.UpdateFileSSHPod(serverIP, "/mnt/test/test.file", config["testString"], sshPod)
	if err != nil {
		delErr := f.TF.ManagedCluster.DeleteSSHPod(sshPod.Name)
		if delErr != nil {
			f.TF.Log.Error().Err(delErr).Msg("")
		}
		t.Fatal(err.Error())
	}

	stdout, err := f.TF.ManagedCluster.ExecSSHPod(serverIP, "cat /mnt/test/test.file", sshPod)
	if err != nil {
		delErr := f.TF.ManagedCluster.DeleteSSHPod(sshPod.Name)
		if delErr != nil {
			f.TF.Log.Error().Err(delErr).Msg("")
		}
		t.Fatal(err.Error())
	}
	if !strings.Contains(stdout, config["testString"]) {
		delErr := f.TF.ManagedCluster.DeleteSSHPod(sshPod.Name)
		if delErr != nil {
			f.TF.Log.Error().Err(delErr).Msg("")
		}
		t.Fatalf("test file content from mounted directory does not equal to expected: expected=%s, actual=%v", config["testString"], stdout)
	}

	f.Step(t, "Delete SSH pod")
	err = f.TF.ManagedCluster.DeleteSSHPod(sshPod.Name)
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Logf("Test %v successfully passed", t.Name())
}

func TestServerVolumeReadOp(t *testing.T) {
	t.Log("#### e2e test: Write to server volume via established ssh connection")
	defer f.SetupTeardown(t)()

	f.Step(t, "Get testconfig for test case")
	config := f.GetConfigForTestCase(t)

	required := []string{"serverName", "privKeySecretName", "testString"}
	for _, req := range required {
		if _, ok := config[req]; !ok {
			t.Fatalf("Testconfig '%s' config option is not set but required", req)
		}
	}

	f.Step(t, "Find keystone pod")
	err := f.TF.ManagedCluster.OpenstackClientSet()
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Disable port security for nova server %s", config["serverName"])
	err = f.TF.ManagedCluster.NovaServerDisablePortSecurity(config["serverName"])
	if err != nil {
		t.Fatalf("failed to disable server %s security port: %v", config["serverName"], err)
	}

	f.Step(t, "Get nova server %s and parse its IP address", config["serverName"])
	server, err := f.TF.ManagedCluster.NovaServerShow(config["serverName"])
	if err != nil {
		t.Fatalf("failed to get nova server %s: %v", config["serverName"], err)
	}
	var serverIP string
	for addr := range server.Addresses {
		serverIP = server.Addresses[addr][0]
		break
	}

	sshPodName := fmt.Sprintf("ssh-pod-%d", time.Now().Unix())
	f.Step(t, "Create SSH pod %s/%s", f.TF.ManagedCluster.LcmNamespace, sshPodName)
	sshPod, err := f.TF.ManagedCluster.NewSSHPod(sshPodName, config["privKeySecretName"])
	if err != nil {
		t.Fatalf("failed to create ssh pod %s/%s: %v", f.TF.ManagedCluster.LcmNamespace, sshPodName, err)
	}

	f.Step(t, "Read content from attached volume on server %s and verify it", config["serverName"])
	stdout, err := f.TF.ManagedCluster.ExecSSHPod(serverIP, "cat /mnt/test/test.file", sshPod)
	if err != nil {
		delErr := f.TF.ManagedCluster.DeleteSSHPod(sshPod.Name)
		if delErr != nil {
			f.TF.Log.Error().Err(delErr).Msg("")
		}
		t.Fatal(err.Error())
	}
	if !strings.Contains(stdout, config["testString"]) {
		delErr := f.TF.ManagedCluster.DeleteSSHPod(sshPod.Name)
		if delErr != nil {
			f.TF.Log.Error().Err(delErr).Msg("")
		}
		t.Fatalf("test file content from mounted directory does not equal to expected: expected=%s, actual=%v", config["testString"], stdout)
	}

	f.Step(t, "Delete SSH pod")
	err = f.TF.ManagedCluster.DeleteSSHPod(sshPod.Name)
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Logf("Test %v successfully passed", t.Name())
}

func TestCreateServerFromServerImage(t *testing.T) {
	t.Log("#### e2e test: Create server image and then create server from that image")
	defer f.SetupTeardown(t)()

	f.Step(t, "Get testconfig for test case")
	config := f.GetConfigForTestCase(t)

	required := []string{"serverFrom", "keypairFrom", "flavor", "network"}
	for _, req := range required {
		if _, ok := config[req]; !ok {
			t.Fatalf("Testconfig '%s' config option is not set but required", req)
		}
	}

	f.Step(t, "Find keystone pod")
	err := f.TF.ManagedCluster.OpenstackClientSet()
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Create image from nova server")
	imageName := fmt.Sprintf("test-server-image-%d", time.Now().Unix())
	image, err := f.TF.ManagedCluster.NovaServerImageCreate(config["serverFrom"], imageName, true)
	if err != nil {
		t.Fatalf("failed to complete image %s create for server %s action: %v", imageName, config["serverFrom"], err)
	}
	t.Logf("Server image %s is created: %v", imageName, image)

	f.Step(t, "Create new server from server image")
	serverName := fmt.Sprintf("test-server-%d", time.Now().Unix())
	server, err := f.TF.ManagedCluster.NovaServerCreate(serverName, config["flavor"], config["keypairFrom"], imageName, config["network"], true)
	if err != nil {
		t.Fatalf("failed to complete server %s create action: %v", serverName, err)
	}
	t.Logf("Server %s is created: %v", serverName, server)

	f.Step(t, "Delete new server")
	err = f.TF.ManagedCluster.NovaServerDelete(serverName, true)
	if err != nil {
		t.Fatalf("failed to complete server %s remove action: %v", serverName, err)
	}

	f.Step(t, "Delete server image")
	err = f.TF.ManagedCluster.GlanceImageDelete(imageName, true)
	if err != nil {
		t.Fatalf("failed to complete image %s remove action: %v", imageName, err)
	}

	t.Logf("Test %v successfully passed", t.Name())
}

func TestRebootNovaServer(t *testing.T) {
	t.Log("#### e2e test: Reboot nova server and wait for its availability")
	defer f.SetupTeardown(t)()

	f.Step(t, "Get testconfig for test case")
	config := f.GetConfigForTestCase(t)

	required := []string{"serverName", "rebootMode"}
	for _, req := range required {
		if _, ok := config[req]; !ok {
			t.Fatalf("Testconfig '%s' config option is not set but required", req)
		}
	}

	f.Step(t, "Find keystone pod")
	err := f.TF.ManagedCluster.OpenstackClientSet()
	if err != nil {
		t.Fatal(err)
	}

	var server *f.NovaShow
	if config["rebootMode"] == "soft" || config["rebootMode"] == "" {
		f.Step(t, "Nova instance soft reboot")
		server, err = f.TF.ManagedCluster.NovaServerReboot(config["serverName"], false, true)
		if err != nil {
			t.Fatalf("Nova instance soft reboot failed")
		}
		t.Logf("Server %s is soft rebooted: %v", config["serverName"], server)
	}
	if config["rebootMode"] == "hard" || config["rebootMode"] == "" {
		f.Step(t, "Nova instance hard reboot")
		server, err = f.TF.ManagedCluster.NovaServerReboot(config["serverName"], true, true)
		if err != nil {
			t.Fatalf("Nova instance hard reboot failed")
		}
		t.Logf("Server %s is hard rebooted: %v", config["serverName"], server)
	}

	t.Logf("Test %v successfully passed", t.Name())
}

func TestMigrateServerConfirm(t *testing.T) {
	t.Log("#### e2e test: Migrate server to another host with confirm and verify host is changed")
	defer f.SetupTeardown(t)()

	f.Step(t, "Get testconfig for test case")
	config := f.GetConfigForTestCase(t)

	required := []string{"serverName"}
	for _, req := range required {
		if _, ok := config[req]; !ok {
			t.Fatalf("Testconfig '%s' config option is not set but required", req)
		}
	}

	f.Step(t, "Find keystone pod")
	err := f.TF.ManagedCluster.OpenstackClientSet()
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Get nova server %s and save its host", config["serverName"])
	server, err := f.TF.ManagedCluster.NovaServerShow(config["serverName"])
	if err != nil {
		t.Fatalf("failed to get nova server %s: %v", config["serverName"], err)
	}
	serverHostOriginal := server.Host

	f.Step(t, "Run nova server %s migration", config["serverName"])
	serverM, err := f.TF.ManagedCluster.NovaServerMigrate(config["serverName"], true)
	if err != nil {
		t.Fatalf("failed to migrate nova server %s: %v", config["serverName"], err)
	}

	f.Step(t, "Verify nova server %s changed its host during migration", config["serverName"])
	serverHostMigrate := serverM.Host
	if serverHostOriginal == serverHostMigrate {
		t.Fatalf("nova server %s didn't change its host on migrate: original=%s, migrate=%s", config["serverName"], serverHostOriginal, serverHostMigrate)
	}

	serverC, err := f.TF.ManagedCluster.NovaServerMigrateAction(config["serverName"], "confirm", true)
	if err != nil {
		t.Fatalf("failed to migrate confirm nova server %s: %v", config["serverName"], err)
	}
	if serverHostMigrate != serverC.Host {
		t.Fatalf("nova server %s host on migrate confirm '%s' is not equal to host on migrate '%s'", config["serverName"], serverC.Host, serverHostMigrate)
	}
	if serverHostOriginal == serverC.Host {
		t.Fatalf("nova server %s didn't change its host on migrate confirm: original=%s, migrate confirm=%s", config["serverName"], serverHostOriginal, serverC.Host)
	}

	t.Logf("Test %v successfully passed", t.Name())
}

func TestMigrateServerRevert(t *testing.T) {
	t.Log("#### e2e test: Migrate server to another host with revert and verify host is not changed")
	defer f.SetupTeardown(t)()

	f.Step(t, "Get testconfig for test case")
	config := f.GetConfigForTestCase(t)

	required := []string{"serverName"}
	for _, req := range required {
		if _, ok := config[req]; !ok {
			t.Fatalf("Testconfig '%s' config option is not set but required", req)
		}
	}

	f.Step(t, "Find keystone pod")
	err := f.TF.ManagedCluster.OpenstackClientSet()
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Get nova server %s and save its host", config["serverName"])
	server, err := f.TF.ManagedCluster.NovaServerShow(config["serverName"])
	if err != nil {
		t.Fatalf("failed to get nova server %s: %v", config["serverName"], err)
	}
	serverHostOriginal := server.Host

	f.Step(t, "Run nova server %s migration", config["serverName"])
	serverM, err := f.TF.ManagedCluster.NovaServerMigrate(config["serverName"], true)
	if err != nil {
		t.Fatalf("failed to migrate nova server %s: %v", config["serverName"], err)
	}

	f.Step(t, "Verify nova server %s changed its host during migration", config["serverName"])
	serverHostMigrate := serverM.Host
	if serverHostOriginal == serverHostMigrate {
		t.Fatalf("nova server %s didn't change its host on migrate: original=%s, migrate=%s", config["serverName"], serverHostOriginal, serverHostMigrate)
	}

	serverC, err := f.TF.ManagedCluster.NovaServerMigrateAction(config["serverName"], "revert", true)
	if err != nil {
		t.Fatalf("failed to migrate revert nova server %s: %v", config["serverName"], err)
	}
	if serverHostMigrate == serverC.Host {
		t.Fatalf("nova server %s host on migrate confirm '%s' is equal to host on migrate '%s'", config["serverName"], serverC.Host, serverHostMigrate)
	}
	if serverHostOriginal != serverC.Host {
		t.Fatalf("nova server %s changed its host on migrate revert: original=%s, migrate revert=%s", config["serverName"], serverHostOriginal, serverC.Host)
	}

	t.Logf("Test %v successfully passed", t.Name())
}

func TestDeleteServerWithVolume(t *testing.T) {
	t.Log("#### e2e test: Delete nova server and attached cinder volume")
	defer f.SetupTeardown(t)()

	f.Step(t, "Get testconfig for test case")
	config := f.GetConfigForTestCase(t)

	required := []string{"serverName", "volumeName", "keypairName", "privKeySecretName"}
	for _, req := range required {
		if _, ok := config[req]; !ok {
			t.Fatalf("Testconfig '%s' config option is not set but required", req)
		}
	}

	f.Step(t, "Find keystone pod")
	err := f.TF.ManagedCluster.OpenstackClientSet()
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Detach cinder volume from nova server")
	err = f.TF.ManagedCluster.NovaServerRemoveVolume(config["serverName"], config["volumeName"])
	if err != nil {
		t.Fatalf("failed to complete volume %s detach action: %v", config["volumeName"], err)
	}

	f.Step(t, "Remove cinder volume with waiting for deletion")
	err = f.TF.ManagedCluster.CinderVolumeDelete(config["volumeName"], true)
	if err != nil {
		t.Fatalf("failed to complete volume %s remove action: %v", config["volumeName"], err)
	}

	f.Step(t, "Remove nova server with waiting for delete")
	err = f.TF.ManagedCluster.NovaServerDelete(config["serverName"], true)
	if err != nil {
		t.Fatalf("failed to complete server %s remove action: %v", config["serverName"], err)
	}

	f.Step(t, "Remove keypair")
	err = f.TF.ManagedCluster.KeypairDelete(config["keypairName"], fmt.Sprintf("/tmp/%s", config["keypairName"]))
	if err != nil {
		t.Fatalf("failed to complete keypair %s remove action: %v", config["keypairName"], err)
	}

	f.Step(t, "Delete private key secret")
	err = f.TF.ManagedCluster.DeleteSecret(config["privKeySecretName"], f.TF.ManagedCluster.LcmNamespace)
	if err != nil {
		t.Fatalf("failed to delete private key secret: %v", err)
	}

	t.Logf("Test %v successfully passed", t.Name())
}

func TestSwiftCreateContainerUploadObject(t *testing.T) {
	t.Log("#### e2e test: Create swift container and upload test file object to it")
	defer f.SetupTeardown(t)()

	f.Step(t, "Get testconfig for test case")
	config := f.GetConfigForTestCase(t)

	required := []string{"containerName", "objectName", "testString"}
	for _, req := range required {
		if _, ok := config[req]; !ok {
			t.Fatalf("Testconfig '%s' config option is not set but required", req)
		}
	}

	f.Step(t, "Find keystone pod")
	err := f.TF.ManagedCluster.OpenstackClientSet()
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Create swift container")
	err = f.TF.ManagedCluster.SwiftContainerCreate(config["containerName"])
	if err != nil {
		t.Fatalf("failed to create swift container %s: %v", config["containerName"], err)
	}

	f.Step(t, "Upload test object to created swift container")
	err = f.TF.ManagedCluster.SwiftObjectUpload(config["containerName"], config["objectName"], config["testString"])
	if err != nil {
		t.Fatalf("failed to upload object %s to swift container %s: %v", config["objectName"], config["containerName"], err)
	}

	t.Logf("Test %v successfully passed", t.Name())
}

func TestSwiftDownloadObjectDeleteContainer(t *testing.T) {
	t.Log("#### e2e test: Download test file object from swift container and delete container after")
	defer f.SetupTeardown(t)()

	f.Step(t, "Get testconfig for test case")
	config := f.GetConfigForTestCase(t)

	required := []string{"containerName", "objectName", "testString"}
	for _, req := range required {
		if _, ok := config[req]; !ok {
			t.Fatalf("Testconfig '%s' config option is not set but required", req)
		}
	}

	f.Step(t, "Find keystone pod")
	err := f.TF.ManagedCluster.OpenstackClientSet()
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Download object from swift container")
	actual, err := f.TF.ManagedCluster.SwiftObjectDownload(config["containerName"], config["objectName"])
	if err != nil {
		t.Fatalf("failed to download object %s from swift container %s: %v", config["objectName"], config["containerName"], err)
	}
	if !strings.Contains(actual, config["testString"]) {
		t.Fatalf("actual object content is not included in expected: expected='%s', actual='%s'", config["testString"], actual)
	}

	f.Step(t, "Delete swift container with all objects in it")
	err = f.TF.ManagedCluster.SwiftContainerDelete(config["containerName"])
	if err != nil {
		t.Fatalf("failed to delete swift container %s: %v", config["containerName"], err)
	}

	t.Logf("Test %v successfully passed", t.Name())
}
