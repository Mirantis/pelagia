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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ConfigMapListEmpty = &corev1.ConfigMapList{Items: []corev1.ConfigMap{}}
var ConfigMapList = &corev1.ConfigMapList{Items: []corev1.ConfigMap{*RookOperatorConfigMapBase}}

var RookOperatorConfigMapBase = RookOperatorConfig(nil)

func RookOperatorConfig(parameters map[string]string) *corev1.ConfigMap {
	// default values for unit tests if overrides not passed
	configMapParams := map[string]string{
		"ROOK_CSI_ENABLE_RBD":    "true",
		"ROOK_CSI_ENABLE_CEPHFS": "true",
	}
	if len(parameters) > 0 {
		for k, v := range parameters {
			configMapParams[k] = v
		}
	}
	return GetConfigMap("rook-ceph-operator-config", RookNamespace, configMapParams)
}

func GetConfigMap(name, namespace string, params map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: params,
	}
}

var PelagiaConfig = corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: LcmObjectMeta.Namespace,
		Name:      "pelagia-lcmconfig",
	},
	Data: map[string]string{
		"DEPLOYMENT_CEPH_IMAGE":                      fmt.Sprintf("mirantis.azurecr.io/ceph/ceph:%s", LatestCephVersionImage),
		"DEPLOYMENT_ROOK_IMAGE":                      "mirantis.azurecr.io/mirantis/rook:v1.17.4-15",
		"DEPLOYMENT_OPENSTACK_CEPH_SHARED_NAMESPACE": "openstack-ceph-shared",
		"DEPLOYMENT_NETPOL_ENABLED":                  "true",
		"DEPLOYMENT_LOG_LEVEL":                       "trace",
	},
}

var PelagiaConfigForPrevCephVersion = corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: LcmObjectMeta.Namespace,
		Name:      "pelagia-lcmconfig",
	},
	Data: map[string]string{
		"DEPLOYMENT_CEPH_RELEASE":                    PreviousCephVersion,
		"DEPLOYMENT_CEPH_IMAGE":                      fmt.Sprintf("mirantis.azurecr.io/ceph/ceph:%s", PreviousCephVersionImage),
		"DEPLOYMENT_ROOK_IMAGE":                      "mirantis.azurecr.io/mirantis/rook:v1.16.7-1",
		"DEPLOYMENT_OPENSTACK_CEPH_SHARED_NAMESPACE": "openstack-ceph-shared",
		"DEPLOYMENT_NETPOL_ENABLED":                  "true",
		"DEPLOYMENT_LOG_LEVEL":                       "trace",
	},
}

var RookCephMonEndpoints = corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "rook-ceph-mon-endpoints",
	},
	Data: map[string]string{
		"data": "a=127.0.0.1,b=127.0.0.2,c=127.0.0.3",
	},
}

var RookCephMonEndpointsExternal = corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "rook-ceph-mon-endpoints",
	},
	Data: map[string]string{
		"data":     "cmn01=10.0.0.1:6969,cmn02=10.0.0.2:6969,cmn03=10.0.0.3:6969",
		"mapping":  "{}",
		"maxMonId": "3",
	},
}

var EmptyRookConfigOverride = corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "rook-config-override",
	},
	Data: map[string]string{
		"config":  "",
		"runtime": "",
	},
}

var BaseRookConfigOverride = corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "rook-config-override",
		Annotations: map[string]string{
			"cephdeployment.lcm.mirantis.com/config-global-hash": "95b401f9fc7db148cf2cc3bbcbbe09f7722b2060acf714c142fdf07ee249f0bb",
			"cephdeployment.lcm.mirantis.com/config-mon-hash":    "52235ccf3c9f953de0fc2b8e2928f8119e1be19c14a4cf300c55e8498ec81fa2",
		},
	},
	Data: map[string]string{
		"config": `[global]
cluster_network = 127.0.0.0/16
public_network = 192.168.0.0/16
mon_max_pg_per_osd = 300
mon_target_pg_per_osd = 100

[mon]
mon_warn_on_insecure_global_id_reclaim = false
mon_warn_on_insecure_global_id_reclaim_allowed = false

[osd]
osd_class_dir = /usr/lib64/rados-classes
`,
		"runtime": "osd|bdev_async_discard_threads = 1\nosd|bdev_enable_discard = true\n",
	},
}
