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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

func verifyRgwConnection(t *testing.T, rgwName string, httpPort int32, httpsPort int32) {
	t.Logf("Verify RGW is accessible from external loadbalancer IP")
	externalSvc, err := f.TF.ManagedCluster.GetService(fmt.Sprintf("rook-ceph-rgw-%s-external", rgwName), f.TF.ManagedCluster.LcmConfig.RookNamespace)
	rgwURL := ""
	if err != nil {
		ingress, _ := f.TF.ManagedCluster.GetIngress(fmt.Sprintf("rook-ceph-rgw-%s-ingress", rgwName), f.TF.ManagedCluster.LcmConfig.RookNamespace)
		if ingress != nil {
			rgwURL = ingress.Spec.Rules[0].Host
		}
	} else {
		rgwURL = externalSvc.Status.LoadBalancer.Ingress[0].IP
	}

	t.Logf("Send GET request to HTTP pkg RGW endpoint")
	stdout, _, err := f.TF.ManagedCluster.RunCommand(fmt.Sprintf("curl --silent http://rook-ceph-rgw-%s.%s.svc:%d/", rgwName, f.TF.ManagedCluster.LcmConfig.RookNamespace, httpPort), f.TF.ManagedCluster.LcmConfig.RookNamespace, "app=rook-ceph-rgw")
	t.Logf("cURL response for HTTP RGW endpoint is:\n%v", stdout)
	assert.Nil(t, err)

	t.Logf("Send GET request to HTTPs pkg RGW endpoint")
	stdout, _, err = f.TF.ManagedCluster.RunCommand(fmt.Sprintf("curl --silent https://rook-ceph-rgw-%s.%s.svc:%d/", rgwName, f.TF.ManagedCluster.LcmConfig.RookNamespace, httpsPort), f.TF.ManagedCluster.LcmConfig.RookNamespace, "app=rook-ceph-rgw")
	t.Logf("cURL response for HTTPs RGW endpoint is:\n%v", stdout)
	assert.Nil(t, err)

	if rgwURL != "" {
		t.Logf("Send GET request to HTTPs public RGW endpoint")
		stdout, _, err = f.TF.ManagedCluster.RunCommand(fmt.Sprintf("curl -k --silent https://%s/", rgwURL), f.TF.ManagedCluster.LcmConfig.RookNamespace, "app=rook-ceph-rgw")
		t.Logf("cURL response for HTTP public RGW endpoint is:\n%v", stdout)
		assert.Nil(t, err)
	}
}

func TestObjectStorageRgwAccessibility(t *testing.T) {
	t.Logf("#### e2e test: access existent Ceph Object Storage RGW")
	defer f.SetupTeardown(t)()
	t.Logf("Verify there is object storage in ceph cluster already created")
	rgws, err := f.TF.ManagedCluster.ListCephObjectStore()
	assert.Nil(t, err)
	if len(rgws) == 0 {
		t.Skip("There is no CephObjectStore created in current Ceph Cluster, skip test")
	}

	for _, rgw := range rgws {
		verifyRgwConnection(t, rgw.Name, rgw.Spec.Gateway.Port, rgw.Spec.Gateway.SecurePort)
	}

	t.Logf("#### Test successfully passed")
}

func TestRgwUserCreateAccess(t *testing.T) {
	t.Log("e2e test: access existent Ceph Object Storage RGW")
	defer f.SetupTeardown(t)()

	f.Step(t, "Verify there is object storage in ceph cluster already created")
	rgws, err := f.TF.ManagedCluster.ListCephObjectStore()
	assert.Nil(t, err)
	if len(rgws) == 0 {
		t.Skip("There is no CephObjectStore created in current Ceph Cluster, skip test")
	} else if len(rgws) > 1 {
		t.Fatal("There are several CephObjectStore created in current Ceph Cluster, which is invalid")
	}

	runRgwAccessTest(t, "", "", "", "", true)

	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	if lcmcommon.IsOpenStackPoolsPresent(cd.Spec.Pools) {
		f.Step(t, "check Openstack Ceilometer user access")
		runRgwAccessTest(t, "", "", "", "rgw-ceilometer", false)
	} else {
		t.Log("There is no Rockoon installed, skipping step")
	}

	t.Log("Test successfully passed")
}

func TestRgwAccessPublicDomainRockoon(t *testing.T) {
	t.Log("e2e test: verify ingress rgw endpoint with default Rockoon public domain")
	defer f.SetupTeardown(t)()

	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	if !lcmcommon.IsOpenStackPoolsPresent(cd.Spec.Pools) {
		t.Skip("There are no openstack pools therefore could not proceed the test")
	}

	f.Step(t, "Verify there is object storage in ceph cluster already created")
	rgws, err := f.TF.ManagedCluster.ListCephObjectStore()
	assert.Nil(t, err)
	if len(rgws) == 0 {
		t.Skip("There is no CephObjectStore created in current Ceph Cluster, skip test")
	} else if len(rgws) > 1 {
		t.Fatal("There are several CephObjectStore created in current Ceph Cluster, which is invalid")
	}

	f.Step(t, "Obtain RGW public endpoint with Rockoon default domain")
	endpoint, err := f.GetRgwPublicEndpoint(cd.Name)
	if err != nil {
		t.Fatal(err)
	}

	runRgwAccessTest(t, endpoint, "", "", "", true)

	t.Log("Test successfully passed")
}

func TestRgwAccessPublicDomainCustomMKE(t *testing.T) {
	t.Log("e2e test: verify ingress rgw endpoint with custom public domain")
	defer f.SetupTeardown(t)()

	f.Step(t, "Verify there is object storage in ceph cluster already created")
	rgws, err := f.TF.ManagedCluster.ListCephObjectStore()
	assert.Nil(t, err)
	if len(rgws) == 0 {
		t.Skip("There is no CephObjectStore created in current Ceph Cluster, skip test")
	} else if len(rgws) > 1 {
		t.Fatal("There are several CephObjectStore created in current Ceph Cluster, which is invalid")
	}

	f.Step(t, "Create public domain and generate certificates for it")
	rgwName := rgws[0].Name
	publicDomain := "mirantis.example.com"
	fqdn := fmt.Sprintf("%v.%v", rgwName, publicDomain)
	tlsKey, tlsCert, caCert, err := lcmcommon.GenerateSelfSignedCert("kubernetes-radosgw", fmt.Sprintf("*.%s", publicDomain), []string{fmt.Sprintf("*.%s", publicDomain)})
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Define custom ingress with public domain in spec")
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	cd.Spec.IngressConfig = &cephlcmv1alpha1.CephDeploymentIngressConfig{
		Annotations: map[string]string{
			"nginx.ingress.kubernetes.io/proxy-body-size": "0",
			"nginx.ingress.kubernetes.io/rewrite-target":  "/",
			"nginx.ingress.kubernetes.io/upstream-vhost":  fqdn,
		},
		ControllerClassName: "nginx-default",
		TLSConfig: &cephlcmv1alpha1.CephDeploymentIngressTLSConfig{
			TLSCerts: &cephlcmv1alpha1.CephDeploymentCert{
				Cacert:  caCert,
				TLSCert: tlsCert,
				TLSKey:  tlsKey,
			},
			Domain: publicDomain,
		},
	}
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Check new public rgw endpoint")
	endpoint, err := f.GetRgwPublicEndpoint(cd.Name)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(endpoint, publicDomain) {
		t.Fatal("public endpoint has no expected domain")
	}

	f.Step(t, "Obtain ingress controller IP address")
	ingressSvc, err := f.TF.ManagedCluster.GetService("ingress-nginx-controller", "ingress-nginx")
	if err != nil {
		t.Fatalf("failed to get ingress service: %v", err)
	}
	ingressIP := ingressSvc.Status.LoadBalancer.Ingress[0].IP

	runRgwAccessTest(t, endpoint, ingressIP, fqdn, "", true)

	t.Log("Test successfully passed")
}

func TestRgwAccessPublicTlsByRefAndCustomHostnameRockoon(t *testing.T) {
	t.Log("e2e test: verify ingress rgw endpoint with custom hostname and Rockoon certs")
	defer f.SetupTeardown(t)()

	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	if !lcmcommon.IsOpenStackPoolsPresent(cd.Spec.Pools) {
		t.Skip("There are no openstack pools therefore could not proceed the test")
	}

	f.Step(t, "Verify there is object storage in ceph cluster already created")
	rgws, err := f.TF.ManagedCluster.ListCephObjectStore()
	assert.Nil(t, err)
	if len(rgws) == 0 {
		t.Skip("There is no CephObjectStore created in current Ceph Cluster, skip test")
	} else if len(rgws) > 1 {
		t.Fatal("There are several CephObjectStore created in current Ceph Cluster, which is invalid")
	}

	f.Step(t, "Prepare secret based on Rockoon certs")
	secret, err := f.TF.ManagedCluster.GetSecret("openstack-rgw-creds", f.TF.ManagedCluster.LcmConfig.DeployParams.OpenstackCephSharedNamespace)
	if err != nil {
		t.Fatal(err)
	}
	tlsSecretResource := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-ingress-tls-cert",
			Namespace: f.TF.ManagedCluster.LcmConfig.RookNamespace,
		},
		Data: map[string][]byte{
			"ca.crt":  secret.Data["ca_cert"],
			"tls.crt": secret.Data["tls_crt"],
			"tls.key": secret.Data["tls_key"],
		},
	}
	err = f.TF.ManagedCluster.CreateSecret(tlsSecretResource)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		f.Step(t, "Clean up created custom secret")
		err := f.TF.ManagedCluster.DeleteSecret(tlsSecretResource.Name, tlsSecretResource.Namespace)
		if err != nil && !apierrors.IsNotFound(err) {
			t.Fatalf("failed to delete '%s/%s' secret: %v", tlsSecretResource.Namespace, tlsSecretResource.Name, err)
		}
	}()

	f.Step(t, "Define ingress with hostname and tls by ref in spec")
	cd.Spec.IngressConfig = &cephlcmv1alpha1.CephDeploymentIngressConfig{
		TLSConfig: &cephlcmv1alpha1.CephDeploymentIngressTLSConfig{
			TLSSecretRefName: tlsSecretResource.Name,
			Domain:           "it.just.works",
			Hostname:         "my-rgw-custom",
		},
	}
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Check new public rgw endpoint")
	endpoint, err := f.GetRgwPublicEndpoint(cd.Name)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(endpoint, cd.Spec.IngressConfig.TLSConfig.Domain) {
		t.Fatal("public endpoint has no expected domain")
	}

	f.Step(t, "Obtain ingress controller IP address")
	ingressSvc, err := f.TF.ManagedCluster.GetService("ingress", "openstack")
	if err != nil {
		t.Fatalf("failed to get ingress service: %v", err)
	}
	ingressIP := ingressSvc.Status.LoadBalancer.Ingress[0].IP

	runRgwAccessTest(t, endpoint, ingressIP, fmt.Sprintf("%v.%v", cd.Spec.IngressConfig.TLSConfig.Hostname, cd.Spec.IngressConfig.TLSConfig.Domain), "", true)

	t.Log("Test successfully passed")
}

func runRgwAccessTest(t *testing.T, endpoint, ingressIP, domain, rgwUserName string, checkOverQuota bool) {
	awscliName := fmt.Sprintf("awscli-%d", time.Now().Unix())
	customUserCmName := fmt.Sprintf("custom-rgw-user-creds-%d", time.Now().Unix())
	testNamespace := f.TF.ManagedCluster.LcmConfig.RookNamespace

	f.Step(t, "Get test deployment image from Rook Ceph Operator")
	testImage := f.TF.E2eImage
	if testImage == "" {
		rco, err := f.TF.ManagedCluster.GetDeployment("rook-ceph-operator", f.TF.ManagedCluster.LcmConfig.RookNamespace)
		if err != nil {
			t.Fatal(errors.Wrapf(err, "failed to get deployment %s/rook-ceph-operator", f.TF.ManagedCluster.LcmConfig.RookNamespace))
		}
		testImage = rco.Spec.Template.Spec.Containers[0].Image
	}

	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	if rgwUserName == "" {
		f.Step(t, "Create custom rgw user through spec")
		rgwUserName = fmt.Sprintf("rgw-test-user-%d", time.Now().Unix())
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
	}

	f.Step(t, "Get custom rgw user credentials")
	var userCreds *corev1.Secret
	err = wait.PollUntilContextTimeout(f.TF.ManagedCluster.Context, 5*time.Second, 10*time.Minute, true, func(_ context.Context) (bool, error) {
		creds, secretErr := f.GetSecretForRgwCreds(cd.Name, rgwUserName)
		if secretErr != nil {
			t.Logf("failed to get custom rgw user credentials secret: %v", secretErr)
			return false, nil
		}
		userCreds = creds
		return true, nil
	})
	if err != nil {
		t.Fatalf("failed to wait for custom rgw user credentials secret: %v", err)
	}

	customAccessKey := string(userCreds.Data["AccessKey"])
	customSecretKey := string(userCreds.Data["SecretKey"])
	if endpoint == "" {
		endpoint = string(userCreds.Data["Endpoint"])
	}

	f.Step(t, "Configure awscli files to verify rgw user access")
	customUserCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      customUserCmName,
			Namespace: f.TF.ManagedCluster.LcmConfig.RookNamespace,
		},
		Data: map[string]string{
			"config": `[default]
`,
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
		f.Step(t, "Clean up created aws config configmap")
		err := f.TF.ManagedCluster.DeleteConfigMap(customUserCmName, f.TF.ManagedCluster.LcmConfig.RookNamespace)
		if err != nil && !apierrors.IsNotFound(err) {
			t.Fatalf("failed to delete 'rook-ceph/%s' configmap: %v", customUserCmName, err)
		}
	}()

	f.Step(t, "Create awscli pod to verify public endpoint accessibility")
	_, err = f.TF.ManagedCluster.CreateAWSCliDeployment(awscliName, "", testImage, customUserCmName, "rgw-ssl-certificate", ingressIP, domain)
	if err != nil {
		t.Fatalf("failed to create and configure awscli for custom rgw user: %v", err)
	}

	defer func() {
		f.Step(t, "Clean up created awscli")
		err := f.TF.ManagedCluster.DeleteDeployment(awscliName, testNamespace)
		if err != nil && !apierrors.IsNotFound(err) {
			t.Fatalf("failed to delete '%s/%s' deployment: %v", testNamespace, awscliName, err)
		}
	}()

	f.Step(t, "Verify rgw access with public endpoint")
	awsCliLabel := "app=awscli"
	s3Api := fmt.Sprintf("aws --endpoint-url %s --ca-bundle /etc/rgwcerts/cabundle s3api", endpoint)
	createBucketCmd := fmt.Sprintf("%s create-bucket --bucket bucket-%s", s3Api, rgwUserName)
	_, _, err = f.TF.ManagedCluster.RunCommand(createBucketCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsCliLabel)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to create bucket with public endpoint with error"))
	}

	defer func() {
		f.Step(t, "Clean up created test bucket")
		deleteBucketCmd := fmt.Sprintf("%s delete-bucket --bucket bucket-%s", s3Api, rgwUserName)
		_, _, err = f.TF.ManagedCluster.RunCommand(deleteBucketCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsCliLabel)
		if err != nil {
			t.Fatal(errors.Wrap(err, "failed to delete bucket with public endpoint with error"))
		}
	}()

	if checkOverQuota {
		quotaBucketCmd := fmt.Sprintf("%s --debug create-bucket --bucket bucket-%s-quota", s3Api, rgwUserName)
		_, stderr, err := f.TF.ManagedCluster.RunCommand(quotaBucketCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsCliLabel)
		if err == nil {
			deleteBucketCmd2 := fmt.Sprintf("%s delete-bucket --bucket bucket-%s-quota", s3Api, rgwUserName)
			_, _, err := f.TF.ManagedCluster.RunCommand(deleteBucketCmd2, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsCliLabel)
			if err != nil {
				t.Log(errors.Wrap(err, "failed to delete bucket with public endpoint with error"))
			}
			t.Fatal("bucket quota with public endpoint not working - over-quota bucket created successfully")
		}
		if !strings.Contains(stderr, "TooManyBuckets") && !strings.Contains(err.Error(), "TooManyBuckets") {
			t.Log(stderr)
			t.Fatalf("expected bucket quota error, got: %v", err)
		}
	}

	listBucketCmd := fmt.Sprintf("%s list-buckets", s3Api)
	stdout, _, err := f.TF.ManagedCluster.RunCommand(listBucketCmd, f.TF.ManagedCluster.LcmConfig.RookNamespace, awsCliLabel)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to list buckets with public endpoint with error"))
	}
	if !strings.Contains(stdout, fmt.Sprintf("bucket-%s", rgwUserName)) {
		t.Fatalf("test bucket bucket-%s not created with public endpoint", rgwUserName)
	}
}
