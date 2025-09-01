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
	"fmt"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gotesting "k8s.io/client-go/testing"
)

func cleanupFakeClientReactions(fakeClient *gotesting.Fake) {
	fakeClient.ReactionChain = nil
	fakeClient.WatchReactionChain = nil
	fakeClient.ProxyReactionChain = nil
	fakeClient.ClearActions()
}

func addReaction(fakeClient *gotesting.Fake, action, resource string, resourcesMap map[string]runtime.Object, expectedAPIErrorsMap map[string]error) {
	defaultAPIErrors := func(reason string) map[string]error {
		return map[string]error{
			fmt.Sprintf("%s-%s", action, resource): errors.Errorf("failed to %s resource(s) kind of '%s': %s", action, resource, reason),
		}
	}
	apiErrors := defaultAPIErrors("type is not supported in fakeclient fixture")
	if resourcesMap != nil && resourcesMap[resource] != nil {
		if supportedKinds[resource] {
			apiErrors = expectedAPIErrorsMap
		}
	} else {
		apiErrors = defaultAPIErrors("list object is not specified in test")
	}
	switch action {
	case "get":
		addGetReaction(fakeClient, resource, resourcesMap, apiErrors)
	case "create":
		addCreateReaction(fakeClient, resource, resourcesMap, apiErrors)
	case "delete":
		addDeleteReaction(fakeClient, resource, resourcesMap, apiErrors)
	case "delete-collection":
		addDeleteCollectionReaction(fakeClient, resource, resourcesMap, apiErrors)
	case "update":
		addUpdateReaction(fakeClient, resource, resourcesMap, apiErrors)
	case "list":
		addListReaction(fakeClient, resource, resourcesMap)
	default:
		fmt.Printf("action '%s' is not implemented in fakeclient fixture, reaction for resource kind of '%s' can't be added", action, resource)
	}
}

func checkAPIError(apiErrorMap map[string]error, operation, resourceType, objName string) error {
	if apiErrorMap != nil {
		operationWideErr := fmt.Sprintf("%s-%s", operation, resourceType)
		if apiErrorMap[operationWideErr] != nil {
			return apiErrorMap[operationWideErr]
		}
		if objName != "" {
			operationObjectErr := fmt.Sprintf("%s-%s-%s", operation, resourceType, objName)
			if apiErrorMap[operationObjectErr] != nil {
				return apiErrorMap[operationObjectErr]
			}
		}
	}
	return nil
}

func addListReaction(fakeClient *gotesting.Fake, resource string, resourcesMap map[string]runtime.Object) {
	fakeClient.AddReactor("list", resource, func(_ gotesting.Action) (handled bool, ret runtime.Object, err error) {
		if resourcesMap != nil && resourcesMap[resource] != nil {
			// TODO: base list reaction has no support for field selector
			// as a result output will contain all objects despite field filter
			return true, resourcesMap[resource].DeepCopyObject(), nil
		}
		return true, nil, errors.Errorf("failed to list %s", resource)
	})
}

func addGetReaction(fakeClient *gotesting.Fake, resource string, resourcesMap map[string]runtime.Object, apiErrorMap map[string]error) {
	fakeClient.AddReactor("get", resource, func(action gotesting.Action) (handled bool, ret runtime.Object, err error) {
		objName := action.(gotesting.GetActionImpl).Name
		objNamespace := action.(gotesting.GetActionImpl).Namespace
		if err := checkAPIError(apiErrorMap, "get", resource, objName); err != nil {
			return true, nil, err
		}
		objs, err := meta.ExtractList(resourcesMap[resource])
		if err != nil {
			return true, nil, err
		}
		for _, obj := range objs {
			metaObj, err := meta.Accessor(obj)
			if err != nil {
				return true, nil, err
			}
			if metaObj.GetName() == objName {
				if metaObj.GetNamespace() == objNamespace {
					return true, obj.DeepCopyObject(), nil
				}
			}
		}
		return true, nil, apierrors.NewNotFound(schema.GroupResource{
			Group: resourcesMap[resource].GetObjectKind().GroupVersionKind().Group, Resource: resource}, objName)
	})
}

func addCreateReaction(fakeClient *gotesting.Fake, resource string, resourcesMap map[string]runtime.Object, apiErrorMap map[string]error) {
	fakeClient.AddReactor("create", resource, func(action gotesting.Action) (handled bool, ret runtime.Object, err error) {
		createdObj := action.(gotesting.CreateActionImpl).Object
		createdObjMeta, err := meta.Accessor(createdObj)
		if err != nil {
			return true, nil, errors.Wrap(err, "failed to access meta")
		}
		if err := checkAPIError(apiErrorMap, "create", resource, createdObjMeta.GetName()); err != nil {
			return true, nil, err
		}
		objs, err := meta.ExtractList(resourcesMap[resource])
		if err != nil {
			return true, nil, err
		}
		for _, obj := range objs {
			metaObj, err := meta.Accessor(obj)
			if err != nil {
				return true, nil, err
			}
			if metaObj.GetName() == createdObjMeta.GetName() {
				if metaObj.GetNamespace() == createdObjMeta.GetNamespace() {
					return true, nil, errors.Errorf("can't create resource %s with name %s: already exists", resource, createdObjMeta.GetName())
				}
			}
		}
		objs = append(objs, createdObj)
		err = meta.SetList(resourcesMap[resource], objs)
		if err != nil {
			return true, nil, err
		}
		return true, createdObj, nil
	})
}

func addDeleteReaction(fakeClient *gotesting.Fake, resource string, resourcesMap map[string]runtime.Object, apiErrorMap map[string]error) {
	fakeClient.AddReactor("delete", resource, func(action gotesting.Action) (handled bool, ret runtime.Object, err error) {
		objName := action.(gotesting.DeleteActionImpl).Name
		objNamespace := action.(gotesting.DeleteActionImpl).Namespace
		if err := checkAPIError(apiErrorMap, "delete", resource, objName); err != nil {
			return true, nil, err
		}
		objs, err := meta.ExtractList(resourcesMap[resource])
		if err != nil {
			return true, nil, err
		}
		for idx, obj := range objs {
			metaObj, err := meta.Accessor(obj)
			if err != nil {
				return true, nil, err
			}
			if metaObj.GetName() == objName {
				if metaObj.GetNamespace() == objNamespace {
					objs = append(objs[:idx], objs[idx+1:]...)
					err = meta.SetList(resourcesMap[resource], objs)
					return true, nil, err
				}
			}
		}
		return true, nil, apierrors.NewNotFound(schema.GroupResource{
			Group: resourcesMap[resource].GetObjectKind().GroupVersionKind().Group, Resource: resource}, objName)
	})
}

func addDeleteCollectionReaction(fakeClient *gotesting.Fake, resource string, resourcesMap map[string]runtime.Object, apiErrorMap map[string]error) {
	fakeClient.AddReactor("delete-collection", resource, func(action gotesting.Action) (handled bool, ret runtime.Object, err error) {
		if err := checkAPIError(apiErrorMap, "delete-collection", resource, ""); err != nil {
			return true, nil, err
		}
		listRestrictions := action.(gotesting.DeleteCollectionActionImpl).ListRestrictions
		objs, err := meta.ExtractList(resourcesMap[resource])
		if err != nil {
			return true, nil, err
		}
		removed := false
		newObjs := []runtime.Object{}
		for _, obj := range objs {
			metaObj, err := meta.Accessor(obj)
			if err != nil {
				return true, nil, err
			}
			matches := false
			if !listRestrictions.Fields.Empty() {
				// fields hardcoded, to actual used in object and code, no good way through tests :(
				switch castedObj := obj.(type) {
				case *corev1.Secret:
					matches = listRestrictions.Fields.Matches(fields.Set{"type": string(castedObj.Type)})
				}
			}
			if !listRestrictions.Labels.Empty() {
				matches = listRestrictions.Labels.Matches(labels.Set(metaObj.GetLabels())) || matches
			}
			if matches {
				removed = true
				continue
			}
			newObjs = append(newObjs, obj)
		}
		if removed {
			err = meta.SetList(resourcesMap[resource], newObjs)
			return true, nil, err
		}
		return true, nil, apierrors.NewNotFound(schema.GroupResource{
			Group: resourcesMap[resource].GetObjectKind().GroupVersionKind().Group, Resource: resource}, "collection")
	})
}

func addUpdateReaction(fakeClient *gotesting.Fake, resource string, resourcesMap map[string]runtime.Object, apiErrorMap map[string]error) {
	fakeClient.AddReactor("update", resource, func(action gotesting.Action) (handled bool, ret runtime.Object, err error) {
		updatedObj := action.(gotesting.UpdateActionImpl).Object
		updatedObjMeta, err := meta.Accessor(updatedObj)
		if err != nil {
			return true, nil, errors.Wrap(err, "failed to access meta")
		}
		if err := checkAPIError(apiErrorMap, "update", resource, updatedObjMeta.GetName()); err != nil {
			return true, nil, err
		}
		objs, err := meta.ExtractList(resourcesMap[resource])
		if err != nil {
			return true, nil, err
		}
		for idx, obj := range objs {
			metaObj, err := meta.Accessor(obj)
			if err != nil {
				return true, nil, err
			}
			if metaObj.GetName() == updatedObjMeta.GetName() {
				if metaObj.GetNamespace() == updatedObjMeta.GetNamespace() {
					if action.(gotesting.UpdateActionImpl).Subresource == "scale" {
						objs[idx].(*appsv1.Deployment).Spec.Replicas = &updatedObj.(*autoscalingv1.Scale).Spec.Replicas
					} else {
						objs[idx] = updatedObj
					}
					err = meta.SetList(resourcesMap[resource], objs)
					return true, nil, err
				}
			}
		}
		return true, nil, apierrors.NewNotFound(schema.GroupResource{
			Group: resourcesMap[resource].GetObjectKind().GroupVersionKind().Group, Resource: resource}, updatedObjMeta.GetName())
	})
}
