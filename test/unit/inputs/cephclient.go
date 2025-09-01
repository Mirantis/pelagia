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
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var CephClientListEmpty = cephv1.CephClientList{Items: []cephv1.CephClient{}}

var CephClientListReady = cephv1.CephClientList{
	Items: []cephv1.CephClient{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "client1", Namespace: RookNamespace},
			Status:     &cephv1.CephClientStatus{Phase: cephv1.ConditionReady},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "client2", Namespace: RookNamespace},
			Status:     &cephv1.CephClientStatus{Phase: cephv1.ConditionReady},
		},
	},
}

var CephClientListNotReady = cephv1.CephClientList{
	Items: []cephv1.CephClient{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "client1", Namespace: RookNamespace},
			Status:     &cephv1.CephClientStatus{Phase: cephv1.ConditionFailure},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "client2", Namespace: RookNamespace},
		},
	},
}

func GetCephClientWithStatus(client cephv1.CephClient, ready bool) *cephv1.CephClient {
	newClient := client.DeepCopy()
	newClient.Status = &cephv1.CephClientStatus{Phase: cephv1.ConditionProgressing}
	if ready {
		newClient.Status.Phase = cephv1.ConditionReady
	}
	return newClient
}

var CephClientListOpenstack = cephv1.CephClientList{
	Items: []cephv1.CephClient{CephClientCinder, CephClientGlance, CephClientNova},
}
var CephClientListOpenstackFull = cephv1.CephClientList{
	Items: []cephv1.CephClient{CephClientCinder, CephClientGlance, CephClientNova, CephClientManila},
}

var CephClientTest = cephv1.CephClient{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "test",
	},
	Spec: cephv1.ClientSpec{
		Name: "test",
		Caps: map[string]string{
			"osd": "custom-caps",
		},
	},
}

var CephClientCinder = cephv1.CephClient{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "cinder",
	},
	Spec: cephv1.ClientSpec{
		Name: "cinder",
		Caps: map[string]string{
			"mon": "allow profile rbd",
			"osd": "profile rbd pool=volumes-hdd, profile rbd-read-only pool=images-hdd, profile rbd pool=backup-hdd",
		},
	},
}

var CephClientNova = cephv1.CephClient{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "nova",
	},
	Spec: cephv1.ClientSpec{
		Name: "nova",
		Caps: map[string]string{
			"mon": "allow profile rbd",
			"osd": "profile rbd pool=vms-hdd, profile rbd pool=images-hdd, profile rbd pool=volumes-hdd",
		},
	},
}

var CephClientGlance = cephv1.CephClient{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "glance",
	},
	Spec: cephv1.ClientSpec{
		Name: "glance",
		Caps: map[string]string{
			"mon": "allow profile rbd",
			"osd": "profile rbd pool=images-hdd",
		},
	},
}

var CephClientManila = cephv1.CephClient{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "manila",
	},
	Spec: cephv1.ClientSpec{
		Name: "manila",
		Caps: map[string]string{
			"mds": "allow rw",
			"mgr": "allow rw",
			"osd": "allow rw tag cephfs *=*",
			"mon": `allow r, allow command "auth del", allow command "auth caps", allow command "auth get", allow command "auth get-or-create"`,
		},
	},
}

var TestCephClient = cephv1.CephClient{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "test",
	},
	Spec: cephv1.ClientSpec{
		Name: "test",
		Caps: map[string]string{
			"osd": "custom-caps",
		},
	},
}
var TestCephClientReady = func() cephv1.CephClient {
	c := GetCephClientWithStatus(TestCephClient, true)
	c.Status.Info = map[string]string{"secretName": "rook-ceph-client-test"}
	return *c
}()
var TestCephClientNotReady = *GetCephClientWithStatus(TestCephClientReady, false)
