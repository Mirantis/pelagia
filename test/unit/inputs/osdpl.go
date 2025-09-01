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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetOpenstackDeploymentStatusList(release string, state string, correctOsdplStatus bool) *unstructured.UnstructuredList {
	u := unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "lcm.mirantis.com",
		Kind:    "OpenStackDeploymentStatus",
		Version: "v1alpha1",
	})
	if correctOsdplStatus {
		status := map[string]interface{}{
			"release": release,
			"state":   state,
		}
		_ = unstructured.SetNestedMap(u.Object, status, "status", "osdpl")
	}
	_ = unstructured.SetNestedField(u.Object, "openstack", "metadata", "namespace")
	uList := unstructured.UnstructuredList{}
	uList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "lcm.mirantis.com",
		Kind:    "OpenStackDeploymentStatusList",
		Version: "v1alpha1",
	})
	uList.Items = append(uList.Items, u)
	_ = unstructured.SetNestedField(uList.Object, "openstack", "metadata", "namespace")
	uList.Items = []unstructured.Unstructured{u}
	return &uList
}
