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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	f "github.com/Mirantis/pelagia/test/e2e/framework"
)

func verifyRgwConnection(t *testing.T, rgwName string, httpPort int32, httpsPort int32) {
	t.Logf("Verify RGW is accessible from internal endpoint")
	t.Logf("Send GET request to HTTP pkg RGW endpoint")
	stdout, _, err := f.TF.ManagedCluster.RunCommand(fmt.Sprintf("curl --silent http://rook-ceph-rgw-%s.%s.svc:%d/", rgwName, f.TF.ManagedCluster.LcmConfig.RookNamespace, httpPort), f.TF.ManagedCluster.LcmConfig.RookNamespace, "app=rook-ceph-rgw")
	t.Logf("CURL response for HTTP RGW endpoint is:\n%v", stdout)
	assert.Nil(t, err)

	t.Logf("Send GET request to HTTPs pkg RGW endpoint")
	stdout, _, err = f.TF.ManagedCluster.RunCommand(fmt.Sprintf("curl --silent https://rook-ceph-rgw-%s.%s.svc:%d/", rgwName, f.TF.ManagedCluster.LcmConfig.RookNamespace, httpsPort), f.TF.ManagedCluster.LcmConfig.RookNamespace, "app=rook-ceph-rgw")
	t.Logf("CURL response for HTTPs RGW endpoint is:\n%v", stdout)
	assert.Nil(t, err)

	t.Logf("Verify RGW is accessible from external loadbalancer IP")
	externalSvc, err := f.TF.ManagedCluster.GetService(fmt.Sprintf("rook-ceph-rgw-%s-external", rgwName), f.TF.ManagedCluster.LcmConfig.RookNamespace)
	rgwURL := ""
	if err != nil {
		if !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
		t.Log("Default external loadbalancer is not present")
	} else {
		rgwURL = externalSvc.Status.LoadBalancer.Ingress[0].IP
		t.Logf("Send GET request to HTTPs public RGW endpoint")
		stdout, _, err = f.TF.ManagedCluster.RunCommand(fmt.Sprintf("curl -k --silent https://%s/", rgwURL), f.TF.ManagedCluster.LcmConfig.RookNamespace, "app=rook-ceph-rgw")
		t.Logf("CURL response for HTTP public RGW endpoint is:\n%v", stdout)
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

	runRgwAccessTest(t, "", "", "", "", "", true)

	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	if cd.Spec.BlockStorage != nil && lcmcommon.IsOpenStackPoolsPresent(cd.Spec.BlockStorage.Pools) {
		f.Step(t, "check Openstack Ceilometer user access")
		runRgwAccessTest(t, "", "", "", "", "rgw-ceilometer", false)
	} else {
		t.Log("There is no Rockoon installed, skipping step")
	}

	t.Log("Test successfully passed")
}

func TestRgwAccessPublicDomainRockoon(t *testing.T) {
	t.Log("e2e test: verify rgw endpoint with default Rockoon public domain")
	defer f.SetupTeardown(t)()

	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	if cd.Spec.BlockStorage == nil || !lcmcommon.IsOpenStackPoolsPresent(cd.Spec.BlockStorage.Pools) {
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
	endpoints, err := f.GetRgwPublicEndpoints(cd.Name, rgws[0].Name)
	if err != nil {
		t.Fatal(err)
	}

	runRgwAccessTest(t, endpoints[0], "", "", "", "", true)

	t.Log("Test successfully passed")
}

func TestRgwIngressAccessPublicDomainCustom(t *testing.T) {
	t.Log("e2e test: verify ingress rgw endpoint with custom public domain")
	defer f.SetupTeardown(t)()
	if !f.TF.ManagedCluster.LcmConfig.CommonParams.KeepIngress {
		t.Skip("There are no Ingress support enabled")
	}

	f.Step(t, "Verify there is object storage in ceph cluster already created")
	rgws, err := f.TF.ManagedCluster.ListCephObjectStore()
	assert.Nil(t, err)
	if len(rgws) == 0 {
		t.Skip("There is no CephObjectStore created in current Ceph Cluster, skip test")
	} else if len(rgws) > 1 {
		t.Fatal("There are several CephObjectStore created in current Ceph Cluster, which is invalid")
	}

	f.Step(t, "Verify there is required ingress class present")
	testConfig := f.GetConfigForTestCase(t)
	ingressClassName, namePresent := testConfig["ingressClass"]
	if namePresent {
		ingressClass, err := f.TF.ManagedCluster.GetIngressClass(ingressClassName)
		if err != nil {
			t.Fatal(err)
		}
		if ingressClass == nil {
			t.Fatalf("IngressClass '%s' is not found for test", ingressClassName)
		}
	} else {
		t.Skip("IngressClass is not specified for test ('ingressClass' test option), skip test")
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
	rgwStoreName := testConfig["rgwServedByIngress"]
	if rgwStoreName == "" {
		rgwStoreName = cd.Spec.ObjectStorage.Rgws[0].Name
	}

	cd.Spec.IngressConfig = &cephlcmv1alpha1.CephDeploymentIngressConfig{
		Annotations: map[string]string{
			"nginx.ingress.kubernetes.io/proxy-body-size": "0",
			"nginx.ingress.kubernetes.io/rewrite-target":  "/",
			"nginx.ingress.kubernetes.io/upstream-vhost":  fqdn,
		},
		ControllerClassName: ingressClassName,
		TLSConfig: &cephlcmv1alpha1.CephDeploymentIngressTLSConfig{
			TLSCerts: &cephlcmv1alpha1.CephDeploymentCert{
				Cacert:  caCert,
				TLSCert: tlsCert,
				TLSKey:  tlsKey,
			},
			Domain: publicDomain,
		},
	}
	for idx, rgw := range cd.Spec.ObjectStorage.Rgws {
		if rgw.Name == rgwStoreName {
			cd.Spec.ObjectStorage.Rgws[idx].ServedByIngress = true
			break
		}
	}
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Check new public rgw endpoint")
	endpoints, err := f.GetRgwPublicEndpoints(cd.Name, rgwStoreName)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(endpoints[0], publicDomain) {
		t.Fatal("public endpoint has no expected domain")
	}

	f.Step(t, "Obtain ingress controller IP address")
	ingressSvc, err := f.TF.ManagedCluster.GetService("ingress-nginx-controller", "ingress-nginx")
	if err != nil {
		t.Fatalf("failed to get ingress service: %v", err)
	}
	ingressIP := ingressSvc.Status.LoadBalancer.Ingress[0].IP

	runRgwAccessTest(t, endpoints[0], ingressIP, fqdn, rgwStoreName, "", true)

	t.Log("Test successfully passed")
}

func TestRgwIngressAccessPublicTlsByRefAndCustomHostnameRockoon(t *testing.T) {
	t.Log("e2e test: verify ingress rgw endpoint with custom hostname and Rockoon certs")
	defer f.SetupTeardown(t)()
	if !f.TF.ManagedCluster.LcmConfig.CommonParams.KeepIngress {
		t.Skip("There are no Ingress support enabled")
	}

	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	if cd.Spec.BlockStorage == nil || !lcmcommon.IsOpenStackPoolsPresent(cd.Spec.BlockStorage.Pools) {
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
	endpoints, err := f.GetRgwPublicEndpoints(cd.Name, rgws[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(endpoints[0], cd.Spec.IngressConfig.TLSConfig.Domain) {
		t.Fatal("public endpoint has no expected domain")
	}

	f.Step(t, "Obtain ingress controller IP address")
	ingressSvc, err := f.TF.ManagedCluster.GetService("ingress", "openstack")
	if err != nil {
		t.Fatalf("failed to get ingress service: %v", err)
	}
	ingressIP := ingressSvc.Status.LoadBalancer.Ingress[0].IP

	runRgwAccessTest(t, endpoints[0], ingressIP, fmt.Sprintf("%v.%v", cd.Spec.IngressConfig.TLSConfig.Hostname, cd.Spec.IngressConfig.TLSConfig.Domain), "", "", true)

	t.Log("Test successfully passed")
}

func TestRgwGatewayAPIAccessPublicCustomHostname(t *testing.T) {
	t.Log("e2e test: verify gateway httproute rgw endpoint with custom hostname")
	defer f.SetupTeardown(t)()
	if !f.TF.ManagedCluster.LcmConfig.CommonParams.GatewayAPIEnabled {
		t.Skip("There are no GatewayAPI support enabled")
	}
	testConfig := f.GetConfigForTestCase(t)
	publicDomain := ""
	if domain, ok := testConfig["publicDomain"]; ok {
		publicDomain = domain
	} else {
		t.Fatal("Public domain for Gateway HTTPRoute is not specified")
	}
	var sslCertNamespace, sslCertName string
	if certNamespace, ok := testConfig["certNamespace"]; ok {
		sslCertNamespace = certNamespace
	}
	if certName, ok := testConfig["certName"]; ok {
		sslCertName = certName
	}

	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	openstackPoolsPresent := cd.Spec.BlockStorage == nil && !lcmcommon.IsOpenStackPoolsPresent(cd.Spec.BlockStorage.Pools)

	if len(cd.Spec.ObjectStorage.GatewayHTTPRoutes) > 0 {
		t.Skip("Some custom Gateway HTTPRoutes already present in spec, skipping")
	}

	customRgwHostname := fmt.Sprintf("rgw-custom.%s", publicDomain)
	rgwName := ""
	for idx, rgw := range cd.Spec.ObjectStorage.Rgws {
		if openstackPoolsPresent {
			if !rgw.UsedForOpenstack {
				continue
			}
		} else if rgw.AuxiliaryService {
			continue
		}
		rgwName = rgw.Name
		rgwCasted, _ := rgw.GetSpec()
		if rgwCasted.Hosting == nil {
			rgwCasted.Hosting = &cephv1.ObjectStoreHostingSpec{}
		}
		rgwCasted.Hosting.DNSNames = append(rgwCasted.Hosting.DNSNames, customRgwHostname)

		if sslCertName != "" && sslCertNamespace != "" {
			f.Step(t, "Found custom ssl secret '%s/%s', creating cabundle for RGW", sslCertNamespace, sslCertName)
			secret, err := f.TF.ManagedCluster.GetSecret(sslCertName, sslCertNamespace)
			if err != nil {
				t.Fatalf("failed to get secret '%s/%s' with ssl certificate: %v", sslCertNamespace, sslCertName, err)
			}
			if v, ok := secret.Data["ca.crt"]; ok {
				caBundleSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "custom-rgw-ca",
						Namespace: f.TF.ManagedCluster.LcmConfig.RookNamespace,
					},
					Data: map[string][]byte{
						"cabundle": v,
					},
				}
				err = f.TF.ManagedCluster.CreateSecret(caBundleSecret)
				if err != nil {
					t.Fatal(err)
				}

				defer func() {
					f.Step(t, "Cleanup custom cabundle secret")
					err := f.TF.ManagedCluster.DeleteSecret(caBundleSecret.Name, caBundleSecret.Namespace)
					if err != nil {
						t.Fatalf("failed to cleanup cabundle secret: %s", err)
					}
				}()

				rgwCasted.Gateway.CaBundleRef = caBundleSecret.Name
			} else {
				t.Fatalf("failed to find 'ca.crt' key in secret '%s/%s'", sslCertNamespace, sslCertName)
			}
		}

		rgwSpec, _ := cephlcmv1alpha1.DecodeStructToRaw(rgwCasted)
		cd.Spec.ObjectStorage.Rgws[idx].Spec.Raw = rgwSpec
		break
	}
	if rgwName == "" {
		t.Fatal("failed to find RGW ObjectStore for test")
	}

	f.Step(t, "Define HTTPRoute with hostname in spec")
	httpRoute := gatewayapi.HTTPRouteSpec{
		Hostnames: []gatewayapi.Hostname{gatewayapi.Hostname(customRgwHostname)},
	}
	gatewayHTTPRouteRaw, _ := cephlcmv1alpha1.DecodeStructToRaw(httpRoute)
	cd.Spec.ObjectStorage.GatewayHTTPRoutes = []cephlcmv1alpha1.CephDeploymentHTTPRoute{
		{
			Name:            "custom-httproute",
			ObjectStoreName: rgwName,
			Spec: runtime.RawExtension{
				Raw: gatewayHTTPRouteRaw,
			},
		},
	}
	err = f.UpdateCephDeploymentSpec(cd, true)
	if err != nil {
		t.Fatal(err)
	}

	f.Step(t, "Check new public rgw endpoint")
	endpoints, err := f.GetRgwPublicEndpoints(cd.Name, rgwName)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(endpoints[0], customRgwHostname) {
		t.Fatal("public endpoint has no expected domain")
	}

	f.Step(t, "Obtain Gateway controller IP address")
	gtwSvc, err := f.TF.ManagedCluster.GetService(f.TF.ManagedCluster.LcmConfig.CommonParams.BaseGatewayName, f.TF.ManagedCluster.LcmConfig.CommonParams.BaseGatewayNamespace)
	if err != nil {
		t.Fatalf("failed to get gateway service: %v", err)
	}
	gtwIP := gtwSvc.Status.LoadBalancer.Ingress[0].IP

	runRgwAccessTest(t, endpoints[0], gtwIP, customRgwHostname, "", "", true)

	t.Log("Test successfully passed")
}

func runRgwAccessTest(t *testing.T, endpoint, proxyIP, domain, rgwStoreName, rgwUserName string, checkOverQuota bool) {
	awscliName := fmt.Sprintf("awscli-%d", time.Now().Unix())
	customUserCmName := fmt.Sprintf("custom-rgw-user-creds-%d", time.Now().Unix())
	testNamespace := f.TF.ManagedCluster.LcmConfig.RookNamespace

	f.Step(t, "Get test deployment image from Rook Ceph Operator")
	cd, err := f.TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		t.Fatal(err)
	}
	if rgwStoreName == "" {
		rgwStoreName = cd.Spec.ObjectStorage.Rgws[0].Name
	}
	if rgwUserName == "" {
		f.Step(t, "Create custom rgw user through spec")
		rgwUserName = fmt.Sprintf("rgw-test-user-%d", time.Now().Unix())
		bucketQuota := 1
		objQuota := int64(1)
		rgwUserRaw, _ := cephlcmv1alpha1.DecodeStructToRaw(
			cephv1.ObjectStoreUserSpec{
				Store:       rgwStoreName,
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
			},
		)
		rgwUser := cephlcmv1alpha1.CephObjectStoreUser{
			Name: rgwUserName,
			Spec: runtime.RawExtension{
				Raw: rgwUserRaw,
			},
		}

		if len(cd.Spec.ObjectStorage.Users) > 0 {
			cd.Spec.ObjectStorage.Users = append(cd.Spec.ObjectStorage.Users, rgwUser)
		} else {
			cd.Spec.ObjectStorage.Users = []cephlcmv1alpha1.CephObjectStoreUser{rgwUser}
		}
		err = f.UpdateCephDeploymentSpec(cd, true)
		if err != nil {
			t.Fatal(err)
		}
	}
	certSecretName := fmt.Sprintf("%s-ssl-cert", rgwStoreName)
	for _, rgw := range cd.Spec.ObjectStorage.Rgws {
		if rgw.Name == rgwStoreName {
			rgwSpec, _ := rgw.GetSpec()
			if rgwSpec.Gateway.CaBundleRef != "" {
				certSecretName = rgwSpec.Gateway.CaBundleRef
			} else if rgwSpec.Gateway.SSLCertificateRef != "" {
				// backward compatibility for upgraded envs from Pelagia v1
				certSecretName = rgwSpec.Gateway.SSLCertificateRef
			}
			break
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
	_, err = f.TF.ManagedCluster.CreateAWSCliDeployment(awscliName, "", f.TF.E2eImage, customUserCmName, certSecretName, proxyIP, domain)
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

	f.Step(t, "Verify rgw access with public endpoint: %s", endpoint)
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
