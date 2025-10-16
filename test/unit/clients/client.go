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
	claimClient "github.com/kube-object-storage/lib-bucket-provisioner/pkg/client/clientset/versioned"
	fakeclaim "github.com/kube-object-storage/lib-bucket-provisioner/pkg/client/clientset/versioned/fake"
	fakeclaimapi "github.com/kube-object-storage/lib-bucket-provisioner/pkg/client/clientset/versioned/typed/objectbucket.io/v1alpha1/fake"
	rookclient "github.com/rook/rook/pkg/client/clientset/versioned"
	fakerook "github.com/rook/rook/pkg/client/clientset/versioned/fake"
	fakecephv1 "github.com/rook/rook/pkg/client/clientset/versioned/typed/ceph.rook.io/v1/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	fakekube "k8s.io/client-go/kubernetes/fake"
	fakeappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1/fake"
	fakebatch "k8s.io/client-go/kubernetes/typed/batch/v1/fake"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	fakenetworkingv1 "k8s.io/client-go/kubernetes/typed/networking/v1/fake"
	fakestorage "k8s.io/client-go/kubernetes/typed/storage/v1/fake"
	gotesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	lcmclient "github.com/Mirantis/pelagia/pkg/client/clientset/versioned"
	fakelcm "github.com/Mirantis/pelagia/pkg/client/clientset/versioned/fake"
	fakelcmv1alpha1 "github.com/Mirantis/pelagia/pkg/client/clientset/versioned/typed/ceph.pelagia.lcm/v1alpha1/fake"
	fscheme "github.com/Mirantis/pelagia/test/unit/scheme"
)

var supportedKinds = map[string]bool{
	// lcm kinds
	"cephdeployments":            true,
	"cephdeploymentmaintenances": true,
	"cephdeploymenthealths":      true,
	"cephdeploymentsecrets":      true,
	"cephosdremovetasks":         true,
	// rook kinds
	"cephblockpools":       true,
	"cephclients":          true,
	"cephclusters":         true,
	"cephfilesystems":      true,
	"cephrbdmirrors":       true,
	"cephobjectstores":     true,
	"cephobjectstoreusers": true,
	"cephobjectrealms":     true,
	"cephobjectzonegroups": true,
	"cephobjectzones":      true,
	// claim kinds
	"objectbucketclaims": true,
	// k8s kinds
	"daemonsets":             true,
	"deployments":            true,
	"jobs":                   true,
	"configmaps":             true,
	"networkpolicies":        true,
	"nodes":                  true,
	"persistentvolumeclaims": true,
	"persistentvolumes":      true,
	"pods":                   true,
	"secrets":                true,
	"services":               true,
	"storageclasses":         true,
	"ingresses":              true,
}

func ExtendSupportedKinds(kinds map[string]bool) {
	for kind, support := range kinds {
		supportedKinds[kind] = support
	}
}

func GetClientBuilder() *fakeclient.ClientBuilder {
	return fakeclient.NewClientBuilder().WithScheme(fscheme.Scheme)
}

func GetClientBuilderWithScheme(scheme *runtime.Scheme) *fakeclient.ClientBuilder {
	return fakeclient.NewClientBuilder().WithScheme(scheme)
}

func GetClientBuilderWithObjects(objects ...client.Object) *fakeclient.ClientBuilder {
	return fakeclient.NewClientBuilder().WithScheme(fscheme.Scheme).WithStatusSubresource(objects...).WithObjects(objects...)
}

func GetClientBuilderWithSchemeWithObjects(scheme *runtime.Scheme, objects ...client.Object) *fakeclient.ClientBuilder {
	return fakeclient.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(objects...).WithObjects(objects...)
}

func GetClient(cl *fakeclient.ClientBuilder) client.WithWatch {
	if cl == nil {
		return fakeclient.NewClientBuilder().WithScheme(fscheme.Scheme).Build()
	}
	return cl.Build()
}

func GetFakeLcmclient(objects ...runtime.Object) lcmclient.Interface {
	lcm := fakelcm.NewSimpleClientset(objects...)
	lcm.ReactionChain = make([]gotesting.Reactor, 0)
	return lcm
}

func GetFakeRookclient(objects ...runtime.Object) rookclient.Interface {
	rs := fakerook.NewSimpleClientset(objects...)
	rs.ReactionChain = make([]gotesting.Reactor, 0)
	return rs
}

func GetFakeClaimclient(objects ...runtime.Object) claimClient.Interface {
	cs := fakeclaim.NewSimpleClientset(objects...)
	cs.ReactionChain = make([]gotesting.Reactor, 0)
	return cs
}

func GetFakeKubeclient(objects ...runtime.Object) kubernetes.Interface {
	ks := fakekube.NewSimpleClientset(objects...)
	ks.ReactionChain = make([]gotesting.Reactor, 0)
	return ks
}

func CleanupFakeClientReactions(clientInterface interface{}) {
	cleanupFakeClientReactions(GetFakeClientForInterface(clientInterface))
}

func GetFakeClientForInterface(clientInterface interface{}) *gotesting.Fake {
	switch clientInterfaceCasted := clientInterface.(type) {
	// lcm fake clients
	case *fakelcmv1alpha1.FakeLcmV1alpha1:
		return clientInterfaceCasted.Fake
	case lcmclient.Interface:
		return clientInterfaceCasted.LcmV1alpha1().(*fakelcmv1alpha1.FakeLcmV1alpha1).Fake
	// rook fake clients
	case *fakecephv1.FakeCephV1:
		return clientInterfaceCasted.Fake
	case rookclient.Interface:
		return clientInterfaceCasted.CephV1().(*fakecephv1.FakeCephV1).Fake
	// claim fake clients
	case *fakeclaimapi.FakeObjectbucketV1alpha1:
		return clientInterfaceCasted.Fake
	case claimClient.Interface:
		return clientInterfaceCasted.ObjectbucketV1alpha1().(*fakeclaimapi.FakeObjectbucketV1alpha1).Fake
	// k8s fake clients
	case *fakeappsv1.FakeAppsV1:
		return clientInterfaceCasted.Fake
	case *fakebatch.FakeBatchV1:
		return clientInterfaceCasted.Fake
	case *fakecorev1.FakeCoreV1:
		return clientInterfaceCasted.Fake
	case *fakenetworkingv1.FakeNetworkingV1:
		return clientInterfaceCasted.Fake
	case *fakestorage.FakeStorageV1:
		return clientInterfaceCasted.Fake
	case *gotesting.Fake:
		// support any fake clients, passed directly to helper
		return clientInterfaceCasted
	}
	// if nothing match - return empty fake client
	return &gotesting.Fake{}
}

func FakeReaction(clientInterface interface{}, action string, resourcesToReact []string, inputResources map[string]runtime.Object, apiErrors map[string]error) {
	for _, reactResource := range resourcesToReact {
		addReaction(GetFakeClientForInterface(clientInterface), action, reactResource, inputResources, apiErrors)
	}
}
