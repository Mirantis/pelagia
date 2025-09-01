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
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

func TestRgwSSERockoon(t *testing.T) {
	t.Log("#### e2e test: save object Ceph Object Storage RGW with SSE backend")
	defer f.SetupTeardown(t)()

	f.Step(t, "Verify there is object storage in ceph cluster already created")
	rgws, err := f.TF.ManagedCluster.ListCephObjectStore()
	assert.Nil(t, err)
	if len(rgws) == 0 {
		t.Skip("There is no CephObjectStore created in current Ceph Cluster, skip test")
	} else if len(rgws) > 1 {
		t.Fatal("There are several CephObjectStore created in current Ceph Cluster, which is invalid")
	}

	secret, err := f.TF.ManagedCluster.GetSecret("openstack-rgw-creds", f.TF.ManagedCluster.LcmConfig.DeployParams.OpenstackCephSharedNamespace)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			t.Skip("There is no Openstack installed, skipping step")
		} else {
			t.Fatalf("failed to get secret '%s/%s': %v", f.TF.ManagedCluster.LcmConfig.DeployParams.OpenstackCephSharedNamespace, "openstack-rgw-creds", err)
		}

	}
	if string(secret.Data["barbican_url"]) == "" {
		t.Skip("There is no Openstack Barbican installed, skipping step")
	}

	f.Step(t, "Create KMS secret")
	openstackNamespace := "openstack"
	keystonePod, err := f.TF.ManagedCluster.GetPodByLabel(openstackNamespace, "application=keystone,component=client")
	if err != nil {
		t.Fatal(err)
	}
	keystoneContainerName := "keystone-client"
	cmd := "openstack secret order create --name rados_e2e_key --algorithm aes --mode ctr --bit-length 256 --payload-content-type=application/octet-stream key --format json"
	orderRes, _, err := f.TF.ManagedCluster.RunPodCommand(cmd, keystoneContainerName, keystonePod)
	if err != nil {
		t.Fatal(err)
	}
	var cmdOutput map[string]string
	err = json.Unmarshal([]byte(orderRes), &cmdOutput)
	if err != nil {
		t.Fatal(err)
	}
	orderHref := cmdOutput["Order href"]

	defer func() {
		f.Step(t, "Remove test KMS order")
		cmd := fmt.Sprintf("openstack secret order delete %s", orderHref)
		_, _, err := f.TF.ManagedCluster.RunPodCommand(cmd, keystoneContainerName, keystonePod)
		if err != nil {
			t.Fatal(err)
		}
	}()
	if cmdOutput["Status"] == "ERROR" {
		t.Fatalf("failed to create KMS order:\n%s", orderRes)
	}
	t.Logf("#### e2e test: KMS order href: %s", orderHref)

	cmd = fmt.Sprintf("openstack secret order get %s --format json", orderHref)
	secretRes, _, err := f.TF.ManagedCluster.RunPodCommand(cmd, keystoneContainerName, keystonePod)
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal([]byte(secretRes), &cmdOutput)
	if err != nil {
		t.Fatal(err)
	}
	secretHref := cmdOutput["Secret href"]
	tokens := strings.Split(secretHref, "/")
	if len(tokens) <= 5 {
		t.Fatalf("Incorrect KMS Secret href: '%s' (expected 'https://%s/v1/secrets/<secret_id>')", string(secret.Data["rgw_barbican_url"]), secretHref)
	}

	defer func() {
		f.Step(t, "Remove test KMS secret")
		cmd := fmt.Sprintf("openstack secret delete %s", secretHref)
		_, _, err := f.TF.ManagedCluster.RunPodCommand(cmd, keystoneContainerName, keystonePod)
		if err != nil {
			t.Fatal(err)
		}
	}()
	if cmdOutput["Status"] == "ERROR" {
		t.Fatalf("failed to create KMS secret:\n%s", secretRes)
	}
	t.Logf("#### e2e test: KMS secret href: %s", secretHref)
	kmsID := tokens[len(tokens)-1]

	f.Step(t, "Assign KMS secret for RGW service user")
	rgwServiceUser := secret.Data["username"]
	rgwUserDomain := secret.Data["user_domain_name"]
	cmd = fmt.Sprintf("openstack user show %s --domain %s --format value --column id", rgwServiceUser, rgwUserDomain)
	rgwUserID, _, err := f.TF.ManagedCluster.RunPodCommand(cmd, keystoneContainerName, keystonePod)
	if err != nil {
		t.Fatal(err)
	}
	cmd = fmt.Sprintf("openstack acl user add --user %s %s", rgwUserID, secretHref)
	_, _, err = f.TF.ManagedCluster.RunPodCommand(cmd, keystoneContainerName, keystonePod)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		f.Step(t, "Remove ACL assign for RGW user")
		cmd = fmt.Sprintf("openstack acl user remove --user %s %s", rgwUserID, secretHref)
		_, _, err := f.TF.ManagedCluster.RunPodCommand(cmd, keystoneContainerName, keystonePod)
		if err != nil {
			t.Fatal(err)
		}
	}()

	f.Step(t, "Get test deployment image")
	testImage := f.TF.E2eImage
	if testImage == "" {
		rco, err := f.TF.ManagedCluster.GetDeployment("rook-ceph-operator", f.TF.ManagedCluster.LcmConfig.RookNamespace)
		if err != nil {
			t.Fatal(errors.Wrapf(err, "failed to get deployment %s/rook-ceph-operator", f.TF.ManagedCluster.LcmConfig.RookNamespace))
		}
		testImage = rco.Spec.Template.Spec.Containers[0].Image
	}
	rgwUserName := fmt.Sprintf("rgw-test-user-%d", time.Now().Unix())

	f.Step(t, "Create custom rgw user through spec")
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	bucketQuota := 1
	objQuota := int64(1)
	rgwUser := cephlcmv1alpha1.CephRGWUser{
		Name:        rgwUserName,
		DisplayName: rgwUserName,
		Capabilities: &cephv1.ObjectUserCapSpec{
			Bucket:   "*",
			User:     "read",
			MetaData: "read",
		},
		Quotas: &cephv1.ObjectUserQuotaSpec{
			MaxBuckets: &bucketQuota,
			MaxObjects: &objQuota,
		},
	}
	if len(cd.Spec.ObjectStorage.Rgw.ObjectUsers) > 0 {
		cd.Spec.ObjectStorage.Rgw.ObjectUsers = append(cd.Spec.ObjectStorage.Rgw.ObjectUsers, rgwUser)
	} else {
		cd.Spec.ObjectStorage.Rgw.ObjectUsers = []cephlcmv1alpha1.CephRGWUser{rgwUser}
	}
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Get custom rgw user credentials")
	var userCreds *corev1.Secret
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 3*time.Minute, true, func(_ context.Context) (done bool, err error) {
		userCreds, err = f.GetSecretForRgwCreds(cd.Name, rgwUserName)
		if err != nil {
			f.TF.Log.Error().Err(err).Msg("failed to get custom rgw user credentials secret")
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatal("failed to wait for rgw user credentials secret becomes available")
	}

	endpoint := string(userCreds.Data["Endpoint"])
	customAccessKey := string(userCreds.Data["AccessKey"])
	customSecretKey := string(userCreds.Data["SecretKey"])

	f.Step(t, "Configure awscli files to verify rgw user access")
	customUserCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-rgw-user-creds",
			Namespace: f.TF.ManagedCluster.LcmConfig.RookNamespace,
		},
		Data: map[string]string{
			"config": fmt.Sprintf(`[default]
ca_bundle = /etc/rgwcerts/cacert
endpoint_url = %s`, endpoint),
			"credentials": fmt.Sprintf(`[default]
aws_access_key_id = %s
aws_secret_access_key = %s`, customAccessKey, customSecretKey),
		},
	}
	err = f.TF.ManagedCluster.CreateConfigMap(customUserCm)
	if err != nil {
		t.Fatalf("failed to create rgw user credentials configmap: %v", err)
	}

	defer func() {
		err = f.TF.ManagedCluster.DeleteConfigMap(customUserCm.Name, customUserCm.Namespace)
		if err != nil {
			t.Fatalf("failed to cleanup rgw user credentials configmap: %v", err)
		}
	}()

	f.Step(t, "Create awscli pod to verify custom rgw users access")
	awscliName := fmt.Sprintf("awscli-%d", time.Now().Unix())
	awscliLabel := "awscli-custom"
	awscli, err := f.TF.ManagedCluster.CreateAWSCliDeployment(awscliName, awscliLabel, testImage, "custom-rgw-user-creds", "rgw-ssl-certificate", "", "")
	if err != nil {
		t.Fatalf("failed to create and configure awscli for custom rgw user: %v", err)
	}

	defer func() {
		f.Step(t, "Clean up created awscli deployment")
		err = f.TF.ManagedCluster.DeleteDeployment(awscli.Name, awscli.Namespace)
		if err != nil {
			t.Fatalf("failed to delete %s/%s deployment: %v", awscli.Namespace, awscli.Name, err)
		}
	}()

	f.Step(t, "Verify custom rgw access to Ceph object storage")
	awsPodLabel := fmt.Sprintf("app=%s", awscliLabel)
	testBucketName := fmt.Sprintf("e2e-bucket-%s", rgwUserName)
	createBucketCmd := fmt.Sprintf("aws s3api create-bucket --bucket %s", testBucketName)
	_, _, err = f.TF.ManagedCluster.RunCommand(createBucketCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsPodLabel)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to create bucket for custom rgw user with error"))
	}

	defer func() {
		f.Step(t, "Clean up created s3 bucket")
		deleteBucketCmd := fmt.Sprintf("aws s3api delete-bucket --bucket %s", testBucketName)
		_, _, err := f.TF.ManagedCluster.RunCommand(deleteBucketCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsPodLabel)
		if err != nil {
			t.Fatal(errors.Wrap(err, "failed to delete bucket for custom rgw user with error"))
		}
	}()

	f.Step(t, "Create test file and upload with SSE+KMS")
	testFileName := "e2e-test-file"
	testFilePath := fmt.Sprintf("/tmp/%s", testFileName)
	createTestFileCmd := fmt.Sprintf("dd if=/dev/urandom of=%s count=2 bs=512KB", testFilePath)
	t.Log("#### e2e test: create test file")
	_, _, err = f.TF.ManagedCluster.RunCommand(createTestFileCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsPodLabel)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to create test file with error"))
	}
	sha256cmd := "sha256sum -z %s"
	testFileSHACmd := fmt.Sprintf(sha256cmd, testFilePath)
	sourceTestFileSHA, _, err := f.TF.ManagedCluster.RunCommand(testFileSHACmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsPodLabel)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to check test file SHA sum with error"))
	}
	t.Log("#### e2e test: upload test file")
	uploadCmd := fmt.Sprintf("aws s3 cp %s s3://%s/%s --sse aws:kms --sse-kms-key-id %s", testFilePath, testBucketName, testFileName, kmsID)
	_, _, err = f.TF.ManagedCluster.RunCommand(uploadCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsPodLabel)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to upload object with custom rgw user with error"))
	}

	defer func() {
		f.Step(t, "Clean up created s3 object with awscli for custom rgw user")
		deleteObjectCmd := fmt.Sprintf("aws s3api delete-object --bucket %s --key %s", testBucketName, testFileName)
		_, _, err := f.TF.ManagedCluster.RunCommand(deleteObjectCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsPodLabel)
		if err != nil {
			t.Fatal(errors.Wrap(err, "failed to delete object with custom rgw user with error"))
		}
	}()

	f.Step(t, "Verify file is uploaded and encrypted")
	t.Log("#### e2e test: download and check test file with rados cli")
	rgwDataPool := fmt.Sprintf("%s.rgw.buckets.data", f.TF.PreviousClusterState.CephDeployment.Spec.ObjectStorage.Rgw.Name)
	radosLs := fmt.Sprintf("rados ls -p %s", rgwDataPool)
	radosLsOutput, err := f.TF.ManagedCluster.RunCephToolsCommand(radosLs)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to get pool objects with error"))
	}
	t.Logf("#### e2e test: rados ls output:\n%s", radosLsOutput)
	objName := ""
	objCount := 0
	for _, obj := range strings.Split(radosLsOutput, "\n") {
		if strings.HasSuffix(obj, testFileName) {
			objName = obj
			objCount++
		}
	}
	if objName == "" {
		t.Fatalf("Failed to find uploaded object ('%s') in pool:\n%s", testFileName, radosLsOutput)
	}
	if objCount > 1 {
		t.Fatalf("detected multipart upload for test object. Please decrease file size for correct verification!")
	}
	radosFile := "/tmp/rados_downloaded"
	radosGet := fmt.Sprintf("rados get -p %s %s %s", rgwDataPool, objName, radosFile)
	_, err = f.TF.ManagedCluster.RunCephToolsCommand(radosGet)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to get rados object with error"))
	}
	testFileSHACmd = fmt.Sprintf(sha256cmd, radosFile)
	radosTestFileSHA, err := f.TF.ManagedCluster.RunCephToolsCommand(testFileSHACmd)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to check test file SHA sum with error"))
	}
	sourceSHA := strings.Split(sourceTestFileSHA, " ")[0]
	radosSHA := strings.Split(radosTestFileSHA, " ")[0]
	if sourceSHA == radosSHA {
		t.Fatalf("Encryption is not working! Test rados file SHA is same to original: source SHA256='%s', target SHA256='%s'", sourceSHA, radosSHA)
	}
	t.Log("#### e2e test: download and check test file with aws cli")
	downloadedFile := "/tmp/aws_downloaded"
	downloadCmd := fmt.Sprintf("aws s3 cp s3://%s/%s %s", testBucketName, testFileName, downloadedFile)
	_, _, err = f.TF.ManagedCluster.RunCommand(downloadCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsPodLabel)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to download object with custom rgw user with error"))
	}
	testFileSHACmd = fmt.Sprintf(sha256cmd, downloadedFile)
	targetTestFileSHA, _, err := f.TF.ManagedCluster.RunCommand(testFileSHACmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsPodLabel)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to check test file SHA sum with error"))
	}
	targetSHA := strings.Split(targetTestFileSHA, " ")[0]
	if sourceSHA != targetSHA {
		t.Fatalf("Test file SHA is different from original: source SHA256='%s', target SHA256='%s'", sourceSHA, targetSHA)
	}
	t.Logf("#### e2e test: source SHA: %s, rados SHA: %s, target SHA: %s", sourceSHA, radosSHA, targetSHA)
	t.Log("test Ceph RGW SSE+KMS completed!")
}
