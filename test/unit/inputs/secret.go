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
MIIJKQIBAAKCAgEA5gqIjUQXnNC67b1w33gif3c2qlWi+gPIAyKOQLLVv7my6hQr
0wEmFbXZb760Fbd4W9greihhauEpli/asn50B0z0fw+Qqgklof8v19OROOBQv9h1
/+4cw5CGYFH6mwLrO+D6SFW+ChUevPsBAnsk9ZCfUNXRly4Pbtqbs+ipkrqFP68h
lYHBR1kdW+F8cBc3lk5MrP2S8PijNkxb7ULh55jGyhjbgTuFxtw6y8Se/+g+PRLq
peBX/KO6oUOb5dOv3qepPYcdUVmoiUZC4snM+T6VI457AZxi6kap8cVvsCuQpDBa
HWie0z8PCeCt9kskVhFYECE/Yv/CxB9pc0sGio0FWAZVb8oIoDJ/yf3yxXwAFwA5
LtnerBAIkGvHd9FtyIadRKLRVDOvuoKKd3SHwlc70NuY8TSRnn4JBwPtQYLOspHT
UEsIBNNqKSwgM2hMFYzPZzfkWsaX8mrbBPIPf4tpprz/S0K9Ofpi9xypVqrUCxdK
7kGK0fRF/OzPiaP6mWFyKoiz+3FqZU+L34bnxI9EjGERaOqKM0x9SWYFWbwhkMca
C3GGxJZpzMbrGAsz/FwXs75X1szvECoDLUUNvS2OhEu106MdKHYqRiesCxMi2NqT
gbyVC+SmohDR23owRkDwLeR+1lsW4UaJR1KsEF2IsS6UYw1SQqnsdsunhK0CAwEA
AQKCAgEAiJsct/cFpqP1PZTP4ISwx8z9c21jWODB98qfeyA3+qDF9GeCFs2J1j6l
Hsy6mGLZYIEgYGx2XUfUsFE+p9yb/nHeh02w9Kh15ptpx9rlVEcw7JwYhqDaB/Bd
O/myvjafhnF1b1srfUVJeaP814JVUlZNpW00B3jcxVGgQNgbSvzkgAr6AJSLOFld
K+DdSpse8W0R73Ctv9eG5Im31U9wY13APudAAqBcMtk5OSRL32AFBbOkEFkHJwn7
nCRQAc0RlUEvKVCBQnvhr5M/yPlZdXGKkxDbTvuVadYIuYArcQyN2lK0UN594qiB
v7Xi2G2K6GloSDvWLm3/NQQKVOLqwF+tGfeA12A2lGuypNU4A7p0hx0FvqyWX3BM
pCik3fOgr3AlSuCtEMfoZ6OCn4enDgvXMz/HDNYXVz0ZKOPpXjqxOQ0KmzzctebL
N3/gUE1p1SrZPAd/2xu3JQLvF+tBPuRHwazxyFPIcYY67GcxoV7r05QhHGJN70cp
JWilNqdbpAcqHSjljEDgDDo/G7MUiyRoCcrMkpFJdbQ39GLH5PlZFS5iRZOAVMJz
nEvjcENyj8JZ3rhegmSYAlVt9TKya94n0bP7CyknX3BlMupWnGZg0FBcxFXCQTt8
QR3TFA2WxpCS2W5TAK2TN8/r1BuVayofs0Ii9oGkPVxoxmEPPikCggEBAOdQQCwZ
9qfSfSTGGIAsAtqRKNd+yYC0gDTY8vUr/Y+wYTBN2hOKNeLBpd5zgfakYK1e+bPc
vP1hYsl7hTJkfJgoDFfcbAnChxBJs5n9NEJkb7zQhbn3tN5SK0YLHYTvELX7BcMs
kidsxOe/ASSqI0ATGlxkUKbN5lCDfx9TPyfSu8PAWrR6FQeBZuIcQyKvCt7TrPwA
plbWoR4V0VCc6u1BCShOep+MIokc09nXVMv3cLMxi4oX2B7tXEdJdVPS+5uOCjku
xKCY5mLIC0JXIK8kFH5HN1oowsKgp7v8wPmFIBFW41LUj349AXv10vYxjxxKxPUE
z1p9xvnIi5/dmZMCggEBAP6XhWkZhnd882a3wYQcqW8cRlvmFqkvWsP93qfDmVqg
Zq/iOp6yZRn4W7xfqiIVLa3GYE/VEIQmKQVkl6nQo4bQkMs1GpeZv+MJGw0h0xqh
IOGRuIhpDyF4/SlI0F/58Bs7n7H0ydSmKMBO4K6Ix/uKDQCYmRUqE9La3ZDLHMO8
H4jk65TMDd2I+Z+oAq1XHSrqDM2m9vpKbD3qzNQxRZvicnFfH0YAUIvU0/zaUrty
4m4KygGrtcabAHuYPvGI29oZGwXGTfWYFIFc0HsnYNbqtoNI7mUrLxYD8qk2f0kK
Q9tQIajh5KncD+UTA3zfE5qNJsT/mEyT5PhdeGncUL8CggEAeNb+v0tNBP08bUKj
uAnF7+LXgER3BirFs1YHDrfNfgw5qZ9yJrUUU4KwoVacdXoIG2o7bpAJlyESF4nU
2q+OO3rof9niAvNB1et6zR5u96Q6j1wsECvsrBwnCS9zW2f8xeT+bKjTLY9wClVJ
RpsvUSDpq4yoaYu1Hyii931ox+gaOTg66n/Ajqw2UDdNh0gEmMXiX8ADJeh4QRxK
vh9Lx2grXYgqHUF7JUAPGIWagfehQ6vFZv3v5LBBfehNR943nVsF0juxcuiNqtsw
rpaPt49UuWeA7jPPExgUqGtxcKjwSL6ogTQURnGeXeDdNcpMJg6VeB7sKCz/DqyK
7Jg6ywKCAQA3M59ntHMlgWA9S2aYQKa0Qss2reMH+A6UJH2cnpqnvdPGGyVet4uY
X/N0GsIG9dSbs0G6zZXxMVz/oFoKJgTu/FYI2ZDUgi/LCHRnGohtY7Z/clsyqKTx
OwyZYQJdbRIUtY7gxRTmGMkJOZEaBuplrf83u96laiQ2OeKEvKWWAzpLMmeqMbxn
5oVJiuJZt2PJpEn2ZVdz2aMyobCb6bsQG794uYlMYlEUoKb+UlBR+I0EEy7Nwe9+
CqnGIrKzKFuTJJJpZCAPOlRn4DoMGfOzZd0BBlU6dmyVN3HsIrbinWktKmjB94jf
E6oWn1LIRo43mpdna4wYPpENESdEvNJ9AoIBAQDY1XasIqtvxMuLbnsfGucS3Ndo
MWEShcL+qAyVSDz7DvQ7c/MNQjwK8qbk+3rssvgIKyrER19+UjUZZquhVZtRqfWd
WNqBtpLiy2q/s0YDPFTtURjsa3E9x0bb/X4dEwQVIpHqx/B4sfgUTvgKPZMW4mRV
oejWTpJ+tnnmrG/8T5bY9teOYBZnOwN96clWE/SNgI9kud5r8w5ONiz3Cx/t9zOV
NXTzlkOyoBdc/61wmC2CgId8AFHJLim5pS40nAWnbD/M+yOL74JtvIABHGAsNDla
S88jZ5/pCve1sS6SZ2zS/PqGXT1U2QiLsDaQDWAcuXaHRX9u3R9eone11Qfn
-----END RSA PRIVATE KEY-----`

var RgwCaCert = `-----BEGIN CERTIFICATE-----
MIIE+DCCAuCgAwIBAgICB+MwDQYJKoZIhvcNAQELBQAwHTEbMBkGA1UEAxMSa3Vi
ZXJuZXRlcy1yYWRvc2d3MB4XDTI0MDcwMzEwMDE0OFoXDTI2MDcwMzEwMDE0OFow
HTEbMBkGA1UEAxMSa3ViZXJuZXRlcy1yYWRvc2d3MIICIjANBgkqhkiG9w0BAQEF
AAOCAg8AMIICCgKCAgEAwrZHhPTfQEpujKHHgUTrMMv+9Xf+eIDpwGXsfqpP3Tlq
Ozvbz4FCCacQTTj4jPbaIK5umQxXhN8orjihJvw3oobDeiBxsrThK8qGmnek0wlc
2l3KiJAdr6Ef50l8JKOOET55uzcUTkW6pwcUulbYqF++tFI6fht4fOaSKYVxQZhT
j+8/absiKaJqtnXP1iddI8zp6MRumVeZZZ9cLmxum8EQ7dlYjrxhn1sUo3aQKcq8
KxW/YLkaSTpqo5R+RvrQGk3h6Ci3a19l6zTje6ESR9/bZZ5wqwMzoWT1kwJwpkrj
mL9t9pgkb5fWRIxWPj75KZSVF0P/iXobBCkYTCTbISpH3w/yMsWAXGKE5g23Mycz
aE6CEpUoeCy8gppm17PSDyXQ4Ee/rHh8uh84k6pxmvipESW2AX1o3/ZbFErJY8qA
kWrtZsj/8Dx1yyjPG9JAd7fb6wAffqQukVIRGPqT/nyUqAmv4VH+PDvjxM1YRzad
iXhzoXWxN9nQ2z6EHLbNDjyxNTUxyW1emsC4jOharmz+4OILHMCoX/ShZ0zWFVju
3heZkzt0Q/HZIh5ilbAgyOiTRrLhWbmoVDEKWY54RLEJ2GJvLHD+hml8pyrBtoW4
PBhj9R1zFEDCI+iv4Wp2g9Poq4Kg444p7ndMKjyVhRQPVuwSl2JbrbVlg2Ji4VsC
AwEAAaNCMEAwDgYDVR0PAQH/BAQDAgKkMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0O
BBYEFINmufQqlNjd3uVCn6B0NYLOUQnaMA0GCSqGSIb3DQEBCwUAA4ICAQA3bgt0
hHcVO2csBFywO6ugzjbxnL36zdUAwRjrQN7x0UzVFghlZlTj4AAAoLkBEt53NUxs
5kWRY6VZhAJJhDoce9LoFt+BYdvRLnT2Z3bXMdzSY7nYdYJrayknqzEKJyYqFzxN
TxAcvOi9lzw2vTcQf+64WLRPN/1l8tzeB5vVNP7njr8JFmKQTTiVtJHqVwXDwqmT
gm5rDRhKtdjZALrozCwVqdF1HK6V9od6b/j99oonYyfy8dnhlLmxIrO79u3VlGD5
WiycoOLsqZlswaaAYTh/T6c6hv3YiPVKpUK7XkXAXM1IrRibdD7dJvNnErIteCvH
Kjs/EkxrvRtWtCoPu/CwVTNzlzNED/zK//a7G7wR/ed9FZFT0tkfwW34tcMowrWf
9ZMz89FohGQ3oi1iAxIt45TFGUVvLO9F+GoS7TEooQo6FUS9ew0OrpMhrkmQYnfK
Q/rJZ3wotiqApNSgQNacg0EwJRlwuepG1auJHfRgqrECAUhMcJx4DCTTG03J9G2Z
S0aLZr9pXj5H+3zMaoY/EfwM5spT6Kf57XN4sWUssB6+L56dvvdQmOmwwFJbjjLz
RL2SMQhhEqpzoHbFhyfR9cavZubREGTCeo7cArVzGVBcNr7YUVyUBqVA5EJr+Esh
AdDohVUB0KCSnoNEIxTiG6stozkygeER004B/g==
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
