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

package input

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetExternalConnectionSecret(content []byte) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: LcmObjectMeta.Namespace,
			Name:      "pelagia-external-connection",
		},
		Data: map[string][]byte{"connection": content},
	}
}

var SecretsListEmpty = corev1.SecretList{Items: []corev1.Secret{}}

var ExternalConnectionSecretWithAdmin = GetExternalConnectionSecret(
	[]byte(`{"client_name":"admin","client_keyring":"AQAcpuJiITYXMhAAXaOoAqOKJ4mhNOAqxFb1Hw==","fsid":"8668f062-3faa-358a-85f3-f80fe6c1e306","mon_endpoints_map":"cmn01=10.0.0.1:6969,cmn02=10.0.0.2:6969,cmn03=10.0.0.3:6969"}`))

var ExternalConnectionSecretWithAdminAndRgw = GetExternalConnectionSecret(
	[]byte(`{"client_name":"admin","client_keyring":"AQAcpuJiITYXMhAAXaOoAqOKJ4mhNOAqxFb1Hw==","fsid":"8668f062-3faa-358a-85f3-f80fe6c1e306","mon_endpoints_map":"cmn01=10.0.0.1:6969,cmn02=10.0.0.2:6969,cmn03=10.0.0.3:6969","rgw_admin_keys":{"accessKey":"5TABLO7H0I6BTW6N25X5","secretKey":"Wd8SDDrtyyAuiD1klOGn9vJqOJh5dOSVlJ6kir9Q"}}`))

var ExternalConnectionSecretNonAdmin = GetExternalConnectionSecret(
	[]byte(`{"client_name":"test","client_keyring":"AQAcpuJiITYXMhAAXaOoAqOKJ4mhNOAqxFb1Hw==","fsid":"8668f062-3faa-358a-85f3-f80fe6c1e306","mon_endpoints_map":"cmn01=10.0.0.1:6969,cmn02=10.0.0.2:6969,cmn03=10.0.0.3:6969","rbd_keyring_info":{"node_user_id":"csi-rbd-node.1","node_key":"AQDd+HRjKiMBOhAATVfdzSNdlOAG3vaPSeTBzw==","provisioner_user_id":"csi-rbd-provisioner.1","provisioner_key":"AQDd+HRjFAcRIBAA102qzSI0WO1JfBnfPf/R2w=="},"cephfs_keyring_info":{"node_user_id":"csi-cephfs-node.1","node_key":"AQDh+HRjCGpLDxAA1DqwfBPBGkW7+XM65JVChg==","provisioner_user_id":"csi-cephfs-provisioner.1","provisioner_key":"AQDg+HRjKB9bLBAArfLLNtGN+KZRq4eaJf6Ptg=="},"rgw_admin_keys":{"accessKey":"5TABLO7H0I6BTW6N25X5","secretKey":"Wd8SDDrtyyAuiD1klOGn9vJqOJh5dOSVlJ6kir9Q"}}`))

var CephAdminKeyringSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "rook-ceph-admin-keyring",
	},
	Data: map[string][]byte{
		"keyring": []byte("AQAcpuJiITYXMhAAXaOoAqOKJ4mhNOAqxFb1Hw=="),
	},
}

var CSIRBDNodeSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rook-csi-rbd-node",
		Namespace: "rook-ceph",
	},
	Data: map[string][]byte{
		"userID":  []byte("csi-rbd-node.1"),
		"userKey": []byte("AQDd+HRjKiMBOhAATVfdzSNdlOAG3vaPSeTBzw=="),
	},
}

var CSIRBDProvisionerSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rook-csi-rbd-provisioner",
		Namespace: "rook-ceph",
	},
	Data: map[string][]byte{
		"userID":  []byte("csi-rbd-provisioner.1"),
		"userKey": []byte("AQDd+HRjFAcRIBAA102qzSI0WO1JfBnfPf/R2w=="),
	},
}

var CSICephFSNodeSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rook-csi-cephfs-node",
		Namespace: "rook-ceph",
	},
	Data: map[string][]byte{
		"userID":  []byte("csi-cephfs-node.1"),
		"userKey": []byte("AQDh+HRjCGpLDxAA1DqwfBPBGkW7+XM65JVChg=="),
	},
}

var CSICephFSProvisionerSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rook-csi-cephfs-provisioner",
		Namespace: "rook-ceph",
	},
	Data: map[string][]byte{
		"userID":  []byte("csi-cephfs-provisioner.1"),
		"userKey": []byte("AQDg+HRjKB9bLBAArfLLNtGN+KZRq4eaJf6Ptg=="),
	},
}

var RookCephMonSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rook-ceph-mon",
		Namespace: "rook-ceph",
	},
	Data: map[string][]byte{
		"cluster-name":  []byte("rook-ceph"),
		"fsid":          []byte("8668f062-3faa-358a-85f3-f80fe6c1e306"),
		"admin-secret":  []byte("AQAcpuJiITYXMhAAXaOoAqOKJ4mhNOAqxFb1Hw=="),
		"ceph-username": []byte("client.admin"),
		"ceph-secret":   []byte("AQAcpuJiITYXMhAAXaOoAqOKJ4mhNOAqxFb1Hw=="),
		"mon-secret":    []byte("mon-secret"),
		"ceph-args":     []byte(""),
	},
}

var RookCephMonSecretNonAdmin = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rook-ceph-mon",
		Namespace: "rook-ceph",
	},
	Data: map[string][]byte{
		"cluster-name":  []byte("rook-ceph"),
		"fsid":          []byte("8668f062-3faa-358a-85f3-f80fe6c1e306"),
		"admin-secret":  []byte("admin-secret"),
		"ceph-args":     []byte("-n client.test"),
		"ceph-username": []byte("client.test"),
		"ceph-secret":   []byte("AQAcpuJiITYXMhAAXaOoAqOKJ4mhNOAqxFb1Hw=="),
		"mon-secret":    []byte("mon-secret"),
	},
}

var RookCephRgwAdminSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rgw-admin-ops-user",
		Namespace: "rook-ceph",
	},
	Data: map[string][]byte{
		"accessKey": []byte("5TABLO7H0I6BTW6N25X5"),
		"secretKey": []byte("Wd8SDDrtyyAuiD1klOGn9vJqOJh5dOSVlJ6kir9Q"),
	},
}

var RookCephRgwMetricsSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "rgw-metrics-user-secret",
	},
	Data: map[string][]byte{
		"AccessKey": []byte("metrics-user-access-key"),
		"SecretKey": []byte("metrics-user-secret-key"),
	},
}

var CephKeysOpenstackSecretBase = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "openstack-ceph-keys",
		Namespace: "openstack-ceph-shared",
	},
	Data: map[string][]byte{
		"client.admin":  []byte("AQAcpuJiITYXMhAAXaOoAqOKJ4mhNOAqxFb1Hw=="),
		"glance":        []byte("client.glance;glance\n;images-hdd:images:hdd"),
		"nova":          []byte("client.nova;nova\n;vms-hdd:vms:hdd;volumes-hdd:volumes:hdd;images-hdd:images:hdd"),
		"cinder":        []byte("client.cinder;cinder\n;volumes-hdd:volumes:hdd;images-hdd:images:hdd;backup-hdd:backup:hdd"),
		"mon_endpoints": []byte("127.0.0.1,127.0.0.2,127.0.0.3"),
	},
}

var CephKeysOpenstackSecretRgwBase = func() corev1.Secret {
	secret := CephKeysOpenstackSecretBase.DeepCopy()
	secret.Data["rgw_internal"] = []byte("https://rook-ceph-rgw-rgw-store.rook-ceph.svc:8443/")
	secret.Data["rgw_external"] = []byte("https://rgw-store.test/")
	secret.Data["rgw_external_custom_cacert"] = []byte("spec-cacert")
	return *secret
}()

var IngressRuleSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rgw-store-ingress-secret",
		Namespace: "rook-ceph",
		Labels: map[string]string{
			"objectStore": "rgw-store",
			"ingress":     "rook-ceph-rgw-rgw-store-ingress",
			"cephdeployment.lcm.mirantis.com/ingress": "ceph-object-store-ingress",
		},
	},
	Data: map[string][]byte{
		"ca.crt":  []byte("ingress-cacert"),
		"tls.crt": []byte("ingress-crt"),
		"tls.key": []byte("ingress-key"),
	},
}

var IngressRuleSecretCustom = func() corev1.Secret {
	secret := IngressRuleSecret.DeepCopy()
	secret.Data = map[string][]byte{
		"ca.crt":  []byte("spec-cacert"),
		"tls.crt": []byte("spec-tlscert"),
		"tls.key": []byte("spec-tlskey"),
	}
	return *secret
}()

var IngressRuleSecretOpenstack = func() corev1.Secret {
	secret := IngressRuleSecret.DeepCopy()
	secret.Data = map[string][]byte{
		"ca.crt":  OpenstackRgwCredsSecret.Data["ca_cert"],
		"tls.crt": OpenstackRgwCredsSecret.Data["tls_crt"],
		"tls.key": OpenstackRgwCredsSecret.Data["tls_key"],
	}
	return *secret
}()

var OpenstackRgwCredsSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "openstack-ceph-shared",
		Name:      "openstack-rgw-creds",
	},
	Data: map[string][]byte{
		"auth_url":            []byte("https://keystone.openstack.com"),
		"username":            []byte("auth-user"),
		"password":            []byte("auth-password"),
		"project_domain_name": []byte("os-domain"),
		"project_name":        []byte("os-project"),
		"public_domain":       []byte("openstack.com"),
		"barbican_url":        []byte("https://barbican.openstack.com"),
		"tls_key":             []byte(OpenstackTLSKey),
		"tls_crt":             []byte(OpenstackTLSCert),
		"ca_cert":             []byte(OpenstackCaCert),
	},
}

var OpenstackRgwCredsSecretNoBarbican = func() corev1.Secret {
	secret := OpenstackRgwCredsSecret.DeepCopy()
	delete(secret.Data, "barbican_url")
	return *secret
}()

var OpenstackTLSKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAsNUQw41ujsb/NGSUPkVSEtHw/kJ0112kGqdhLC2fb3uBJT+0
BbRCg8TUxqnMnU3Avxh/k4pL+Z9mCI+mAvOUrc450zB5UvUoCb2DcUEN7v5Yx2PQ
2coyG2sDcDVCfPQD8mTBEnkuguMg2b9JK6H2uYDH0Wq2bf9/4mEVj0gyCUk4DJ6K
aHSY3Ju4q8ie1pZzW5A9s/rIJbPN+k0zJMuolgzhk/d5ApaOpPtTU0JPwOxf/Qg9
Qff8ExCZl3J4DS8z/HKBFp9G+pa/igOL6xwJThelKRprsa07rOL1zJcjeSLi+PQM
iRxCMNIN9xLBZX6ANJ+tV3TEOrzAIkUyxwwagwIDAQABAoIBAQCsNdOtnf8dbQ78
pzb3rerQCUT5WR8Q3lEC5B7uN0AeAdkzvWZEZ9ifGwFct+BdEWu0rtcPiI+U+ncT
v1GdbjpNSZlm4r5E3Bux4K4xjXlUVr9+7uZmM1O4/+7JSBUIO1vco+KjawCw1yEW
7gMESydMYO44NASV+00/2ex1LYoNH3K5XumbEO8dZNiGHqdQvAdNbvuinK9tpb71
hbwBblywwOp+R4Eg1uOHZfP9ZjNMKuaqEMRkfjtIti8KXEWxURICZr2rQAVuJVpr
h6UD3c8NY+vLCkPB1Su+bZdc7pZpjp09Wbm5i8Wfr6U9n7rQuaYofhWUYrqF7V7o
g/KI0T4BAoGBANpKrlzsNOrnqvpZSACxCL6t0Ug3vrSG7u+eOOHwDXOByRSfGmrt
0qYqNoqTXgqoU2BtzuNQQR32NcsXe0NbAwQaYqCsvAcAMb8mE7MngoSNm4BgaH0o
5FssFZV2JV00z8MWEQEoNUodN6ffAn8AAIFp45c72a57YduBD3qTt65zAoGBAM9g
9gF8A5HKP0WP9GcFhBx6cZcC1/QBAKFpN3rT7FuWrlpGvqNCCq1hQKnNlaaihSgI
7+Adupc76QXRXDgr0vfoQIWo+4GCeDhZRnuZChKJeUh4Ys6XD2qpIb7DZGmQOa7z
rFZzrxMc86YGVLOqhzWGoF6uePKLRrjm5OCELU+xAoGAeO26Xnv0bNXeYEYpn0hz
wb5lHA7VtQizQUdz16a2rPCPRr9FUUti0O69vFMbW+gYGGl8nW0ORdzpvBLMFGpM
527+iGho2a//3xbm/u66XVhddubxu7R1nRR0+JG07UeeeUK2NN/jdaVt+a+PoG+N
2COjE1ryorhzY7jBrHQ844UCgYEAwYQVjGURX6Z/TIZ85vX6xihsfyKkKooU8Iqi
vverg/wkTxHdK7OhCxHJqaqyj4DxCN7uGREk4aOCW292wuQCRlxweUmrCLubO9nz
L7sr7whiKQJOEcJdHIcfekgTF38ClQPGOhZRtWA67R7TQ6VJ7uTmGfRt4MefA0RT
KD+vmMECgYEAzL7MfbRi8Og3Nfrv3NCF6SHkfVlXlGPAxeB843O3z4wCUGGQPVCK
n68kTOs03+3431qel9CowDOl6144yejWz4s8tDVu2P+WmmROJRNg+WIJmHfUWuim
f+jwa47IKvTGFaxYvtO1hEzQtQiqRPoVOBlrKqnt01FQwDQvRyYsh5s=
-----END RSA PRIVATE KEY-----`

var OpenstackTLSCert = `-----BEGIN CERTIFICATE-----
MIIDpjCCAo6gAwIBAgIULkyarAPWvYgcGo/qkYDpYzGUCdYwDQYJKoZIhvcNAQEL
BQAwFTETMBEGA1UEAxMKa3ViZXJuZXRlczAeFw0yMTA0MDYxMTQ2MDBaFw0yMjA0
MDYxMTQ2MDBaMFoxCzAJBgNVBAYTAlVTMRYwFAYDVQQIEw1TYW4gRnJhbmNpc2Nv
MQswCQYDVQQHEwJDQTEmMCQGA1UEAwwdKi5vcGVuc3RhY2suc3ZjLmNsdXN0ZXIu
bG9jYWwwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCw1RDDjW6Oxv80
ZJQ+RVIS0fD+QnTXXaQap2EsLZ9ve4ElP7QFtEKDxNTGqcydTcC/GH+Tikv5n2YI
j6YC85StzjnTMHlS9SgJvYNxQQ3u/ljHY9DZyjIbawNwNUJ89APyZMESeS6C4yDZ
v0krofa5gMfRarZt/3/iYRWPSDIJSTgMnopodJjcm7iryJ7WlnNbkD2z+sgls836
TTMky6iWDOGT93kClo6k+1NTQk/A7F/9CD1B9/wTEJmXcngNLzP8coEWn0b6lr+K
A4vrHAlOF6UpGmuxrTus4vXMlyN5IuL49AyJHEIw0g33EsFlfoA0n61XdMQ6vMAi
RTLHDBqDAgMBAAGjgagwgaUwDgYDVR0PAQH/BAQDAgWgMB0GA1UdJQQWMBQGCCsG
AQUFBwMBBggrBgEFBQcDAjAMBgNVHRMBAf8EAjAAMB0GA1UdDgQWBBQM0vVvZ29h
J3351KvEHAwqBdYx5DBHBgNVHREEQDA+ggwqLmp1c3Qud29ya3OCDyouaXQuanVz
dC53b3Jrc4IdKi5vcGVuc3RhY2suc3ZjLmNsdXN0ZXIubG9jYWwwDQYJKoZIhvcN
AQELBQADggEBAESMi5DfJ+t6HKkRfzS4S52AXPW7AvheChLWf5I8dThjsdtf2k++
rIXg8XTgCn4KhfDKFXbRZEnfM/ZNAc3isCBaT2PCbanqUPmdj8T07tG+Ru1Pw0c7
a4+ti5kBB2OHwOk1I2M2JcO8y0wif2n8dABVvpezmFCEJvaQrBIf98epx9Yqqw7u
BGrPuj7DjKOMaEz7TpIpxVSKUBGKmOw84EoYdK55Zx4WoOJuN0YRyrHPmfaGarmm
fHqFdaLw0y6ysHgJdxbgYw8xZZ3IMD1ScJ9fA23Z0VJcN84/X5PH51cWup31TB6X
HwyHmvRtsbOzwVCMdLr4ZlPU1nr0eIpVR1w=
-----END CERTIFICATE-----`

var OpenstackCaCert = `-----BEGIN CERTIFICATE-----
MIICyDCCAbCgAwIBAgIBADANBgkqhkiG9w0BAQsFADAVMRMwEQYDVQQDEwprdWJl
cm5ldGVzMB4XDTE5MDcxNTEyMTczMFoXDTI5MDcxMjEyMTczMFowFTETMBEGA1UE
AxMKa3ViZXJuZXRlczCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMKs
3cxUNr7uIvIpnYAu9hM03zIgycdzar1deSIU2EZkEv20QOa6+w/yUxRhQK1pFhok
ceqVuOnGnSUW3Uf+mvAd0IgCkCUIWnDOkZE83MhKTa7FXZpEYSBpjNwziLyAffQB
IXdXs0Zf4RHoSmm2msJy2iL6tKzPWrUI8iJINKY687RubX7WTxvFUd/By6VMg1H/
F3UEi/WGBCrYbWeZhjk222N/5T1PNkhZjLM4l7ukedUaK2b9bKsRU7N/p2f9hhDJ
+x2arKmGqGkKkRQnS/19Nlwom2cSU8z36nOghJHv6hWIMbY/dUT8sGLlL/nSa2d0
38rhFhOpB/tu3Wu50NECAwEAAaMjMCEwDgYDVR0PAQH/BAQDAgKkMA8GA1UdEwEB
/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBACpTF3CTDaRUOUnIKo2p7x/HzJGM
DwrQqzKOJ1I4WzeuGVLJ4OUn9wU7jce2oMYZmOw+2opG7CvqM2xx6AQa2OxqJt8E
KoqviidpTgXqWhAaRbRGsF1IxEwEgbiI9sH0f0pQRZBVPKv+LnNlQoip9ZMgoHJ8
f25YSPKBHhMLEvCdsWy6ZRqSx7lk0NK+NiFeL3ZDlkiVveNmejM2OSzsov2wkUu3
Hvo/ugw5tNhtX7Q1iEyE2aFvsmqE1PTOE3zly7xRALAX8WYD5lNc0qGFvoh3x+79
uKVi5MUP48Voc72rJ0n7iaK8dr/byL6S5Qz2PuG9slbDQrDoacBeyjO6hiQ=
-----END CERTIFICATE-----`

var RgwCert = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`

var RgwCaCert = `-----BEGIN CERTIFICATE-----
MIIE8DCCAtigAwIBAgICB+MwDQYJKoZIhvcNAQELBQAwGTEXMBUGA1UEAxMOa3Vi
ZXJuZXRlcy1yZ3cwHhcNMjYwNzIwMDkxNjQ5WhcNMjgwNzIwMDkxNjQ5WjAZMRcw
FQYDVQQDEw5rdWJlcm5ldGVzLXJndzCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCC
AgoCggIBAMW/MDuA7kGoBZ1beb/4CR/2t8+Y3FUyILoqRSdu4b48WMhbLbzRcpTt
gO0g5dJuGW9yzSpbgbWe3VDwhGjsRTPTpUd92/mzAT5sh4sydVr1+/2jKV4ZBv2Z
2cTMZ0jMDeiXYHBX9erSYKYZlGYLWi2Sj7Wj1JUxYkE9siL69OZFCC9VspK7qJuG
kCoHoctugJsJq0dgQz1AgEW+gdQGu/4vLTkTB4LrQipcXFL1mJnkcqLwHAi8DdHB
N9PX1EvfQedrny/4nsqixjIS1Szwl5CBz9/hKgFoMf1Ps7mggBJ7c4RSR/bZ0aWi
LI8+YsOc4aZIv8ug6Sa860aI0+4d3OBgeeYukOV3WTm/my+/uy98sGdfvKENYhTW
OwPcgASu7uMS0LCpGQXlsoPAhfWh9roeOsCQn3LQSRGKyVK1t0+xkgNtPgOxNthP
5Szq7SkSPyd/I+p/9CR1zY5IKA9M06p800EwBklHJWeGZkIbDI73E2V2dPM077/D
wM/FBsJJZSKYw1AB190NMWpr7uuRMSuAuJGPdYEmJV2lxYv42YRhFOrer3RUY9vy
ZyW2M/QLXQFSxpDacalRrITD2hH+5lRoyhi7KwJ8pkmBmtBjztcPITcPD67DgGwz
ZRWLi4GCJP1oUoBu2aidPHlFlxrO644P7mw7Z5RFg5RunSsicrlTAgMBAAGjQjBA
MA4GA1UdDwEB/wQEAwICpDAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBSI+eWx
qQwYzH1EbYUuufmaPyzMZjANBgkqhkiG9w0BAQsFAAOCAgEAgD/DoE6PBWE8SyVb
85AtQn7eSB3wdZU178ofexZwuoLvC2jBtV9S+/ez1Wr9bVx9MpphvcTEFLW5NL8E
lAg3cSRJdec79dmv6V8Lyxdr05qYy7ZLHIwPW9mbIDbSTyOKgj+LVhWp1tA9zTd+
6j1lynw9hPymelKnRNMim0NajD2GOfpq/KggezwKeBF9E/76yBU7U04qDW6G7Yk8
gTzxM7TxLqy51JB+DdXOkmTq0vO8jzwfe+B08pNT3dvj+2doHpw8U/y4UGykfsit
n8v+aJz//8veKJHB9LaPFbNUWEjLwwPEQoe1zw/OZIbwNYwkT3IiuV/5jvRzQmWH
Ar5t7FHyjo/9dW8jCwjG5+dsLm3QkiutU0R5Qx70RvYNkWtQI4FIFnfaCuhDACvn
C0Q+PfZ0c+xXZx6NBzs03KBOHom2nCBoQYo/D5pY2cO0NNgOpBq51SGudh1x4r2X
sCnp2CaSCz+Ew7Ff2lrbQFthQd3xjVlsGBWHQgYj5Qr9lajQEU5Qf2bApeKkk54Y
PFm7rUyPA+EmnpfBVOQcNh444nkbHCRqJAu+NkjVMNZsvrI2wcY6yg/r4ywmDbLb
vgMscX6Fh2GW8/nX7RrQMpl2ND+wZcdc1l3d3lN+p26n+xn/I0P2OU16pZ8BlqPn
aiWJ1uk5yET4y/RiopPlcjBhxJQ=
-----END CERTIFICATE-----`

var RgwCacertExpired = `-----BEGIN CERTIFICATE-----
MIIE+DCCAuCgAwIBAgICB+MwDQYJKoZIhvcNAQELBQAwHTEbMBkGA1UEAxMSa3Vi
ZXJuZXRlcy1yYWRvc2d3MB4XDTE4MDEwMTAwMDAwMFoXDTIwMDEwMTAwMDAwMFow
HTEbMBkGA1UEAxMSa3ViZXJuZXRlcy1yYWRvc2d3MIICIjANBgkqhkiG9w0BAQEF
AAOCAg8AMIICCgKCAgEAz7CG7jDR0OWd+ifu4J9kKwlCzsPdWUZCxDWNxZOltzkP
7aOaBMqphohX2O2aY8K2ixkm69/qr27ipnF0D6PbJmF03B8ZktK7Nax6b0u6BwCM
Xm9ZPK/ELNKNSBrK6ZIfZWwJBjS/ahg2WiW5obCLrexbVaCJGP0hxFP3/vID09Fw
ITpzrMjJtgKIrVaKXkJVKzZirmgkEd0uNsKkAuzZWPtdERt0P50bZb5JjxxQxYkM
CCkmFaBvGq+UwbihZzBozoZDvQg34TMZw6VfesPtK3Bw2JLw2OfB9QdsxCsq0WH5
aLv3fvUC7VklNQlEEw1kxue8MC+hitnsuFuq3zATXJzuRJbpHpNe6oYSJ6Kw+EdR
mxWv+21Zevrl1FLmFarh+npePeG9gEJthgauCPoJDJ/F3J1trt1vg+s4YBLea7sr
PlKywnHDhP0ROx2b1HJgV75H2eBv06q4USLf+U90ZxIaSBEeOxKjEBeuMZcTQSSw
awFx8Dqif+638g10BwNK9jmCt0y8HBXYFQCQN+UkuxKyxwZ5Fd8xfDrGdSMfWb2m
zMwK/meAdN4V6mxzuOKQ3cSs7QxAf9wUiKd4sv5HU5FvFxlYZRSwsdYapso8xNOa
MGhWnKavvK3FcMfbSLIOnpZjEjywZ5GMy+3HTheLf+U356OchdVcdp/Z+8dlQmMC
AwEAAaNCMEAwDgYDVR0PAQH/BAQDAgKkMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0O
BBYEFHaDDfNYIyInjrtQ2MebVeaCQM4BMA0GCSqGSIb3DQEBCwUAA4ICAQC7v+Hz
pxSAQgOpuys966o60327iPrTmttEKpfgITqW5RVGbXutBnTh2D7TfKm90YML/Iw8
M5BcGHexlJ1XIp2TGMA5s2Yxcqtlm1GLrIDNj1AesZFR34+TEKw4qGx/nCoffieq
kaSV/SD8ngk/afCMb+AS/gM/USYJZfunk7kUyBhOS+KU0XgungzgNU6WvgY2fCkQ
8L4gS8AnYPjaispjGFohFuWEcZWS3btTj7ihuq4V5Q2E/dIH4H4oV9u2LN3OppcD
1P2ki+RBbJTmuzPPUcL+HybqQPyhJw+IvweGUbbPXWat4M4fJhj/yBCAAw1jhvOa
8YniIJYcunmdhsIAehD1mJGvellAPEWV7F5Bo4wWhvrhqJVCDczB44BqkUS8Emai
Dn21dCkELZphANwj4SSPoIw+PKWkaX9n8dQ8I/Ba2AWSSnrVORdPT+F0vXAGAkVf
Fx2qAoGCn1Hkau74Mb2TEQImpRmCzuoFEihi+UKqwpwqQMNozhWmGfe4T2H6+jtt
Yw11nwVBjVCa5Z5Zq33XPgqIcxlmLQCG1XyHIweHqhylweQ3HZ8Mx0sXorjAQ6T0
DAGOc64bUtcGnxxPlzMh8bmaoMmjwNAt1jIAQz2csanQuXU0dliryrZGkrUFoIxn
4a0Gi102j5Dq0QM79o1C/fXAn/M6ImzkUkTkNg==
-----END CERTIFICATE-----`

var RgwSSLCertSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "rgw-ssl-certificate",
	},
	Data: map[string][]byte{
		"cert":     []byte(RgwCert),
		"cacert":   []byte(RgwCaCert),
		"cabundle": []byte(RgwCaCert + "\n"),
	},
}

var RgwSSLCertExpiredSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "rgw-ssl-certificate",
	},
	Data: map[string][]byte{
		"cert":     []byte(RgwCert),
		"cacert":   []byte(RgwCacertExpired),
		"cabundle": []byte(RgwCaCert + "\n"),
	},
}

var RgwSSLCertSecretSelfSigned = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "rgw-ssl-certificate",
	},
	Data: map[string][]byte{
		"cert":     []byte("fake-keyfake-crtfake-ca"),
		"cacert":   []byte("fake-ca"),
		"cabundle": []byte("fake-ca\n"),
	},
}

var OpenstackSecretGenerated = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "openstack-ceph-keys",
		Namespace: "openstack-ceph-shared",
	},
	Data: map[string][]byte{
		"client.admin":               []byte("AQAcpuJiITYXMhAAXaOoAqOKJ4mhNOAqxFb1Hw=="),
		"glance":                     []byte("client.glance;glance\n;images-hdd:images:hdd"),
		"nova":                       []byte("client.nova;nova\n;vms-hdd:vms:hdd;volumes-hdd:volumes:hdd;images-hdd:images:hdd"),
		"cinder":                     []byte("client.cinder;cinder\n;volumes-hdd:volumes:hdd;images-hdd:images:hdd;backup-hdd:backup:hdd"),
		"mon_endpoints":              []byte("127.0.0.1,127.0.0.2,127.0.0.3"),
		"rgw_internal":               []byte("https://rook-ceph-rgw-rgw-store.rook-ceph.svc:8443/"),
		"rgw_external":               []byte("https://rgw-store.test/"),
		"rgw_internal_cacert":        []byte(RgwCaCert),
		"rgw_external_custom_cacert": []byte("spec-cacert"),
	},
}

var OpenstackSecretGeneratedCephFS = func() corev1.Secret {
	secret := OpenstackSecretGenerated.DeepCopy()
	secret.Data["manila"] = []byte("client.manila;manila\n")
	return *secret
}()

var ReconcileOpenstackSecret = func() corev1.Secret {
	secret := OpenstackSecretGenerated.DeepCopy()
	secret.Data["rgw_metrics_user_access_key"] = []byte("metrics-user-access-key")
	secret.Data["rgw_metrics_user_secret_key"] = []byte("metrics-user-secret-key")
	return *secret
}()

var CephRBDSecretList = corev1.SecretList{
	Items: []corev1.Secret{CephRBDMirrorSecret1, CephRBDMirrorSecret2},
}

var CephRBDMirrorSecret1 = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rbd-mirror-token-mirror1-pool-1",
		Namespace: "rook-ceph",
	},
	Data: map[string][]byte{
		"pool":  []byte("pool-1"),
		"token": []byte("fake-token"),
	},
	Type: "RBDPeer",
}

var CephRBDMirrorSecret2 = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rbd-mirror-token-mirror1-pool-2",
		Namespace: "rook-ceph",
	},
	Data: map[string][]byte{
		"pool":  []byte("pool-2"),
		"token": []byte("fake-token"),
	},
	Type: "RBDPeer",
}

var MultisiteCabundleSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "extra-rook-ceph-cabundle",
	},
	Data: map[string][]byte{
		"cabundle": []byte("fake-extra-cabundle"),
	},
}

var MultisiteRealmSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "realm1-keys",
	},
	Data: map[string][]byte{
		"access-key": []byte("fakekey"),
		"secret-key": []byte("fakesecret"),
	},
}
