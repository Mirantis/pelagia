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
		Labels: map[string]string{
			"app.kubernetes.io/created-by": "pelagia-deployment-controller",
			"app.kubernetes.io/managed-by": "pelagia-deployment-controller",
			"app.kubernetes.io/part-of":    "ceph.pelagia.lcm",
		},
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
		Name:      "cindervmucf-zxr9yz",
		Labels: map[string]string{
			"app.kubernetes.io/created-by":                    "pelagia-deployment-controller",
			"app.kubernetes.io/managed-by":                    "pelagia-deployment-controller",
			"app.kubernetes.io/part-of":                       "ceph.pelagia.lcm",
			"cephdeployment.lcm.mirantis.com/client-role":     "cinder",
			"cephdeployment.lcm.mirantis.com/client-rotation": "0",
		},
	},
	Spec: cephv1.ClientSpec{
		Name: "cinderVMucF_ZXr9Yz",
		Caps: map[string]string{
			"mon": "allow profile rbd",
			"osd": "profile rbd pool=volumes-hdd, profile rbd-read-only pool=images-hdd, profile rbd pool=backup-hdd",
		},
	},
}

var CephClientNova = cephv1.CephClient{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "novan9y4kl7t4vtb",
		Labels: map[string]string{
			"app.kubernetes.io/created-by":                    "pelagia-deployment-controller",
			"app.kubernetes.io/managed-by":                    "pelagia-deployment-controller",
			"app.kubernetes.io/part-of":                       "ceph.pelagia.lcm",
			"cephdeployment.lcm.mirantis.com/client-role":     "nova",
			"cephdeployment.lcm.mirantis.com/client-rotation": "0",
		},
	},
	Spec: cephv1.ClientSpec{
		Name: "novaN9y4kl7t4vTb",
		Caps: map[string]string{
			"mon": "allow profile rbd",
			"osd": "profile rbd pool=vms-hdd, profile rbd pool=images-hdd, profile rbd pool=volumes-hdd",
		},
	},
}

var CephClientGlance = cephv1.CephClient{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "glanceo1sdveeqdxxo",
		Labels: map[string]string{
			"app.kubernetes.io/created-by":                    "pelagia-deployment-controller",
			"app.kubernetes.io/managed-by":                    "pelagia-deployment-controller",
			"app.kubernetes.io/part-of":                       "ceph.pelagia.lcm",
			"cephdeployment.lcm.mirantis.com/client-role":     "glance",
			"cephdeployment.lcm.mirantis.com/client-rotation": "0",
		},
	},
	Spec: cephv1.ClientSpec{
		Name: "glanceo1sdvEEqDxxo",
		Caps: map[string]string{
			"mon": "allow profile rbd",
			"osd": "profile rbd pool=images-hdd",
		},
	},
}

var CephClientManila = cephv1.CephClient{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "manilaqnjwcflaghgc",
		Labels: map[string]string{
			"app.kubernetes.io/created-by":                    "pelagia-deployment-controller",
			"app.kubernetes.io/managed-by":                    "pelagia-deployment-controller",
			"app.kubernetes.io/part-of":                       "ceph.pelagia.lcm",
			"cephdeployment.lcm.mirantis.com/client-role":     "manila",
			"cephdeployment.lcm.mirantis.com/client-rotation": "0",
		},
	},
	Spec: cephv1.ClientSpec{
		Name: "manilaqnJWCfLAghgC",
		Caps: map[string]string{
			"mds": "allow rw",
			"mgr": "allow rw",
			"osd": "allow rw tag cephfs *=*",
			"mon": `allow r, allow command "auth del", allow command "auth caps", allow command "auth get", allow command "auth get-or-create"`,
		},
	},
}

func GetOSClientWithName(baseName string, osClient cephv1.CephClient, dropRoleLabel, dropRotationLabel bool) cephv1.CephClient {
	cl := osClient.DeepCopy()
	if dropRoleLabel {
		delete(cl.Labels, "cephdeployment.lcm.mirantis.com/client-role")
	}
	if dropRotationLabel {
		delete(cl.Labels, "cephdeployment.lcm.mirantis.com/client-rotation")
	}
	cl.Name = baseName
	cl.Spec.Name = baseName
	return *cl
}

func BumpRotationID(osClient cephv1.CephClient, id string) cephv1.CephClient {
	cl := osClient.DeepCopy()
	cl.Labels["cephdeployment.lcm.mirantis.com/client-rotation"] = id
	return *cl
}

func RandomizeStub(s string) string {
	switch s {
	case "cinder":
		return "cinderVMucF_ZXr9Yz"
	case "glance":
		return "glanceo1sdvEEqDxxo"
	case "nova":
		return "novaN9y4kl7t4vTb"
	case "manila":
		return "manilaqnJWCfLAghgC"
	}
	return s
}

var TestCephClient = cephv1.CephClient{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "rook-ceph",
		Name:      "test",
		Labels: map[string]string{
			"app.kubernetes.io/created-by": "pelagia-deployment-controller",
			"app.kubernetes.io/managed-by": "pelagia-deployment-controller",
			"app.kubernetes.io/part-of":    "ceph.pelagia.lcm",
		},
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
