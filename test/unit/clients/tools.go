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

package helpers

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gotesting "k8s.io/client-go/testing"
)

func GetActionsCount(fakeStruct *gotesting.Fake, countActions []string) map[string]int {
	actions := map[string]int{}
	allActions := true
	if len(countActions) > 0 {
		allActions = false
		for _, action := range countActions {
			actions[action] = 0
		}
	}
	for _, action := range fakeStruct.Actions() {
		curAction := action.GetVerb()
		if allActions {
			actions[curAction]++
		} else if _, ok := actions[curAction]; ok {
			actions[curAction]++
		}
	}
	return actions
}

func GetObjectForCompareFromList(objToCompare runtime.Object, listWithItems runtime.Object) (runtime.Object, error) {
	metaObj, err := meta.Accessor(objToCompare)
	if err != nil {
		return nil, err
	}
	objs, err := meta.ExtractList(listWithItems)
	if err != nil {
		return nil, err
	}
	for _, obj := range objs {
		curMetaObj, err := meta.Accessor(obj)
		if err != nil {
			return nil, err
		}
		if curMetaObj.GetName() == metaObj.GetName() {
			if curMetaObj.GetNamespace() == metaObj.GetNamespace() {
				return obj.DeepCopyObject(), nil
			}
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    listWithItems.GetObjectKind().GroupVersionKind().Group,
		Resource: listWithItems.GetObjectKind().GroupVersionKind().Kind,
	}, metaObj.GetName())
}

func PrepareExpectedResources(input map[string]runtime.Object, output map[string]runtime.Object) map[string]runtime.Object {
	if input == nil {
		return output
	}
	if output == nil {
		output = map[string]runtime.Object{}
	}
	for k, v := range input {
		if _, ok := output[k]; !ok {
			output[k] = v.DeepCopyObject()
		}
	}
	return output
}
