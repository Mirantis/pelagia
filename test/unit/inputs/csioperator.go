/*
Copyright 2026 Mirantis IT.

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
	csiopapi "github.com/ceph/ceph-csi-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var CsiDriversEmpty = &csiopapi.DriverList{}
var CsiDriversRook = &csiopapi.DriverList{
	Items: []csiopapi.Driver{GetCsiDriver("rook-ceph.cephfs.csi.ceph.com", BaseCephDeployment.Name), GetCsiDriver("rook-ceph.rbd.csi.ceph.com", BaseCephDeployment.Name)},
}

var OperatorConfigsEmpty = &csiopapi.OperatorConfigList{}
var OperatorConfigsRook = &csiopapi.OperatorConfigList{
	Items: []csiopapi.OperatorConfig{GetOperatorConfig("ceph-csi-operator-config", BaseCephDeployment.Name)},
}

var CephConnectionsEmpty = &csiopapi.CephConnectionList{}
var CephConnectionsRook = &csiopapi.CephConnectionList{
	Items: []csiopapi.CephConnection{CephConnection},
}

var CephConnection = csiopapi.CephConnection{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rook-ceph",
		Namespace: "rook-ceph",
	},
}

var ClientProfilesEmpty = &csiopapi.ClientProfileList{}
var ClientProfilesRook = &csiopapi.ClientProfileList{
	Items: []csiopapi.ClientProfile{ClientProfile},
}

var ClientProfile = csiopapi.ClientProfile{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rook-ceph",
		Namespace: "rook-ceph",
	},
}

func GetCsiDriver(driverName, clusterName string) csiopapi.Driver {
	return csiopapi.Driver{
		ObjectMeta: metav1.ObjectMeta{
			Name:      driverName,
			Namespace: "rook-ceph",
		},
		Spec: csiopapi.DriverSpec{
			ClusterName: &clusterName,
		},
	}
}

func GetOperatorConfig(configName, clusterName string) csiopapi.OperatorConfig {
	return csiopapi.OperatorConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configName,
			Namespace: "rook-ceph",
		},
		Spec: csiopapi.OperatorConfigSpec{
			DriverSpecDefaults: &csiopapi.DriverSpec{
				ClusterName: &clusterName,
			},
		},
	}
}
