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
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

func TestMultisiteRgw(t *testing.T) {
	t.Logf("#### e2e test: test Ceph RGW multisite")
	defer f.SetupTeardown(t)()

	f.Step(t, "Obtain test case configuration")
	caseConfig := f.GetConfigForTestCase(t)
	backupClusterKubeconfig, ok := caseConfig["backupClusterKubeconfig"]
	if !ok {
		t.Fatal("Could not obtain test case configuration: missed 'backupRgwClusterKubeconfig' paremeter")
	}
	backupClusterNamespace := caseConfig["backupClusterNamespace"]
	if backupClusterNamespace == "" {
		t.Log("### deployment namespace var for backup cluster 'backupClusterNamespace' is not set, using same to master cluster")
		backupClusterNamespace = f.TF.ManagedCluster.LcmNamespace
	}
	if !path.IsAbs(backupClusterKubeconfig) {
		backupClusterKubeconfig, _ = filepath.Abs(backupClusterKubeconfig)
	}
	backupClusterConfig, err := clientcmd.BuildConfigFromFlags("", backupClusterKubeconfig)
	if err != nil {
		t.Fatalf("Cannot build config from kubeconfig: %s", backupClusterKubeconfig)
	}
	backupCluster, err := f.NewManagedCluster(backupClusterNamespace, backupClusterConfig)
	if err != nil {
		t.Fatalf("Cannot initialize consumer cluster clients: %v", err)
	}

	f.Step(t, "Verify RGW multisite with master zone")
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	poolDefaultClass := f.GetDefaultPoolDeviceClass(cd)
	if poolDefaultClass == "" {
		t.Fatal("failed to find default pool")
	}

	realmName := ""
	zonegroupName := ""
	changed := true
	if cd.Spec.ObjectStorage == nil {
		t.Logf("#### e2e test: deploying new RGW Multisite master zone")
		realmName = "rgw-storerealm"
		zonegroupName = "rgw-storezonegroup"
		cd.Spec.ObjectStorage = &cephlcmv1alpha1.CephObjectStorage{
			MultiSite: &cephlcmv1alpha1.CephMultiSite{
				Realms: []cephlcmv1alpha1.CephRGWRealm{
					{
						Name: realmName,
					},
				},
				ZoneGroups: []cephlcmv1alpha1.CephRGWZoneGroup{
					{
						Name:  zonegroupName,
						Realm: realmName,
					},
				},
				Zones: []cephlcmv1alpha1.CephRGWZone{
					{
						Name: "rgw-storezone",
						MetadataPool: cephlcmv1alpha1.CephPoolSpec{
							DeviceClass: poolDefaultClass,
							Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
								Size: 3,
							},
						},
						DataPool: cephlcmv1alpha1.CephPoolSpec{
							DeviceClass: poolDefaultClass,
							ErasureCoded: &cephlcmv1alpha1.CephPoolErasureCodedSpec{
								CodingChunks: 1,
								DataChunks:   2,
							},
						},
						ZoneGroup: zonegroupName,
					},
				},
			},
			Rgw: cephlcmv1alpha1.CephRGW{
				Name: "rgw-store",
				Gateway: cephlcmv1alpha1.CephRGWGateway{
					Instances:  2,
					Port:       80,
					SecurePort: 8443,
				},
				Zone: &cephv1.ZoneSpec{Name: "rgw-storezone"},
			},
		}
	} else {
		if cd.Spec.ObjectStorage.MultiSite == nil {
			realmName = cd.Spec.ObjectStorage.Rgw.Name
			zonegroupName = cd.Spec.ObjectStorage.Rgw.Name
			t.Logf("#### e2e test: reconfigure existing RGW to RGW Multisite master mode")
			multisite := &cephlcmv1alpha1.CephMultiSite{
				Realms: []cephlcmv1alpha1.CephRGWRealm{
					{
						Name: cd.Spec.ObjectStorage.Rgw.Name,
					},
				},
				ZoneGroups: []cephlcmv1alpha1.CephRGWZoneGroup{
					{
						Name:  cd.Spec.ObjectStorage.Rgw.Name,
						Realm: cd.Spec.ObjectStorage.Rgw.Name,
					},
				},
				Zones: []cephlcmv1alpha1.CephRGWZone{
					{
						Name:         cd.Spec.ObjectStorage.Rgw.Name,
						MetadataPool: *cd.Spec.ObjectStorage.Rgw.MetadataPool.DeepCopy(),
						DataPool:     *cd.Spec.ObjectStorage.Rgw.DataPool.DeepCopy(),
						ZoneGroup:    cd.Spec.ObjectStorage.Rgw.Name,
					},
				},
			}
			newRgw := cd.Spec.ObjectStorage.Rgw.DeepCopy()
			newRgw.DataPool = nil
			newRgw.MetadataPool = nil
			newRgw.Zone = &cephv1.ZoneSpec{Name: cd.Spec.ObjectStorage.Rgw.Name}
			cd.Spec.ObjectStorage = &cephlcmv1alpha1.CephObjectStorage{
				MultiSite: multisite,
				Rgw:       *newRgw,
			}
		} else {
			t.Logf("#### e2e test: RGW Multisite master zone is already configured")
			changed = false
			realmName = cd.Spec.ObjectStorage.MultiSite.Realms[0].Name
			zonegroupName = cd.Spec.ObjectStorage.MultiSite.ZoneGroups[0].Name
		}
	}
	if changed {
		t.Logf("### e2e test: applying CephDeployment spec update on master side")
		err = f.UpdateCephDeploymentSpec(cd, true)
		if err != nil {
			t.Fatal(err)
		}
	}

	f.Step(t, "Get RGW multisite master public endpoint")
	// TODO: return endpoint from cdh, which contains HTTPS and support it later
	//rgwMasterPublicEndpoint, err := f.GetRgwPublicEndpoint(cd.Name)
	rgwMasterPublicEndpoint, err := getRgwPublicHTTPEndpoint(f.TF.ManagedCluster.Context, f.TF.ManagedCluster.KubeClient, f.TF.ManagedCluster.LcmConfig.RookNamespace, cd.Spec.ObjectStorage.Rgw.Name)
	if err != nil {
		t.Fatalf("failed to get RGW master zone public endpoint: %v", err)
	}
	t.Logf("#### e2e test: RGW multisite master public endpoint is: %s", rgwMasterPublicEndpoint)

	f.Step(t, "Get RGW multisite master side realm secrets")
	accessKey, secretKey, err := getRgwUserCreds(fmt.Sprintf("%s-keys", realmName))
	if err != nil {
		t.Fatalf("failed to get RGW Multisite master realm secrets: %v", err)
	}

	f.Step(t, "Checking CephDeployment for backup cluster")
	cdBackup, err := backupCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	if cdBackup.Spec.ObjectStorage != nil {
		t.Fatal("failed to configure backup zone, backup cluster already has some RGW setup")
	}
	err = backupCluster.WaitForCephDeploymentReady(cdBackup.Name)
	if err != nil {
		t.Fatal(err)
	}
	err = backupCluster.WaitForCephDeploymentHealthReady(cdBackup.Name)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Verify access backup -> master zone")
	stdout, err := backupCluster.RunCephToolsCommand(fmt.Sprintf("curl --silent %s", rgwMasterPublicEndpoint))
	t.Logf("cURL response for HTTP RGW endpoint is:\n%v", stdout)
	if err != nil {
		t.Fatalf("failed to verify connection between backup -> master zones: %v", err)
	}

	f.Step(t, "Configuring RGW multisite for backup zone")
	t.Logf("#### e2e test: deploying RGW Multisite backup zone")
	zoneName := "rgw-zone-backup"
	cdBackup.Spec.ObjectStorage = &cephlcmv1alpha1.CephObjectStorage{
		MultiSite: &cephlcmv1alpha1.CephMultiSite{
			Realms: []cephlcmv1alpha1.CephRGWRealm{
				{
					Name: realmName,
					Pull: &cephlcmv1alpha1.CephRGWRealmPull{
						AccessKey: accessKey,
						SecretKey: secretKey,
						Endpoint:  rgwMasterPublicEndpoint,
					},
				},
			},
			ZoneGroups: []cephlcmv1alpha1.CephRGWZoneGroup{
				{
					Name:  zonegroupName,
					Realm: realmName,
				},
			},
			Zones: []cephlcmv1alpha1.CephRGWZone{
				{
					Name: zoneName,
					MetadataPool: cephlcmv1alpha1.CephPoolSpec{
						DeviceClass: poolDefaultClass,
						Replicated: &cephlcmv1alpha1.CephPoolReplicatedSpec{
							Size: 3,
						},
					},
					DataPool: cephlcmv1alpha1.CephPoolSpec{
						DeviceClass: poolDefaultClass,
						ErasureCoded: &cephlcmv1alpha1.CephPoolErasureCodedSpec{
							CodingChunks: 1,
							DataChunks:   2,
						},
					},
					ZoneGroup: zonegroupName,
				},
			},
		},
		Rgw: cephlcmv1alpha1.CephRGW{
			Name: "rgw-store-backup",
			Gateway: cephlcmv1alpha1.CephRGWGateway{
				Instances:  1,
				Port:       80,
				SecurePort: 8443,
			},
			Zone: &cephv1.ZoneSpec{Name: zoneName},
		},
	}
	_, err = backupCluster.UpdateCephDeploymentSpec(cdBackup)
	if err != nil {
		t.Fatal(err)
	}
	err = backupCluster.WaitForCephDeploymentReady(cdBackup.Name)
	if err != nil {
		t.Fatal(err)
	}
	err = backupCluster.WaitForCephDeploymentHealthReady(cdBackup.Name)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Get RGW multisite backup zone public endpoint")
	/* TODO: return endpoint from cdh, which contains HTTPS and support it later
	cdhBackup, err := backupCluster.GetCephDeploymentHealth(cdBackup.Name)
	if err != nil {
		t.Fatalf("failed to get RGW backup zone public endpoint: %v", err)
	}
	if cdhBackup.Status.HealthReport == nil || cdhBackup.Status.HealthReport.ClusterDetails == nil || cdhBackup.Status.HealthReport.ClusterDetails.RgwInfo == nil {
		t.Fatal("backup cluster has empty RgwInfo status")
	}
	rgwBackupPublicEndpoint := cdhBackup.Status.HealthReport.ClusterDetails.RgwInfo.PublicEndpoint*/
	rgwBackupPublicEndpoint, err := getRgwPublicHTTPEndpoint(backupCluster.Context, backupCluster.KubeClient, backupCluster.LcmConfig.RookNamespace, cdBackup.Spec.ObjectStorage.Rgw.Name)
	if err != nil {
		t.Fatalf("failed to get RGW backup zone public endpoint: %v", err)
	}
	t.Logf("#### e2e test: RGW multisite backup public endpoint is: %s", rgwBackupPublicEndpoint)

	f.Step(t, "Create e2e test rgw user on master zone side")
	cd, err = f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	rgwUserName := fmt.Sprintf("rgw-e2e-test-user-%d", time.Now().Unix())
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

	f.Step(t, "Get e2e test rgw user credentials")
	userCreds, err := f.GetSecretForRgwCreds(cd.Name, rgwUserName)
	if err != nil {
		t.Fatalf("failed to get rgw user credentials secret: %v", err)
	}

	customUserCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-rgw-user-creds",
			Namespace: "rook-ceph",
		},
		Data: map[string]string{
			"config": `[default]
`,
			"credentials": fmt.Sprintf(`[default]
aws_access_key_id = %s
aws_secret_access_key = %s`, string(userCreds.Data["AccessKey"]), string(userCreds.Data["SecretKey"])),
		},
	}

	f.Step(t, "Configure awscli files to verify rgw user access on master side")
	err = f.TF.ManagedCluster.CreateConfigMap(customUserCm)
	if err != nil {
		t.Fatalf("failed to create e2e rgw user credentials configmap: %v", err)
	}
	defer func() {
		t.Logf("#### e2e test: cleaning up rgw creds config map for master side")
		err = f.TF.ManagedCluster.DeleteConfigMap(customUserCm.Name, customUserCm.Namespace)
		if err != nil {
			t.Fatalf("failed to delete rgw user credentials configmap: %v", err)
		}
	}()

	f.Step(t, "Configure awscli files to verify rgw user access on backup side")
	err = backupCluster.CreateConfigMap(customUserCm)
	if err != nil {
		t.Fatalf("failed to create rgw user credentials configmap: %v", err)
	}
	defer func() {
		t.Logf("#### e2e test: cleaning up rgw creds config map for backup side")
		err = backupCluster.DeleteConfigMap(customUserCm.Name, customUserCm.Namespace)
		if err != nil {
			t.Fatalf("failed to delete rgw user credentials configmap: %v", err)
		}
	}()

	f.Step(t, "Get test deployment image")
	awsCliImage := f.TF.E2eImage
	if awsCliImage == "" {
		rco, err := f.TF.ManagedCluster.GetDeployment("rook-ceph-operator", f.TF.ManagedCluster.LcmConfig.RookNamespace)
		if err != nil {
			t.Fatal(errors.Wrapf(err, "failed to get deployment %s/rook-ceph-operator", f.TF.ManagedCluster.LcmConfig.RookNamespace))
		}
		awsCliImage = rco.Spec.Template.Spec.Containers[0].Image
	}

	f.Step(t, "Create awscli pod to verify public endpoint accessibility on master side")
	awsCliName := fmt.Sprintf("awscli-%d", time.Now().Unix())
	awsAppCliLabel := "awscli-multisite-e2e"
	awscliMaster, err := f.TF.ManagedCluster.CreateAWSCliDeployment(awsCliName, awsAppCliLabel, awsCliImage, customUserCm.Name, "rgw-ssl-certificate", "", "")
	if err != nil {
		t.Fatalf("failed to create and configure awscli for custom rgw user: %v", err)
	}
	defer func() {
		t.Logf("#### e2e test: cleaning up aws deployment for master cluster")
		err = f.TF.ManagedCluster.DeleteDeployment(awscliMaster.Name, awscliMaster.Namespace)
		if err != nil {
			t.Fatalf("failed to delete %s/%s deployment: %v", awscliMaster.Namespace, awscliMaster.Name, err)
		}
	}()

	f.Step(t, "Create awscli pod to verify public endpoint accessibility on backup side")
	awscliBackup, err := backupCluster.CreateAWSCliDeployment(awsCliName, awsAppCliLabel, awsCliImage, customUserCm.Name, "rgw-ssl-certificate", "", "")
	if err != nil {
		t.Fatalf("failed to create and configure awscli for custom rgw user: %v", err)
	}
	defer func() {
		t.Logf("#### e2e test: cleaning up aws deployment for backup cluster")
		err = backupCluster.DeleteDeployment(awscliBackup.Name, awscliBackup.Namespace)
		if err != nil {
			t.Fatalf("failed to delete %s/%s deployment: %v", awscliBackup.Namespace, awscliBackup.Name, err)
		}
	}()

	testBucketName := "e2e-test-bucket"
	testFileName := "e2e-test-file"
	awsCliLabel := fmt.Sprintf("app=%s", awsAppCliLabel)
	testFilePath := fmt.Sprintf("/tmp/%s", testFileName)
	f.Step(t, "Verify custom rgw access to Ceph object storage on master side")
	t.Logf("#### e2e test: preparing test file")
	createTestFileCmd := fmt.Sprintf("dd if=/dev/urandom of=%s count=10 bs=1M", testFilePath)
	_, _, err = f.TF.ManagedCluster.RunCommand(createTestFileCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsCliLabel)
	if err != nil {
		t.Fatalf("failed to create test file with error: %v", err)
	}
	testFileSHACmd := fmt.Sprintf("sha256sum %s", testFilePath)
	sourceTestFileSHA, _, err := f.TF.ManagedCluster.RunCommand(testFileSHACmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsCliLabel)
	if err != nil {
		t.Fatalf("failed to check test file SHA sum with error: %v", err)
	}
	createBucketCmd := fmt.Sprintf("aws --endpoint-url %s --ca-bundle /etc/rgwcerts/cacert s3api create-bucket --bucket %s", rgwMasterPublicEndpoint, testBucketName)
	_, _, err = f.TF.ManagedCluster.RunCommand(createBucketCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsCliLabel)
	if err != nil {
		t.Fatalf("failed to create bucket with custom rgw user with error: %v", err)
	}
	defer func() {
		t.Logf("#### e2e test: cleaning up aws test bucket")
		deleteBucketCmd := fmt.Sprintf("aws --endpoint-url %s --ca-bundle /etc/rgwcerts/cacert s3api delete-bucket --bucket %s", rgwMasterPublicEndpoint, testBucketName)
		_, _, err = f.TF.ManagedCluster.RunCommand(deleteBucketCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsCliLabel)
		if err != nil {
			t.Fatalf("failed to delete bucket with custom rgw user with error: %v", err)
		}
	}()
	t.Logf("#### e2e test: pushing file to s3")
	createFileCmd := fmt.Sprintf("aws --endpoint-url %s --ca-bundle /etc/rgwcerts/cacert s3api put-object --bucket %s --key %s --body %s", rgwMasterPublicEndpoint, testBucketName, testFileName, testFilePath)
	_, _, err = f.TF.ManagedCluster.RunCommand(createFileCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsCliLabel)
	if err != nil {
		t.Fatalf("failed to create object with custom rgw user with error: %v", err)
	}
	defer func() {
		t.Logf("#### e2e test: cleaning up aws test object")
		deleteObjectCmd := fmt.Sprintf("aws --endpoint-url %s --ca-bundle /etc/rgwcerts/cacert s3api delete-object --bucket %s --key %s", rgwMasterPublicEndpoint, testBucketName, testFileName)
		_, _, err := f.TF.ManagedCluster.RunCommand(deleteObjectCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsCliLabel)
		if err != nil {
			t.Fatalf("failed to delete object with custom rgw user with error: %v", err)
		}
	}()

	f.Step(t, "Sleep 1 minute, waiting for MultiSite synchronisation between zones...")
	time.Sleep(1 * time.Minute)

	f.Step(t, "Verify custom rgw access to s3 on backup side")
	getObjectCmd := fmt.Sprintf("aws --endpoint-url %s --ca-bundle /etc/rgwcerts/cacert s3api get-object --bucket %s --key %s %s", rgwBackupPublicEndpoint, testBucketName, testFileName, testFilePath)
	_, _, err = backupCluster.RunCommand(getObjectCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsCliLabel)
	if err != nil {
		t.Fatalf("failed to get object with custom rgw user with error: %v", err)
	}
	targetTestFileSHA, _, err := backupCluster.RunCommand(testFileSHACmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsCliLabel)
	if err != nil {
		t.Fatalf("failed to check test file SHA sum with error: %v", err)
	}
	if sourceTestFileSHA != targetTestFileSHA {
		t.Fatalf("test file SHA is different from original: source SHA256='%s', target SHA256='%s'", sourceTestFileSHA, targetTestFileSHA)
	}
	f.Step(t, "Multisite test completed!")
}

func getRgwPublicHTTPEndpoint(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string, rgwName string) (string, error) {
	name := fmt.Sprintf("rook-ceph-rgw-%s-external", rgwName)
	externalSvc, err := kubeClient.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "failed to get external RGW service %s/%s", namespace, name)
	}
	httpPort := 0
	for _, port := range externalSvc.Spec.Ports {
		if port.Name == "http" {
			httpPort = int(port.Port)
			break
		}
	}
	ip := ""
	if len(externalSvc.Status.LoadBalancer.Ingress) > 0 {
		ip = externalSvc.Status.LoadBalancer.Ingress[0].IP
	}
	if ip == "" || httpPort == 0 {
		return "", errors.Errorf("failed to find http endpoint for RGW %s/%s (no http port or ip found)", namespace, name)
	}
	return fmt.Sprintf("http://%s:%d", ip, httpPort), nil
}

func getRgwUserCreds(secretName string) (string, string, error) {
	secret, err := f.TF.ManagedCluster.KubeClient.CoreV1().Secrets(f.TF.ManagedCluster.LcmConfig.RookNamespace).Get(f.TF.ManagedCluster.Context, secretName, metav1.GetOptions{})
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to get multsite realm secret keys %s/%s", f.TF.ManagedCluster.LcmConfig.RookNamespace, secretName)
	}
	if _, ok := secret.Data["access-key"]; !ok {
		return "", "", errors.Errorf("access key is not specified in secret %s/%s", f.TF.ManagedCluster.LcmConfig.RookNamespace, secretName)
	}
	if _, ok := secret.Data["secret-key"]; !ok {
		return "", "", errors.Errorf("secret key is not specified in secret %s/%s", f.TF.ManagedCluster.LcmConfig.RookNamespace, secretName)
	}
	return string(secret.Data["access-key"]), string(secret.Data["secret-key"]), nil
}
