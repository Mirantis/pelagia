/*
Copyright 2025 Mirantis IT.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless taskuired by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lcmcommon

type RgwUserKeys struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

type CSIKeyring struct {
	NodeUser        string `json:"node_user_id"`
	NodeKey         string `json:"node_key"`
	ProvisionerUser string `json:"provisioner_user_id"`
	ProvisionerKey  string `json:"provisioner_key"`
}

type CephConnection struct {
	ClientName    string `json:"client_name"`
	ClientKeyring string `json:"client_keyring"`
	FSID          string `json:"fsid"`
	MonEndpoints  string `json:"mon_endpoints_map"`
	// csi keyring info
	RBDKeyring    *CSIKeyring `json:"rbd_keyring_info,omitempty"`
	CephFSKeyring *CSIKeyring `json:"cephfs_keyring_info,omitempty"`
	// rgw admin ops user creds
	RgwAdminUserKeys *RgwUserKeys `json:"rgw_admin_keys,omitempty"`
}
