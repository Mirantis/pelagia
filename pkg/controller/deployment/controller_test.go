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

package deployment

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	lcmconfig "github.com/Mirantis/pelagia/pkg/controller/config"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
	fscheme "github.com/Mirantis/pelagia/test/unit/scheme"
)

func FakeReconciler() *ReconcileCephDeployment {
	return &ReconcileCephDeployment{
		Config:           &rest.Config{},
		Client:           faketestclients.GetClient(nil),
		Kubeclientset:    faketestclients.GetFakeKubeclient(),
		Rookclientset:    faketestclients.GetFakeRookclient(),
		CephLcmclientset: faketestclients.GetFakeLcmclient(),
		Claimclientset:   faketestclients.GetFakeClaimclient(),
		Scheme:           fscheme.Scheme,
	}
}

func fakeDeploymentConfig(dconfig *deployConfig, lcmConfigData map[string]string) *cephDeploymentConfig {
	if lcmConfigData == nil {
		lcmConfigData = map[string]string{}
	}
	lcmConfigData["DEPLOYMENT_LOG_LEVEL"] = "TRACE"
	lcmConfig := lcmconfig.ReadConfiguration(log.With().Str(lcmcommon.LoggerObjectField, "configmap").Logger(), lcmConfigData)
	sublog := log.With().Str(lcmcommon.LoggerObjectField, "cephdeployment").Logger().Level(lcmConfig.DeployParams.LogLevel)
	dc := deployConfig{
		cephDpl: &cephlcmv1alpha1.CephDeployment{ObjectMeta: unitinputs.LcmObjectMeta},
	}
	if dconfig != nil {
		dc = *dconfig
	}
	return &cephDeploymentConfig{
		context:   context.TODO(),
		api:       FakeReconciler(),
		log:       &sublog,
		lcmConfig: &lcmConfig,
		cdConfig:  dc,
	}
}

func TestStatusUpdatePhases(t *testing.T) {
	r := FakeReconciler()
	request := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: unitinputs.LcmObjectMeta.Namespace, Name: unitinputs.LcmObjectMeta.Name}}

	doReconcile := func(expectedPhase cephlcmv1alpha1.CephDeploymentPhase, expectedMsg string) {
		_, err := r.Reconcile(context.TODO(), request)
		assert.Nil(t, err)
		cephDpl := &cephlcmv1alpha1.CephDeployment{}
		err = r.Client.Get(context.Background(), client.ObjectKey{Name: request.Name, Namespace: request.Namespace}, cephDpl)
		assert.Nil(t, err)
		assert.Equal(t, expectedPhase, cephDpl.Status.Phase)
		assert.Equal(t, expectedMsg, cephDpl.Status.Message)
	}
	// check that update to failed is done only after 3 tries
	mc := *unitinputs.CephDeployNonMosk.DeepCopy()
	clientMc := unitinputs.CephDeployNonMosk.DeepCopy()
	inputs := map[string]runtime.Object{"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{mc, mc}}}
	faketestclients.FakeReaction(r.CephLcmclientset, "list", []string{"cephdeployments"}, inputs, nil)
	r.Client = faketestclients.GetClient(faketestclients.GetClientBuilder().WithStatusSubresource(clientMc).WithObjects(clientMc))
	for i := 0; i <= 4; i++ {
		if i == 3 {
			doReconcile(cephlcmv1alpha1.PhaseFailed, "incorrect number of CephDeployments in lcm-namespace namespace")
			clientMc.Status.Phase = cephlcmv1alpha1.PhaseReady
			clientMc.Status.Message = ""
			r.Client = faketestclients.GetClient(faketestclients.GetClientBuilder().WithStatusSubresource(clientMc).WithObjects(clientMc))
		} else if i < 3 {
			doReconcile(cephlcmv1alpha1.CephDeploymentPhase(""), "")
		} else {
			doReconcile(cephlcmv1alpha1.PhaseReady, "")
		}
	}
	// check that update to failed is not happend for maintenance/request processing phases
	clientMc.Status.Phase = cephlcmv1alpha1.PhaseOnHold
	r.Client = faketestclients.GetClient(faketestclients.GetClientBuilder().WithStatusSubresource(clientMc).WithObjects(clientMc))
	doReconcile(cephlcmv1alpha1.PhaseOnHold, "")
	clientMc.Status.Phase = cephlcmv1alpha1.PhaseMaintenance
	r.Client = faketestclients.GetClient(faketestclients.GetClientBuilder().WithStatusSubresource(clientMc).WithObjects(clientMc))
	doReconcile(cephlcmv1alpha1.PhaseMaintenance, "")
}

var cephAPIResources = []string{"cephclusters", "cephblockpools", "cephclients", "cephfilesystems", "cephrbdmirrors", "cephobjectstores", "cephobjectstoreusers"}

func TestReconcile(t *testing.T) {
	r := FakeReconciler()
	request := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: unitinputs.LcmObjectMeta.Namespace, Name: unitinputs.LcmObjectMeta.Name}}
	requeueAfterInterval := reconcile.Result{RequeueAfter: requeueAfterInterval}
	noRequeue := reconcile.Result{}
	immediateRequeue := reconcile.Result{Requeue: true}
	latestClusterVersion := &lcmcommon.CephVersion{
		Name:            "Squid",
		MajorVersion:    "v19.2",
		MinorVersion:    "3",
		Order:           19,
		SupportedMinors: []string{"3"},
	}
	tests := []struct {
		name            string
		result          reconcile.Result
		inputResources  map[string]runtime.Object
		apiErrors       map[string]error
		testclient      *fakeclient.ClientBuilder
		expectedError   string
		expectedVersion *lcmcommon.CephVersion
		expectedStatus  *cephlcmv1alpha1.CephDeploymentStatus
	}{
		{
			name:          "reconcile cephdeployment - list cephdeployment failed",
			expectedError: "failed to list CephDeployments lcm-namespace namespace: failed to list cephdeployments",
			result:        requeueAfterInterval,
		},
		{
			name: "reconcile cephdeployment - list cephdeployment empty",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{}},
			},
			result: noRequeue,
		},
		{
			name: "reconcile cephdeployment - cephdeployment number > 1, status update succeed",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{
					Items: []cephlcmv1alpha1.CephDeployment{unitinputs.BaseCephDeployment, unitinputs.BaseCephDeployment},
				},
			},
			testclient: faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseFailed,
				Message: "incorrect number of CephDeployments in lcm-namespace namespace",
				LastRun: "2021-08-15T14:30:12+04:00",
			},
			result: requeueAfterInterval,
		},
		{
			name: "reconcile cephdeployment - cephdeployment number > 1, status update failed",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{
					Items: []cephlcmv1alpha1.CephDeployment{unitinputs.BaseCephDeployment, unitinputs.BaseCephDeployment},
				},
			},
			result: requeueAfterInterval,
		},
		{
			name: "reconcile cephdeployment - cephdeployment name not equals to request name",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{
					Items: []cephlcmv1alpha1.CephDeployment{{ObjectMeta: metav1.ObjectMeta{Name: "fake-cephdeployment"}}},
				},
			},
			expectedError: "incorrect CephDeployment object",
			result:        requeueAfterInterval,
		},
		{
			name: "reconcile cephdeployment - failed to build cephdeployment nodes",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.CephDeployWithWrongNodes}},
				"configmaps":      &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig}},
			},
			testclient: faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.CephDeployWithWrongNodes.DeepCopy()).WithObjects(unitinputs.CephDeployWithWrongNodes.DeepCopy()),
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseFailed,
				Message: "failed to expand node list for CephDeployment lcm-namespace/cephcluster",
				LastRun: "2021-08-15T14:30:15+04:00",
			},
			result: requeueAfterInterval,
		},
		{
			name: "reconcile cephdeployment - cephdeployment is going to be removed, but prevent is set",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{
					func() cephlcmv1alpha1.CephDeployment {
						mc := unitinputs.BaseCephDeploymentDelete.DeepCopy()
						mc.Spec.ExtraOpts = &cephlcmv1alpha1.CephDeploymentExtraOpts{PreventClusterDestroy: true}
						return *mc
					}(),
				}},
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig}},
				"nodes":      &corev1.NodeList{},
			},
			testclient: faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase: cephlcmv1alpha1.PhaseFailed,
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  cephlcmv1alpha1.ValidationFailed,
					LastValidatedGeneration: int64(0),
					Messages: []string{
						"CephDeployment has no default pool specified",
						"The following nodes are present in CephDeployment spec but not present in k8s cluster node list: node-1,node-2,node-3",
					},
				},
				LastRun: "2021-08-15T14:30:16+04:00",
			},
			result: immediateRequeue,
		},
		{
			name: "reconcile cephdeployment - delete started, update status success",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.BaseCephDeploymentDelete}},
				"configmaps":      &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig}},
				"cephclusters":    &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterOpenstack()}},
			},
			testclient: faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeploymentDelete.DeepCopy()).WithObjects(unitinputs.BaseCephDeploymentDelete.DeepCopy()),
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseDeleting,
				Message: "Ceph cluster deletion is in progress",
				LastRun: "2021-08-15T14:30:17+04:00",
			},
			result: immediateRequeue,
		},
		{
			name: "reconcile cephdeployment - delete started, update status failed",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.BaseCephDeploymentDelete}},
				"configmaps":      &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig}},
				"cephclusters":    &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterOpenstack()}},
			},
			result: requeueAfterInterval,
		},
		{
			name: "reconcile cephdeployment - delete failed",
			inputResources: map[string]runtime.Object{
				"cephdeployments":      &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.BaseCephDeploymentDeleting}},
				"configmaps":           &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig}},
				"nodes":                &corev1.NodeList{},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephclusters":         &cephv1.CephClusterList{},
				"cephblockpools":       &cephv1.CephBlockPoolList{},
				"cephclients":          &cephv1.CephClientList{},
				"cephfilesystems":      &cephv1.CephFilesystemList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{},
			},
			testclient: faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeploymentDeleting.DeepCopy()).WithObjects(unitinputs.BaseCephDeploymentDeleting.DeepCopy()),
			apiErrors:  map[string]error{"delete-cephdeploymenthealths": errors.New("cephdeploymenthealth delete failed")},
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseDeleting,
				Message: "Ceph cluster is failing to remove",
				LastRun: "2021-08-15T14:30:19+04:00",
			},
			result: requeueAfterInterval,
		},
		{
			name: "reconcile cephdeployment - delete in progress, status update succeed",
			inputResources: map[string]runtime.Object{
				"cephdeployments":            &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.BaseCephDeploymentDeleting}},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{},
				"cephdeploymentmaintenances": &cephlcmv1alpha1.CephDeploymentMaintenanceList{},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{},
				"configmaps":                 &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig}},
				"cephclusters":               &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterOpenstack()}},
				"secrets":                    &corev1.SecretList{},
				"nodes":                      &corev1.NodeList{},
				"daemonsets":                 &appsv1.DaemonSetList{},
				"deployments":                &appsv1.DeploymentList{},
				"storageclasses":             &storagev1.StorageClassList{},
				"cephblockpools":             &cephv1.CephBlockPoolList{},
				"cephclients":                &cephv1.CephClientList{},
				"cephfilesystems":            &cephv1.CephFilesystemList{},
				"cephrbdmirrors":             &cephv1.CephRBDMirrorList{},
				"cephobjectstores":           &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers":       &cephv1.CephObjectStoreUserList{},
				"networkpolicies":            &networkingv1.NetworkPolicyList{},
			},
			testclient: faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeploymentDeleting.DeepCopy()).WithObjects(unitinputs.BaseCephDeploymentDeleting.DeepCopy()),
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseDeleting,
				Message: "Ceph cluster deletion is in progress",
				LastRun: "2021-08-15T14:30:20+04:00",
			},
			result: requeueAfterInterval,
		},
		{
			name: "reconcile cephdeployment - delete in progress, status update failed",
			inputResources: map[string]runtime.Object{
				"cephdeployments":      &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.BaseCephDeploymentDeleting}},
				"configmaps":           &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig}},
				"cephclusters":         &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterOpenstack()}},
				"nodes":                &corev1.NodeList{},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephblockpools":       &cephv1.CephBlockPoolList{},
				"cephclients":          &cephv1.CephClientList{},
				"cephfilesystems":      &cephv1.CephFilesystemList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{},
			},
			result: requeueAfterInterval,
		},
		{
			name: "reconcile cephdeployment - delete succeed, finalizer remove failed",
			inputResources: map[string]runtime.Object{
				"cephdeployments":            &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.BaseCephDeploymentDeleting}},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{},
				"cephdeploymentmaintenances": &cephlcmv1alpha1.CephDeploymentMaintenanceList{},
				"configmaps":                 &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig}},
				"cephclusters":               &cephv1.CephClusterList{},
				"secrets":                    &corev1.SecretList{},
				"nodes":                      &corev1.NodeList{},
				"daemonsets":                 &appsv1.DaemonSetList{},
				"deployments":                &appsv1.DeploymentList{},
				"storageclasses":             &storagev1.StorageClassList{},
				"cephblockpools":             &cephv1.CephBlockPoolList{},
				"cephclients":                &cephv1.CephClientList{},
				"cephfilesystems":            &cephv1.CephFilesystemList{},
				"cephrbdmirrors":             &cephv1.CephRBDMirrorList{},
				"cephobjectstores":           &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers":       &cephv1.CephObjectStoreUserList{},
				"networkpolicies":            &networkingv1.NetworkPolicyList{},
			},
			testclient: faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeploymentDeleting.DeepCopy()).WithObjects(unitinputs.BaseCephDeploymentDeleting.DeepCopy()),
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseDeleting,
				Message: "Ceph cluster is removed, failed to cleanup CephDeployment",
				LastRun: "2021-08-15T14:30:22+04:00",
			},
			apiErrors: map[string]error{"update-cephdeployments": errors.New("update failed")},
			result:    requeueAfterInterval,
		},
		{
			name: "reconcile cephdeployment - delete succeed, finalized removed",
			inputResources: map[string]runtime.Object{
				"cephdeployments":            &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.BaseCephDeploymentDeleting}},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{},
				"cephdeploymentmaintenances": &cephlcmv1alpha1.CephDeploymentMaintenanceList{},
				"configmaps":                 &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig}},
				"secrets":                    &corev1.SecretList{},
				"nodes":                      &corev1.NodeList{},
				"daemonsets":                 &appsv1.DaemonSetList{},
				"deployments":                &appsv1.DeploymentList{},
				"storageclasses":             &storagev1.StorageClassList{},
				"cephblockpools":             &cephv1.CephBlockPoolList{},
				"cephclients":                &cephv1.CephClientList{},
				"cephclusters":               &cephv1.CephClusterList{},
				"cephfilesystems":            &cephv1.CephFilesystemList{},
				"cephrbdmirrors":             &cephv1.CephRBDMirrorList{},
				"cephobjectstores":           &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers":       &cephv1.CephObjectStoreUserList{},
				"networkpolicies":            &networkingv1.NetworkPolicyList{},
			},
			result: noRequeue,
		},
		{
			name: "reconcile cephdeployment - validation failed",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.BaseCephDeployment}},
				"configmaps":      &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig}},
				"nodes":           &corev1.NodeList{},
			},
			testclient: faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase: cephlcmv1alpha1.PhaseFailed,
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  cephlcmv1alpha1.ValidationFailed,
					LastValidatedGeneration: int64(0),
					Messages: []string{
						"CephDeployment has no default pool specified",
						"The following nodes are present in CephDeployment spec but not present in k8s cluster node list: node-1,node-2,node-3",
					},
				},
				LastRun: "2021-08-15T14:30:24+04:00",
			},
			result: immediateRequeue,
		},
		{
			name: "reconcile cephdeployment - failed to add finalizer",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{
					func() cephlcmv1alpha1.CephDeployment {
						mc := unitinputs.CephDeployNonMosk.DeepCopy()
						mc.Finalizers = nil
						return *mc
					}(),
				}},
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig}},
				"nodes":      &corev1.NodeList{},
			},
			apiErrors: map[string]error{"update-cephdeployments": errors.New("update failed")},
			result:    requeueAfterInterval,
		},
		{
			name: "reconcile cephdeployment - add finalizer succeed",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{
					func() cephlcmv1alpha1.CephDeployment {
						mc := unitinputs.CephDeployNonMosk.DeepCopy()
						mc.Finalizers = nil
						return *mc
					}(),
				}},
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig}},
				"nodes":      &corev1.NodeList{Items: []corev1.Node{unitinputs.GetAvailableNode("node-1"), unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
			},
			result: immediateRequeue,
		},
		{
			name: "reconcile cephdeployment - ceph version check images are not consistent",
			inputResources: map[string]runtime.Object{
				"cephdeployments":            &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.CephDeployNonMosk}},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{},
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
				"secrets":                    &corev1.SecretList{},
				"configmaps":                 &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig, unitinputs.RookCephMonEndpoints}},
				"nodes":                      &corev1.NodeList{Items: []corev1.Node{unitinputs.GetAvailableNode("node-1"), unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
				"cephclusters":               &cephv1.CephClusterList{},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentPrevVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"daemonsets": &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"pods":       unitinputs.ToolBoxPodList,
			},
			testclient:      faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			expectedVersion: latestClusterVersion,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseFailed,
				Message: "failed to ensure consistent Rook image version: deployment rook-ceph/rook-ceph-operator rook image update is in progress",
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 10,
				},
				LastRun:     "2021-08-15T14:30:27+04:00",
				ObjectsRefs: unitinputs.CephDeploymentObjectsRefs,
			},
			result: requeueAfterInterval,
		},
		{
			name: "reconcile cephdeployment - maintenance mode is set",
			inputResources: map[string]runtime.Object{
				"cephdeployments":            &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.CephDeployNonMosk}},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{},
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListActing,
				"configmaps":                 &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig, unitinputs.BaseRookConfigOverride}},
				"nodes":                      &corev1.NodeList{Items: []corev1.Node{unitinputs.GetAvailableNode("node-1"), unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
				"cephclusters":               &cephv1.CephClusterList{},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"daemonsets":         &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"pods":               unitinputs.ToolBoxPodList,
				"cephosdremovetasks": &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{}},
			},
			testclient:      faketestclients.GetClientBuilderWithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			expectedVersion: latestClusterVersion,
			result:          requeueAfterInterval,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseMaintenance,
				Message: "Cluster maintenance (upgrade) detected, reconcile is paused",
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 10,
				},
				LastRun:     "2021-08-15T14:30:28+04:00",
				ObjectsRefs: unitinputs.CephDeploymentObjectsRefs,
			},
		},
		{
			name: "reconcile cephdeployment - failed apply configuration",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.CephDeployNonMosk}},
				"configmaps":      &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig, unitinputs.BaseRookConfigOverride}},
				"nodes":           &corev1.NodeList{Items: []corev1.Node{unitinputs.GetAvailableNode("node-1"), unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
				"cephclusters":    &cephv1.CephClusterList{},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"daemonsets":         &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"pods":               unitinputs.ToolBoxPodList,
				"cephosdremovetasks": &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{}},
				"networkpolicies":    &networkingv1.NetworkPolicyList{},
			},
			testclient:      faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			expectedVersion: latestClusterVersion,
			apiErrors: map[string]error{
				"get-networkpolicies": errors.New("get networkpolicy failed"),
				"update-nodes":        errors.New("update node failed"),
			},
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseFailed,
				Message: "Ceph cluster failed to check CephDeploymentHealth presence, failed to check CephDeploymentSecret presence, failed to check CephDeploymentMaintenance presence",
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 10,
				},
				LastRun:     "2021-08-15T14:30:29+04:00",
				ObjectsRefs: unitinputs.CephDeploymentObjectsRefs,
			},
			result: requeueAfterInterval,
		},
		{
			name: "reconcile cephdeployment - create non-mosk ceph cluster",
			inputResources: map[string]runtime.Object{
				"cephdeployments":            &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.CephDeployNonMosk}},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{}},
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{Items: []cephlcmv1alpha1.CephDeploymentHealth{}},
				"cephosdremovetasks":         &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{}},
				"configmaps":                 &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig, unitinputs.RookCephMonEndpoints, unitinputs.BaseRookConfigOverride}},
				"secrets":                    &corev1.SecretList{},
				"ingresses":                  &networkingv1.IngressList{},
				"networkpolicies":            &networkingv1.NetworkPolicyList{},
				"nodes":                      &corev1.NodeList{Items: []corev1.Node{unitinputs.GetAvailableNode("node-1"), unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"daemonsets":           &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"pods":                 unitinputs.ToolBoxPodList,
				"storageclasses":       &storagev1.StorageClassList{},
				"cephblockpools":       &cephv1.CephBlockPoolList{},
				"cephclients":          &cephv1.CephClientList{},
				"cephclusters":         &cephv1.CephClusterList{},
				"cephrbdmirrors":       &cephv1.CephRBDMirrorList{},
				"cephfilesystems":      &cephv1.CephFilesystemList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{},
			},
			testclient:      faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			expectedVersion: latestClusterVersion,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseDeploying,
				Message: "Ceph cluster configuration apply is in progress: label nodes, network policies",
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 10,
				},
				LastRun:     "2021-08-15T14:30:30+04:00",
				ObjectsRefs: unitinputs.CephDeploymentObjectsRefs,
			},
			result: requeueAfterInterval,
		},
		{
			name: "reconcile cephdeployment - cephdeployment status has no version, but cluster is already deploying",
			inputResources: map[string]runtime.Object{
				"cephdeployments":            &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.CephDeployNonMosk}},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.EmptyCephSecret}},
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
				"cephosdremovetasks":         &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{}},
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{
					unitinputs.PelagiaConfig, unitinputs.RookCephMonEndpoints, unitinputs.BaseRookConfigOverride,
				}},
				"nodes": &corev1.NodeList{Items: []corev1.Node{unitinputs.GetAvailableNode("node-1"), unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"pods":         unitinputs.ToolBoxPodList,
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{unitinputs.TestCephCluster}},
			},
			testclient:      faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			expectedVersion: latestClusterVersion,
			result:          immediateRequeue,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 10,
				},
				ClusterVersion: "v19.2.3",
				LastRun:        "2021-08-15T14:30:31+04:00",
				ObjectsRefs:    unitinputs.CephDeploymentObjectsRefs,
			},
		},
		{
			name: "reconcile cephdeployment - create non-mosk ceph cluster is in progress",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{
					*unitinputs.GetUpdatedClusterVersionCephDeploy(unitinputs.CephDeployNonMosk.DeepCopy(), unitinputs.LatestCephVersionImage)}},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{Items: []cephlcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth}},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.EmptyCephSecret}},
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
				"cephosdremovetasks":         &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{}},
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{
					unitinputs.PelagiaConfig, unitinputs.RookCephMonEndpoints, *unitinputs.BaseRookConfigOverride.DeepCopy(),
				}},
				"ingresses": &networkingv1.IngressList{},
				"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{
					unitinputs.NetworkPolicyMds, unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw,
				}},
				"secrets": &corev1.SecretList{Items: []corev1.Secret{*unitinputs.RgwSSLCertSecret.DeepCopy(), unitinputs.RookCephMonSecret}},
				"nodes":   &corev1.NodeList{Items: []corev1.Node{unitinputs.GetAvailableNode("node-1"), unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"daemonsets":     &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"pods":           unitinputs.ToolBoxPodList,
				"storageclasses": &storagev1.StorageClassList{},
				"cephblockpools": &cephv1.CephBlockPoolList{Items: []cephv1.CephBlockPool{
					unitinputs.GetCephBlockPoolWithStatus(unitinputs.CephBlockPoolReplicated, true),
					func() cephv1.CephBlockPool {
						pool := unitinputs.GetCephBlockPoolWithStatus(unitinputs.CephBlockPoolReplicated, true)
						pool.Name = "test-cephfs-some-pool-name"
						return pool
					}(),
				}},
				"cephclients":          &cephv1.CephClientList{Items: []cephv1.CephClient{}},
				"cephclusters":         &cephv1.CephClusterList{Items: []cephv1.CephCluster{unitinputs.TestCephCluster}},
				"cephfilesystems":      unitinputs.CephFSListReady.DeepCopy(),
				"cephobjectstores":     &cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase.DeepCopy()}},
				"cephrbdmirrors":       &cephv1.CephRBDMirrorList{},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{Items: []cephv1.CephObjectStoreUser{*unitinputs.RgwUserWithStatus(unitinputs.RgwUserBase, "Ready")}},
			},
			testclient:      faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			expectedVersion: latestClusterVersion,
			result:          requeueAfterInterval,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseDeploying,
				Message: "Ceph cluster configuration apply is in progress: label nodes, cephcluster, cephblockpools, shared filesystems, storageclasses, cephclients, ceph object storage, cluster state",
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 10,
				},
				ClusterVersion: "v19.2.3",
				LastRun:        "2021-08-15T14:30:32+04:00",
				ObjectsRefs:    unitinputs.CephDeploymentObjectsRefs,
			},
		},
		{
			name: "reconcile cephdeployment - configuration non-mosk ceph cluster is done",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{
					func() cephlcmv1alpha1.CephDeployment {
						mc := unitinputs.CephDeployNonMosk.DeepCopy()
						mc.Spec.ObjectStorage = nil
						mc.Status.ClusterVersion = unitinputs.LatestCephVersionImage
						return *mc
					}(),
				}},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{Items: []cephlcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth}},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.EmptyCephSecret}},
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
				"cephosdremovetasks":         &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{}},
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{
					unitinputs.PelagiaConfig, unitinputs.RookCephMonEndpoints,
					func() corev1.ConfigMap {
						cm := unitinputs.BaseRookConfigOverride.DeepCopy()
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-generated"] = "2021-08-15T14:30:45+04:00"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-global-updated"] = "2021-08-15T14:30:45+04:00"
						cm.Annotations["cephdeployment.lcm.mirantis.com/config-mon-updated"] = "2021-08-15T14:30:45+04:00"
						return *cm
					}(),
				}},
				"ingresses": &networkingv1.IngressList{},
				"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{
					unitinputs.NetworkPolicyMds, unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd,
				}},
				"secrets": &corev1.SecretList{Items: []corev1.Secret{unitinputs.RookCephMonSecret}},
				"nodes": &corev1.NodeList{
					Items: []corev1.Node{
						unitinputs.GetNodeWithLabels("node-1", map[string]string{"ceph_role_mon": "true", "ceph_role_mgr": "true", "ceph_role_osd": "true", "ceph_role_mds": "true"}, nil),
						unitinputs.GetNodeWithLabels("node-2", map[string]string{"ceph_role_mon": "true", "ceph_role_osd": "true"}, nil),
						unitinputs.GetNodeWithLabels("node-3", map[string]string{"ceph_role_mon": "true", "ceph_role_osd": "true"}, nil),
					},
				},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"daemonsets": &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"pods":       unitinputs.ToolBoxPodList,
				"storageclasses": &storagev1.StorageClassList{
					Items: []storagev1.StorageClass{
						*unitinputs.BaseStorageClassDefault.DeepCopy(), *unitinputs.CephFSStorageClass.DeepCopy(),
					},
				},
				"cephblockpools": &cephv1.CephBlockPoolList{Items: []cephv1.CephBlockPool{
					unitinputs.GetCephBlockPoolWithStatus(unitinputs.CephBlockPoolReplicated, true), *unitinputs.BuiltinMgrPool,
				}},
				"cephclients":    &cephv1.CephClientList{Items: []cephv1.CephClient{*unitinputs.TestCephClientReady.DeepCopy()}},
				"cephrbdmirrors": &cephv1.CephRBDMirrorList{},
				"cephclusters": &cephv1.CephClusterList{
					Items: []cephv1.CephCluster{
						func() cephv1.CephCluster {
							cluster := unitinputs.TestCephCluster.DeepCopy()
							cluster.Spec.Annotations = map[cephv1.KeyType]cephv1.Annotations{
								cephv1.KeyMon: map[string]string{
									"cephdeployment.lcm.mirantis.com/config-global-updated": "2021-08-15T14:30:45+04:00",
									"cephdeployment.lcm.mirantis.com/config-mon-updated":    "2021-08-15T14:30:45+04:00",
								},
								cephv1.KeyMgr: map[string]string{
									"cephdeployment.lcm.mirantis.com/config-global-updated": "2021-08-15T14:30:45+04:00",
								},
							}
							return *cluster
						}(),
					},
				},
				"cephfilesystems": &cephv1.CephFilesystemList{
					Items: []cephv1.CephFilesystem{
						func() cephv1.CephFilesystem {
							fs := unitinputs.GetCephFsWithStatus(cephv1.ConditionReady)
							fs.Spec.MetadataServer.Annotations = map[string]string{
								"cephdeployment.lcm.mirantis.com/config-global-updated": "2021-08-15T14:30:45+04:00",
							}
							return *fs
						}(),
					},
				},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{},
			},
			testclient:      faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			expectedVersion: latestClusterVersion,
			result:          requeueAfterInterval,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseReady,
				Message: "Ceph cluster configuration successfully applied",
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 10,
				},
				ClusterVersion: "v19.2.3",
				LastRun:        "2021-08-15T14:30:33+04:00",
				ObjectsRefs:    unitinputs.CephDeploymentObjectsRefs,
			},
		},
		{
			name: "reconcile cephdeployment - update non-mosk ceph cluster",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{
					*unitinputs.GetUpdatedClusterVersionCephDeploy(unitinputs.CephDeployNonMosk.DeepCopy(), "v18.2.7")}},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.EmptyCephSecret}},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{Items: []cephlcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth}},
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
				"cephosdremovetasks":         &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{}},
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{
					unitinputs.PelagiaConfig, unitinputs.RookCephMonEndpoints, *unitinputs.BaseRookConfigOverride.DeepCopy(),
				}},
				"secrets": &corev1.SecretList{Items: []corev1.Secret{*unitinputs.RgwSSLCertSecretSelfSigned.DeepCopy(), unitinputs.RookCephMonSecret}},
				"nodes":   &corev1.NodeList{Items: []corev1.Node{unitinputs.GetAvailableNode("node-1"), unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"daemonsets": &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"pods":       unitinputs.ToolBoxPodList,
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cluster := unitinputs.TestCephCluster.DeepCopy()
						cluster.Spec.CephVersion.Image = "fake/fake:v.2.3.4"
						return *cluster
					}(),
				}},
			},
			testclient: faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			expectedVersion: &lcmcommon.CephVersion{
				Name:            "Reef",
				MajorVersion:    "v18.2",
				MinorVersion:    "7",
				Order:           18,
				SupportedMinors: []string{"3", "4", "7"},
			},
			result: requeueAfterInterval,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseFailed,
				Message: "failed to ensure consistent Ceph cluster version: update CephCluster rook-ceph/cephcluster version is in progress",
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 10,
				},
				ClusterVersion: "v18.2.7",
				LastRun:        "2021-08-15T14:30:34+04:00",
				ObjectsRefs:    unitinputs.CephDeploymentObjectsRefs,
			},
		},
		{
			name: "reconcile cephdeployment - create mosk ceph cluster is started",
			inputResources: map[string]runtime.Object{
				"cephdeployments":            &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.CephDeployMosk}},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.EmptyCephSecret}},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{Items: []cephlcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth}},
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
				"cephosdremovetasks":         &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{}},
				"configmaps":                 &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig, unitinputs.RookCephMonEndpoints, unitinputs.BaseRookConfigOverride}},
				"secrets":                    &corev1.SecretList{},
				"nodes":                      &corev1.NodeList{Items: []corev1.Node{unitinputs.GetAvailableNode("node-1"), unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"daemonsets":           &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"pods":                 unitinputs.ToolBoxPodList,
				"storageclasses":       &storagev1.StorageClassList{},
				"cephblockpools":       &cephv1.CephBlockPoolList{},
				"cephclients":          &cephv1.CephClientList{},
				"cephclusters":         &cephv1.CephClusterList{},
				"cephfilesystems":      &cephv1.CephFilesystemList{},
				"cephrbdmirrors":       &cephv1.CephRBDMirrorList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{},
				"networkpolicies":      &networkingv1.NetworkPolicyList{},
			},
			testclient:      faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			expectedVersion: latestClusterVersion,
			result:          requeueAfterInterval,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseDeploying,
				Message: "Ceph cluster configuration apply is in progress: label nodes, network policies",
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 0,
				},
				LastRun:     "2021-08-15T14:30:35+04:00",
				ObjectsRefs: unitinputs.CephDeploymentObjectsRefs,
			},
		},
		{
			name: "reconcile cephdeployment - create mosk ceph cluster is in progress",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{
					*unitinputs.GetUpdatedClusterVersionCephDeploy(unitinputs.CephDeployMosk.DeepCopy(), unitinputs.LatestCephVersionImage)}},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.EmptyCephSecret}},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{Items: []cephlcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth}},
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
				"cephosdremovetasks":         &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{}},
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{
					unitinputs.PelagiaConfig, unitinputs.RookCephMonEndpoints, *unitinputs.BaseRookConfigOverride.DeepCopy(),
				}},
				"secrets": &corev1.SecretList{Items: []corev1.Secret{
					unitinputs.RookCephMonSecret, unitinputs.RookCephRgwMetricsSecret, unitinputs.RgwSSLCertSecret,
					*unitinputs.RgwSSLCertSecretSelfSigned.DeepCopy(), *unitinputs.OpenstackSecretGenerated.DeepCopy(),
					func() corev1.Secret {
						secret := unitinputs.IngressRuleSecretCustom.DeepCopy()
						secret.Name = "rook-ceph-rgw-rgw-store-tls-public-1675587456"
						return *secret
					}(),
				}},
				"nodes": &corev1.NodeList{Items: []corev1.Node{unitinputs.GetAvailableNode("node-1"), unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{
					unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw,
				}},
				"daemonsets": &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"pods":       unitinputs.ToolBoxPodList,
				"storageclasses": &storagev1.StorageClassList{Items: []storagev1.StorageClass{
					*unitinputs.GetNamedStorageClass("pool1-hdd", false),
					*unitinputs.GetNamedStorageClass("vms-hdd", false),
					*unitinputs.GetNamedStorageClass("volumes-hdd", false),
					*unitinputs.GetNamedStorageClass("images-hdd", false),
					*unitinputs.GetNamedStorageClass("backup-hdd", false),
				}},
				"cephblockpools": &cephv1.CephBlockPoolList{Items: append([]cephv1.CephBlockPool{unitinputs.GetCephBlockPoolWithStatus(unitinputs.CephBlockPoolReplicated, true)},
					unitinputs.OpenstackCephBlockPoolsListReady.DeepCopy().Items...)},
				"cephclients": &cephv1.CephClientList{Items: []cephv1.CephClient{
					*unitinputs.CephClientGlance.DeepCopy(), *unitinputs.GetCephClientWithStatus(unitinputs.CephClientNova, true), *unitinputs.GetCephClientWithStatus(unitinputs.CephClientCinder, true),
				}},
				"cephclusters":         &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterOpenstack()}},
				"cephfilesystems":      &cephv1.CephFilesystemList{},
				"cephrbdmirrors":       &cephv1.CephRBDMirrorList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase.DeepCopy()}},
				"cephobjectstoreusers": unitinputs.CephObjectStoreUserListMetrics.DeepCopy(),
			},
			testclient:      faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			expectedVersion: latestClusterVersion,
			result:          requeueAfterInterval,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseDeploying,
				Message: "Ceph cluster configuration apply is in progress: label nodes, cephcluster, storageclasses, ceph object storage, Openstack secret, ingress proxy, cluster state; configuration apply is failed: failed to ensure cephclients",
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 0,
				},
				ClusterVersion: "v19.2.3",
				LastRun:        "2021-08-15T14:30:36+04:00",
				ObjectsRefs:    unitinputs.CephDeploymentObjectsRefs,
			},
		},
		{
			name: "reconcile cephdeployment external - no extra info in spec, create success",
			inputResources: map[string]runtime.Object{
				"cephdeployments":            &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{*unitinputs.CephDeployExternal.DeepCopy()}},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{},
				"cephdeploymentmaintenances": &cephlcmv1alpha1.CephDeploymentMaintenanceList{},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{},
				"configmaps":                 &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig}},
				"secrets":                    &corev1.SecretList{Items: []corev1.Secret{unitinputs.ExternalConnectionSecretWithAdmin}},
				"daemonsets":                 &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"deployments":                &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy()}},
				"nodes": &corev1.NodeList{
					Items: []corev1.Node{
						unitinputs.GetNodeWithLabels("node-1", map[string]string{}, nil),
						unitinputs.GetNodeWithLabels("node-2", map[string]string{}, nil),
						unitinputs.GetNodeWithLabels("node-3", map[string]string{}, nil),
					},
				},
				"storageclasses": &storagev1.StorageClassList{
					Items: []storagev1.StorageClass{*unitinputs.BaseStorageClassDefault.DeepCopy()},
				},
				"cephclusters":         &cephv1.CephClusterList{},
				"cephclients":          &cephv1.CephClientList{},
				"cephrbdmirrors":       &cephv1.CephRBDMirrorList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{},
			},
			testclient:      faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			result:          requeueAfterInterval,
			expectedVersion: latestClusterVersion,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseDeploying,
				Message: "Ceph cluster configuration apply is in progress: cephcluster",
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 0,
				},
				LastRun:     "2021-08-15T14:30:37+04:00",
				ObjectsRefs: unitinputs.CephDeploymentObjectsRefs,
			},
		},
		{
			name: "reconcile cephdeployment external - with extra non-external info, create success",
			inputResources: map[string]runtime.Object{
				"cephdeployments":            &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{*unitinputs.CephDeployExternalCephFS.DeepCopy()}},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{},
				"cephdeploymentmaintenances": &cephlcmv1alpha1.CephDeploymentMaintenanceList{},
				"configmaps":                 &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig}},
				"secrets":                    &corev1.SecretList{Items: []corev1.Secret{unitinputs.ExternalConnectionSecretNonAdmin}},
				"daemonsets":                 &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"deployments":                &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy()}},
				"nodes": &corev1.NodeList{
					Items: []corev1.Node{
						unitinputs.GetNodeWithLabels("node-1", map[string]string{}, nil),
						unitinputs.GetNodeWithLabels("node-2", map[string]string{}, nil),
						unitinputs.GetNodeWithLabels("node-3", map[string]string{}, nil),
					},
				},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephclusters":         &cephv1.CephClusterList{},
				"cephclients":          &cephv1.CephClientList{},
				"cephrbdmirrors":       &cephv1.CephRBDMirrorList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{},
			},
			testclient:      faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			result:          requeueAfterInterval,
			expectedVersion: latestClusterVersion,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseDeploying,
				Message: "Ceph cluster configuration apply is in progress: cephcluster, storageclasses",
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 0,
				},
				LastRun:     "2021-08-15T14:30:38+04:00",
				ObjectsRefs: unitinputs.CephDeploymentObjectsRefs,
			},
		},
		{
			name: "reconcile cephdeployment external - updated cephcluster, success",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{
					func() cephlcmv1alpha1.CephDeployment {
						mc := unitinputs.CephDeployExternal.DeepCopy()
						mc.Spec.Pools = []cephlcmv1alpha1.CephPool{
							*unitinputs.CephDeployPoolReplicated.DeepCopy(),
						}
						mc.Status.ClusterVersion = unitinputs.LatestCephVersionImage
						return *mc
					}(),
				}},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{},
				"cephdeploymentmaintenances": &cephlcmv1alpha1.CephDeploymentMaintenanceList{},
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{unitinputs.PelagiaConfig, *unitinputs.RookCephMonEndpointsExternal.DeepCopy()}},
				"secrets":    &corev1.SecretList{Items: []corev1.Secret{*unitinputs.RookCephMonSecretNonAdmin.DeepCopy(), unitinputs.ExternalConnectionSecretWithAdmin}},
				"daemonsets": &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady}},
				"nodes":                &corev1.NodeList{Items: []corev1.Node{unitinputs.GetAvailableNode("node-1"), unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
				"pods":                 unitinputs.ToolBoxPodList,
				"storageclasses":       &storagev1.StorageClassList{Items: []storagev1.StorageClass{*unitinputs.ExternalStorageClassDefault.DeepCopy()}},
				"cephclusters":         &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterExternal.DeepCopy()}},
				"cephclients":          &cephv1.CephClientList{},
				"cephrbdmirrors":       &cephv1.CephRBDMirrorList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{},
			},
			testclient:      faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			result:          requeueAfterInterval,
			expectedVersion: latestClusterVersion,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseDeploying,
				Message: "Ceph cluster configuration apply is in progress: cephcluster",
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 0,
				},
				ClusterVersion: "v19.2.3",
				LastRun:        "2021-08-15T14:30:39+04:00",
				ObjectsRefs:    unitinputs.CephDeploymentObjectsRefs,
			},
		},
		{
			name: "reconcile cephdeployment external - no change, success",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{
					func() cephlcmv1alpha1.CephDeployment {
						mc := unitinputs.CephDeployExternal.DeepCopy()
						mc.Spec.Pools = []cephlcmv1alpha1.CephPool{
							*unitinputs.CephDeployPoolReplicated.DeepCopy(),
						}
						mc.Status.ClusterVersion = unitinputs.LatestCephVersionImage
						return *mc
					}(),
				}},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{},
				"cephdeploymentmaintenances": &cephlcmv1alpha1.CephDeploymentMaintenanceList{},
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{
						unitinputs.PelagiaConfig,
						func() corev1.ConfigMap {
							cm := unitinputs.RookCephMonEndpointsExternal.DeepCopy()
							cm.OwnerReferences = []metav1.OwnerReference{
								{
									APIVersion: "ceph.rook.io/v1",
									Kind:       "CephCluster",
									Name:       "cephcluster",
								},
							}
							return *cm
						}(),
					}},
				"secrets": &corev1.SecretList{Items: []corev1.Secret{
					unitinputs.ExternalConnectionSecretWithAdmin,
					func() corev1.Secret {
						secret := unitinputs.RookCephMonSecret.DeepCopy()
						secret.OwnerReferences = []metav1.OwnerReference{
							{
								APIVersion: "ceph.rook.io/v1",
								Kind:       "CephCluster",
								Name:       "cephcluster",
							},
						}
						return *secret
					}(),
				}},
				"daemonsets": &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady}},
				"nodes":                &corev1.NodeList{Items: []corev1.Node{unitinputs.GetAvailableNode("node-1"), unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
				"pods":                 unitinputs.ToolBoxPodList,
				"storageclasses":       &storagev1.StorageClassList{Items: []storagev1.StorageClass{*unitinputs.ExternalStorageClassDefault.DeepCopy()}},
				"cephclusters":         &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterExternal.DeepCopy()}},
				"cephclients":          &cephv1.CephClientList{},
				"cephrbdmirrors":       &cephv1.CephRBDMirrorList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{},
			},
			testclient:      faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			result:          requeueAfterInterval,
			expectedVersion: latestClusterVersion,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseReady,
				Message: "Ceph cluster configuration successfully applied",
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 0,
				},
				ClusterVersion: "v19.2.3",
				LastRun:        "2021-08-15T14:30:40+04:00",
				ObjectsRefs:    unitinputs.CephDeploymentObjectsRefs,
			},
		},
		{
			name: "reconcile cephdeployment external - maintenance mode skip",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{
					*unitinputs.GetUpdatedClusterVersionCephDeploy(unitinputs.CephDeployExternal.DeepCopy(), unitinputs.LatestCephVersionImage)}},
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListActing,
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{},
				"configmaps":                 &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.PelagiaConfig, *unitinputs.RookCephMonEndpointsExternal.DeepCopy()}},
				"secrets":                    &corev1.SecretList{Items: []corev1.Secret{*unitinputs.RookCephMonSecret.DeepCopy(), unitinputs.ExternalConnectionSecretWithAdmin}},
				"daemonsets":                 &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady}},
				"nodes":        &corev1.NodeList{},
				"pods":         unitinputs.ToolBoxPodList,
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterExternal.DeepCopy()}},
			},
			testclient:      faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			result:          requeueAfterInterval,
			expectedVersion: latestClusterVersion,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseMaintenance,
				Message: "Cluster maintenance (upgrade) detected, reconcile is paused",
				Validation: cephlcmv1alpha1.CephDeploymentValidation{
					Result:                  "Succeed",
					LastValidatedGeneration: 0,
				},
				ClusterVersion: "v19.2.3",
				LastRun:        "2021-08-15T14:30:41+04:00",
				ObjectsRefs:    unitinputs.CephDeploymentObjectsRefs,
			},
		},
		{
			name: "reconcile cephdeployment external - deletion in progress",
			inputResources: map[string]runtime.Object{
				"cephdeployments": &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{
					func() cephlcmv1alpha1.CephDeployment {
						mc := unitinputs.CephDeployExternal.DeepCopy()
						mc.Spec.Pools = []cephlcmv1alpha1.CephPool{
							*unitinputs.CephDeployPoolReplicated.DeepCopy(),
						}
						mc.Spec.SharedFilesystem = &cephlcmv1alpha1.CephSharedFilesystem{
							CephFS: []cephlcmv1alpha1.CephFS{
								unitinputs.CephFSNewOk,
							},
						}
						mc.Spec.Nodes = unitinputs.CephNodesOk
						mc.Spec.IngressConfig = &unitinputs.CephIngressConfig
						mc.DeletionTimestamp = &metav1.Time{Time: time.Now()}
						mc.Status = cephlcmv1alpha1.CephDeploymentStatus{
							Phase: cephlcmv1alpha1.PhaseDeleting,
						}
						return *mc
					}(),
				}},
				"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{},
				"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{},
				"cephdeploymentmaintenances": &cephlcmv1alpha1.CephDeploymentMaintenanceList{},
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{unitinputs.PelagiaConfig, *unitinputs.RookCephMonEndpointsExternal.DeepCopy()}},
				"secrets":    &corev1.SecretList{Items: []corev1.Secret{*unitinputs.RookCephMonSecret.DeepCopy()}},
				"daemonsets": &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady}},
				"nodes":                &corev1.NodeList{Items: []corev1.Node{unitinputs.GetAvailableNode("node-1"), unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
				"pods":                 unitinputs.ToolBoxPodList,
				"storageclasses":       &storagev1.StorageClassList{},
				"cephclusters":         &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterExternal.DeepCopy()}},
				"cephclients":          &cephv1.CephClientList{},
				"cephrbdmirrors":       &cephv1.CephRBDMirrorList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{},
			},
			testclient:      faketestclients.GetClientBuilder().WithStatusSubresource(unitinputs.BaseCephDeployment.DeepCopy()).WithObjects(unitinputs.BaseCephDeployment.DeepCopy()),
			result:          requeueAfterInterval,
			expectedVersion: latestClusterVersion,
			expectedStatus: &cephlcmv1alpha1.CephDeploymentStatus{
				Phase:   cephlcmv1alpha1.PhaseDeleting,
				Message: "Ceph cluster deletion is in progress",
				LastRun: "2021-08-15T14:30:42+04:00",
			},
		},
	}

	oldTriesLeft := failTriesLeft
	oldTimeFunc := lcmcommon.GetCurrentTimeString
	oldUnixTimeFunc := lcmcommon.GetCurrentUnixTimeString
	oldGenerateCrtFunc := lcmcommon.GenerateSelfSignedCert
	oldCephCmdFunc := lcmcommon.RunPodCommandWithValidation
	failTriesLeft = 0
	t.Setenv("CEPH_CONTROLLER_CLUSTER_RELEASE", "1.1.1")
	t.Setenv("WAIT_FOR_OPENSTACK_LOCK", "false")
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r.Client = faketestclients.GetClient(test.testclient)

			if test.inputResources != nil && test.inputResources["configmaps"] != nil {
				for _, cm := range test.inputResources["configmaps"].(*corev1.ConfigMapList).Items {
					if cm.Name == "pelagia-lcmconfig" {
						configReconciler := &lcmconfig.ReconcileCephDeploymentHealthConfig{
							Client: faketestclients.GetClientBuilderWithObjects(&cm).Build(),
							Scheme: fscheme.Scheme,
						}
						configRequest := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: cm.Namespace, Name: cm.Name}}
						_, err := configReconciler.Reconcile(context.TODO(), configRequest)
						assert.Nil(t, err)
						break
					}
				}
			}

			lcmcommon.GetCurrentTimeString = func() string {
				return fmt.Sprintf("2021-08-15T14:30:%d+04:00", 10+idx)
			}
			lcmcommon.GetCurrentUnixTimeString = func() string {
				return "1675587456"
			}
			lcmcommon.GenerateSelfSignedCert = func(_, _ string, _ []string) (string, string, string, error) {
				return "fake-key", "fake-crt", "fake-ca", nil
			}
			// cephdpl actions
			faketestclients.FakeReaction(r.CephLcmclientset, "list", []string{"cephosdremovetasks", "cephdeployments"}, test.inputResources, nil)
			faketestclients.FakeReaction(r.CephLcmclientset, "get", []string{"cephdeploymenthealths", "cephdeploymentsecrets", "cephdeploymentmaintenances"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.CephLcmclientset, "create", []string{"cephdeploymenthealths", "cephdeploymentsecrets", "cephdeploymentmaintenances"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.CephLcmclientset, "delete", []string{"cephdeploymenthealths", "cephdeploymentsecrets", "cephdeploymentmaintenances"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.CephLcmclientset, "update", []string{"cephdeployments"}, test.inputResources, test.apiErrors)

			// kube actions
			faketestclients.FakeReaction(r.Kubeclientset.CoreV1(), "get", []string{"configmaps", "pods", "nodes", "secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Kubeclientset.CoreV1(), "list", []string{"deployments", "nodes", "pods"}, test.inputResources, nil)
			faketestclients.FakeReaction(r.Kubeclientset.CoreV1(), "update", []string{"nodes"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Kubeclientset.CoreV1(), "delete", []string{"configmaps", "secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Kubeclientset.AppsV1(), "get", []string{"daemonsets", "deployments"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Kubeclientset.AppsV1(), "delete", []string{"daemonsets", "deployments"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Kubeclientset.StorageV1(), "list", []string{"storageclasses"}, test.inputResources, nil)
			faketestclients.FakeReaction(r.Kubeclientset.StorageV1(), "delete", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Kubeclientset.NetworkingV1(), "get", []string{"networkpolicies"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Kubeclientset.NetworkingV1(), "delete", []string{"ingresses", "networkpolicies"}, test.inputResources, test.apiErrors)

			// rook actions
			faketestclients.FakeReaction(r.Rookclientset, "list", cephAPIResources, test.inputResources, nil)
			faketestclients.FakeReaction(r.Rookclientset, "get", cephAPIResources, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Rookclientset, "create", cephAPIResources, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Rookclientset, "delete", cephAPIResources, test.inputResources, test.apiErrors)

			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if strings.Contains(e.Command, "radosgw-admin") {
					return unitinputs.CephZoneGroupInfoHostnamesFromOpenstack, "", nil
				} else if strings.Contains(e.Command, "/usr/local/bin/zonegroup_hostnames_update.sh") {
					return "", "", nil
				} else if strings.Contains(e.Command, "ceph auth get-key client.cinder") {
					return "cinder", "", nil
				} else if strings.Contains(e.Command, "ceph auth get-key client.nova") {
					return "nova", "", nil
				} else if strings.Contains(e.Command, "ceph auth get-key client.glance") {
					return "glance", "", nil
				} else if strings.Contains(e.Command, "ceph auth get-key") {
					return "fake", "", nil
				} else if strings.Contains(e.Command, "ceph osd pool ls -f json") {
					return unitinputs.CephOsdLspools, "", nil
				} else if e.Command == "ceph config get mgr mgr/progress/allow_pg_recovery_event" {
					return "false", "", nil
				} else if e.Command == "ceph versions --format json" {
					return unitinputs.CephVersionsLatest, "", nil
				} else if strings.Contains(e.Command, "config dump") {
					return unitinputs.CephConfigDumpDefaults, "", nil
				} else if strings.Contains(e.Command, "ceph fs subvolumegroup -f json ls test-cephfs") {
					return `[{"name":"csi"}]`, "", nil
				} else if strings.HasPrefix(e.Command, "ceph mgr module ls") {
					if test.apiErrors["cluster-state"] != nil {
						return unitinputs.MgrModuleLsNoPrometheus, "", nil
					}
					return unitinputs.MgrModuleLsWithPrometheus, "", nil
				}
				return "", "", errors.New("unexpected run pod command call")
			}

			res, err := r.Reconcile(context.TODO(), request)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.result, res)
			cephDpl := &cephlcmv1alpha1.CephDeployment{}
			err = r.Client.Get(context.Background(), client.ObjectKey{Name: request.Name, Namespace: request.Namespace}, cephDpl)
			if test.expectedStatus == nil {
				assert.NotNil(t, err)
				assert.Equal(t, "cephdeployments.lcm.mirantis.com \"cephcluster\" not found", err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, *test.expectedStatus, cephDpl.Status)
			}
			// clean reactions
			faketestclients.CleanupFakeClientReactions(r.CephLcmclientset)
			faketestclients.CleanupFakeClientReactions(r.Rookclientset)
			faketestclients.CleanupFakeClientReactions(r.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(r.Kubeclientset.AppsV1())
			faketestclients.CleanupFakeClientReactions(r.Kubeclientset.StorageV1())
			faketestclients.CleanupFakeClientReactions(r.Kubeclientset.NetworkingV1())
		})
	}
	failTriesLeft = oldTriesLeft
	lcmcommon.GetCurrentTimeString = oldTimeFunc
	lcmcommon.GetCurrentUnixTimeString = oldUnixTimeFunc
	lcmcommon.GenerateSelfSignedCert = oldGenerateCrtFunc
	lcmcommon.RunPodCommandWithValidation = oldCephCmdFunc
	unsetTimestampsVar()
}

func TestCleanCephDeployment(t *testing.T) {
	// prepare full spec for object remove, except multisite
	cephDplFull := unitinputs.CephDeployMosk.DeepCopy()
	cephDplFull.Spec.SharedFilesystem = unitinputs.CephSharedFileSystemOk.DeepCopy()
	cephDplFull.Spec.Clients = []cephlcmv1alpha1.CephClient{*unitinputs.CephDeployClientTest.DeepCopy()}

	inputResourcesBase := map[string]runtime.Object{
		"cephclusters":         &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.TestCephCluster.DeepCopy()}},
		"cephblockpools":       unitinputs.CephBlockPoolListBaseReady.DeepCopy(),
		"cephclients":          &cephv1.CephClientList{Items: []cephv1.CephClient{*unitinputs.TestCephClient.DeepCopy()}},
		"cephfilesystems":      &cephv1.CephFilesystemList{Items: []cephv1.CephFilesystem{*unitinputs.TestCephFs.DeepCopy()}},
		"cephobjectstores":     &cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase.DeepCopy()}},
		"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{},
		"cephrbdmirrors":       &cephv1.CephRBDMirrorList{Items: []cephv1.CephRBDMirror{*unitinputs.CephRBDMirror.DeepCopy()}},
		"deployments":          &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady}},
		"ingresses":            &networkingv1.IngressList{Items: []networkingv1.Ingress{*unitinputs.RgwIngress.DeepCopy()}},
		"networkpolicies": &networkingv1.NetworkPolicyList{Items: []networkingv1.NetworkPolicy{
			unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw,
		}},
		"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{Items: []cephlcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth}},
		"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.EmptyCephSecret}},
		"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle.DeepCopy(),
		"nodes":                      &corev1.NodeList{Items: []corev1.Node{*unitinputs.RolesTopologyLabelsNode.DeepCopy(), unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
		"secrets":                    &corev1.SecretList{Items: []corev1.Secret{unitinputs.OpenstackSecretGenerated, unitinputs.RgwSSLCertSecret, unitinputs.IngressRuleSecret, unitinputs.CephRBDMirrorSecret1, unitinputs.CephRBDMirrorSecret2}},
		"storageclasses":             &storagev1.StorageClassList{Items: []storagev1.StorageClass{*unitinputs.GetNamedStorageClass("vms-hdd", false)}},
		"daemonsets":                 &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
	}

	cephDplExternal := unitinputs.CephDeployExternal.DeepCopy()
	cephDplExternal.Spec.Pools = []cephlcmv1alpha1.CephPool{*unitinputs.CephDeployPoolReplicated.DeepCopy()}
	inputResourcesExternal := map[string]runtime.Object{
		"cephclusters":               &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterExternal.DeepCopy()}},
		"cephclients":                &cephv1.CephClientList{Items: []cephv1.CephClient{*unitinputs.CephClientGlance.DeepCopy()}},
		"cephobjectstores":           &cephv1.CephObjectStoreList{Items: []cephv1.CephObjectStore{*unitinputs.CephObjectStoreBase.DeepCopy()}},
		"cephobjectstoreusers":       &cephv1.CephObjectStoreUserList{},
		"cephrbdmirrors":             &cephv1.CephRBDMirrorList{},
		"deployments":                &appsv1.DeploymentList{Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady}},
		"cephdeploymenthealths":      &cephlcmv1alpha1.CephDeploymentHealthList{Items: []cephlcmv1alpha1.CephDeploymentHealth{unitinputs.CephDeploymentHealth}},
		"cephdeploymentsecrets":      &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.EmptyCephSecret}},
		"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle.DeepCopy(),
		"nodes":                      &corev1.NodeList{Items: []corev1.Node{unitinputs.GetAvailableNode("node-2"), unitinputs.GetAvailableNode("node-3")}},
		"secrets":                    &corev1.SecretList{Items: []corev1.Secret{*unitinputs.ExternalConnectionSecretWithAdmin.DeepCopy()}},
		"storageclasses":             &storagev1.StorageClassList{Items: []storagev1.StorageClass{*unitinputs.ExternalStorageClassDefault.DeepCopy()}},
	}

	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		inputResources map[string]runtime.Object
		apiErrors      map[string]error
		expectedError  string
		cleanupDone    bool
	}{
		{
			name:           "failed to delete resources, first step",
			cephDpl:        cephDplFull.DeepCopy(),
			inputResources: inputResourcesBase,
			apiErrors: map[string]error{
				"delete-cephblockpools":             errors.New("failed to delete cephblockpool"),
				"delete-cephclients":                errors.New("failed to delete cephclient"),
				"delete-cephfilesystems":            errors.New("failed to delete cephfilesystem"),
				"delete-cephobjectstores":           errors.New("failed to delete cephobjectstore"),
				"delete-cephrbdmirrors":             errors.New("failed to delete cephrbdmirror"),
				"delete-deployments":                errors.New("failed to delete deployment"),
				"delete-ingresses":                  errors.New("failed to delete ingress"),
				"delete-cephdeploymentsecrets":      errors.New("failed to delete cephdeploymentsecret"),
				"delete-cephdeploymentmaintenances": errors.New("failed to delete cephdeploymentmaintenance"),
				"delete-secrets":                    errors.New("failed to delete secret"),
				"delete-collection-secrets":         errors.New("failed to delete secret"),
				"delete-storageclasses":             errors.New("failed to delete storageclass"),
				"delete-daemonsets":                 errors.New("failed to delete daemonset"),
			},
			expectedError: "deletion is not completed for CephDeployment: failed to remove CephDeploymentSecret 'lcm-namespace/cephcluster', failed to remove CephDeploymentMaintenance 'lcm-namespace/cephcluster', failed to remove openstack shared secret, failed to remove object storage, failed to remove ingress proxy, failed to remove rbd mirror, failed to remove ceph clients, failed to remove ceph block pools, failed to remove ceph shared filesystem, failed to remove storage classes",
		},
		{
			name:           "delete resources is in progress, first step",
			cephDpl:        cephDplFull.DeepCopy(),
			inputResources: inputResourcesBase,
		},
		{
			name:           "delete resources is in progress (remove everything depended for first step)",
			cephDpl:        cephDplFull.DeepCopy(),
			inputResources: inputResourcesBase,
		},
		{
			name:           "failed to delete resources, (cluster remove and nodes)",
			cephDpl:        cephDplFull.DeepCopy(),
			inputResources: inputResourcesBase,
			apiErrors: map[string]error{
				"delete-cephclusters":          errors.New("failed to delete cephcluster"),
				"delete-cephdeploymenthealths": errors.New("failed to delete cephdeploymenthealth"),
				"delete-daemonsets":            errors.New("failed to delete daemonset"),
				"delete-networkpolicies":       errors.New("failed to delete networkpolicy"),
				"update-nodes":                 errors.New("failed to update node"),
			},
			expectedError: "deletion is not completed for CephDeployment: failed to remove CephDeploymentHealth 'lcm-namespace/cephcluster', failed to remove ceph cluster, failed to remove network policies, failed to remove node ceph labels, failed to remove daemonset ceph labels",
		},
		{
			name:           "delete resources is in progress (cluster remove and nodes)",
			cephDpl:        cephDplFull.DeepCopy(),
			inputResources: inputResourcesBase,
		},
		{
			name:           "resources are deleted",
			cephDpl:        cephDplFull.DeepCopy(),
			inputResources: inputResourcesBase,
			cleanupDone:    true,
		},
		{
			name:           "external - failed to delete resources (first step)",
			cephDpl:        cephDplExternal.DeepCopy(),
			inputResources: inputResourcesExternal,
			apiErrors: map[string]error{
				"delete-cephclients":                errors.New("failed to delete cephclient"),
				"delete-cephobjectstores":           errors.New("failed to delete cephobjectstore"),
				"delete-cephrbdmirrors":             errors.New("failed to delete cephrbdmirror"),
				"delete-deployments":                errors.New("failed to delete deployment"),
				"delete-cephdeploymentsecrets":      errors.New("failed to delete cephdeploymentsecret"),
				"delete-cephdeploymentmaintenances": errors.New("failed to delete cephdeploymentmaintenance"),
				"delete-secrets":                    errors.New("failed to delete secret"),
				"delete-storageclasses":             errors.New("failed to delete storageclass"),
			},
			expectedError: "deletion is not completed for CephDeployment: failed to remove CephDeploymentSecret 'lcm-namespace/cephcluster', failed to remove CephDeploymentMaintenance 'lcm-namespace/cephcluster', failed to remove openstack shared secret, failed to remove object storage, failed to remove rbd mirror, failed to remove ceph clients, failed to remove storage classes, failed to remove external resources",
		},
		{
			name:           "external - delete resources is in progress (first step)",
			cephDpl:        cephDplExternal.DeepCopy(),
			inputResources: inputResourcesExternal,
		},
		{
			name:           "external - failed to delete resources, (cluster)",
			cephDpl:        cephDplExternal.DeepCopy(),
			inputResources: inputResourcesExternal,
			apiErrors: map[string]error{
				"delete-cephclusters":          errors.New("failed to delete cephcluster"),
				"delete-cephdeploymenthealths": errors.New("failed to delete cephdeploymenthealth"),
				"update-nodes":                 errors.New("failed to update node"),
			},
			expectedError: "deletion is not completed for CephDeployment: failed to remove CephDeploymentHealth 'lcm-namespace/cephcluster', failed to remove ceph cluster, failed to remove daemonset ceph labels",
		},
		{
			name:           "external - resources are deleting (cluster)",
			cephDpl:        cephDplExternal.DeepCopy(),
			inputResources: inputResourcesExternal,
		},
		{
			name:           "external - resources are deleted",
			cephDpl:        cephDplExternal.DeepCopy(),
			inputResources: inputResourcesExternal,
			cleanupDone:    true,
		},
	}

	oldRunCmdFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, map[string]string{"DEPLOYMENT_NETPOL_ENABLED": "true"})
			c.cdConfig.currentCephVersion = lcmcommon.Reef

			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if strings.HasPrefix(e.Command, "ceph fs subvolumegroup -f json") {
					return "[]", "", nil
				}
				return "", "", errors.New("unexpected command")
			}

			// deployment actions
			faketestclients.FakeReaction(c.api.CephLcmclientset, "delete", []string{"cephdeploymenthealths", "cephdeploymentsecrets", "cephdeploymentmaintenances"}, test.inputResources, test.apiErrors)
			// kube actions
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"nodes", "secrets"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "list", []string{"ingresses", "networkpolicies"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"nodes"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"nodes"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"configmaps", "secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete-collection", []string{"secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "delete", []string{"ingresses", "networkpolicies"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "delete", []string{"daemonsets", "deployments"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "list", []string{"storageclasses"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "delete", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			// rook actions
			faketestclients.FakeReaction(c.api.Rookclientset, "list", cephAPIResources, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephclusters"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "delete", cephAPIResources, test.inputResources, test.apiErrors)

			deleted, err := c.cleanCephDeployment()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.cleanupDone, deleted)
			if test.cleanupDone {
				assert.Equal(t, updateTimestamps{cephConfigMap: map[string]string{}}, resourceUpdateTimestamps)
			}
			// clean reactions
			faketestclients.CleanupFakeClientReactions(c.api.CephLcmclientset)
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.AppsV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.StorageV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.NetworkingV1())
		})
	}
	lcmcommon.RunPodCommandWithValidation = oldRunCmdFunc
}

func TestVerifySetup(t *testing.T) {
	tests := []struct {
		name            string
		cephDpl         *cephlcmv1alpha1.CephDeployment
		rookImage       string
		cephImage       string
		cephVersion     *lcmcommon.CephVersion
		rookOverrideSet bool
		inputResources  map[string]runtime.Object
		expectedError   string
	}{
		{
			name:      "version is updated and images are not consistent",
			cephDpl:   unitinputs.CephDeployEnsureRbdMirror.DeepCopy(),
			rookImage: unitinputs.PelagiaConfigForPrevCephVersion.Data["DEPLOYMENT_ROOK_IMAGE"],
			cephImage: unitinputs.PelagiaConfigForPrevCephVersion.Data["DEPLOYMENT_CEPH_IMAGE"],
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{
						*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"pods": unitinputs.ToolBoxPodList,
			},
			cephVersion:   lcmcommon.Squid,
			expectedError: "failed to ensure consistent Rook image version: deployment rook-ceph/rook-ceph-operator rook image update is in progress",
		},
		{
			name:      "global ceph version is not set",
			cephDpl:   unitinputs.CephDeployEnsureRbdMirror.DeepCopy(),
			rookImage: unitinputs.PelagiaConfigForPrevCephVersion.Data["DEPLOYMENT_ROOK_IMAGE"],
			cephImage: unitinputs.PelagiaConfigForPrevCephVersion.Data["DEPLOYMENT_CEPH_IMAGE"],
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{
						*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"pods": unitinputs.ToolBoxPodList,
			},
			expectedError: "current Ceph version is not detected",
		},

		{
			name:      "failed on rook images consistency - can't get daemonset",
			cephDpl:   unitinputs.CephDeployEnsureRbdMirror.DeepCopy(),
			rookImage: unitinputs.PelagiaConfig.Data["DEPLOYMENT_ROOK_IMAGE"],
			cephImage: unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"],
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{
						*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"daemonsets": &appsv1.DaemonSetList{},
				"pods":       unitinputs.ToolBoxPodList,
			},
			cephVersion:   lcmcommon.Squid,
			expectedError: "failed to ensure consistent Rook image version: failed to get rook-ceph/rook-discover daemonset: daemonsets \"rook-discover\" not found",
		},
		{
			name:      "ceph image is updated - wait image is updated",
			cephDpl:   unitinputs.CephDeployEnsureRbdMirror.DeepCopy(),
			rookImage: unitinputs.PelagiaConfig.Data["DEPLOYMENT_ROOK_IMAGE"],
			cephImage: unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"],
			inputResources: map[string]runtime.Object{
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"daemonsets": &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"pods":       unitinputs.ToolBoxPodList,
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.BaseRookConfigOverride}},
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{
					func() cephv1.CephCluster {
						cluster := unitinputs.TestCephCluster.DeepCopy()
						cluster.Spec.CephVersion.Image = "fake/fake:v.2.3.4"
						return *cluster
					}(),
				}},
			},
			cephVersion:     lcmcommon.Squid,
			rookOverrideSet: true,
			expectedError:   "failed to ensure consistent Ceph cluster version: update CephCluster rook-ceph/cephcluster version is in progress",
		},
		{
			name:      "verify succeed",
			cephDpl:   unitinputs.CephDeployEnsureRbdMirror.DeepCopy(),
			rookImage: unitinputs.PelagiaConfig.Data["DEPLOYMENT_ROOK_IMAGE"],
			cephImage: unitinputs.PelagiaConfig.Data["DEPLOYMENT_CEPH_IMAGE"],
			inputResources: map[string]runtime.Object{
				"cephclusters": &cephv1.CephClusterList{},
				"deployments": &appsv1.DeploymentList{
					Items: []appsv1.Deployment{
						*unitinputs.RookDeploymentLatestVersion.DeepCopy(), *unitinputs.ToolBoxDeploymentReady},
				},
				"configmaps": &corev1.ConfigMapList{Items: []corev1.ConfigMap{unitinputs.BaseRookConfigOverride}},
				"daemonsets": &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*unitinputs.RookDiscover.DeepCopy()}},
				"pods":       unitinputs.ToolBoxPodList,
			},
			cephVersion:     lcmcommon.Squid,
			rookOverrideSet: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, nil)
			if test.cephVersion != nil {
				c.cdConfig.currentCephVersion = test.cephVersion
			} else {
				c.cdConfig.currentCephVersion = nil
			}
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephclusters"}, test.inputResources, nil)
			if test.rookOverrideSet {
				faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"pods", "configmaps"}, test.inputResources, nil)
			} else {
				faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"pods"}, test.inputResources, nil)
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"pods"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "get", []string{"daemonsets", "deployments"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.AppsV1(), "update", []string{"daemonsets", "deployments"}, test.inputResources, nil)

			c.cdConfig.currentCephImage = test.cephImage
			c.lcmConfig.DeployParams.RookImage = test.rookImage
			err := c.verifySetup()
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			// clean reactions before next test
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
		})
	}
}

func TestCheckLCMStuff(t *testing.T) {
	testCephDpl := unitinputs.CephDeployNonMosk.DeepCopy()
	testCephDpl.Status.Phase = cephlcmv1alpha1.PhaseDeploying

	tests := []struct {
		name              string
		cephDpl           *cephlcmv1alpha1.CephDeployment
		ccsettingsMap     map[string]string
		inputResources    map[string]runtime.Object
		expectedError     string
		expectedLcmActive bool
		expectedPhase     cephlcmv1alpha1.CephDeploymentPhase
		apiErrors         map[string]error
	}{
		{
			name:          "list cephosdremovetasks failed",
			cephDpl:       testCephDpl,
			ccsettingsMap: unitinputs.PelagiaConfig.Data,
			expectedPhase: cephlcmv1alpha1.PhaseFailed,
			expectedError: "failed to list CephOsdRemoveTasks in lcm-namespace namespace: failed to list cephosdremovetasks",
		},
		{
			name:    "task on validation - no hold, failed to check cephcluster spec",
			cephDpl: testCephDpl,
			inputResources: map[string]runtime.Object{
				"cephosdremovetasks": &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{
					*unitinputs.CephOsdRemoveTaskOnValidation,
				}},
				"cephclusters": &cephv1.CephClusterList{},
			},
			ccsettingsMap: unitinputs.PelagiaConfig.Data,
			expectedPhase: cephlcmv1alpha1.PhaseFailed,
			expectedError: "failed to check Ceph cluster state: cephclusters \"cephcluster\" not found",
		},
		{
			name:    "task on validation - no hold, ceph cluster is not updated yet and workloadlock failed to check",
			cephDpl: testCephDpl,
			inputResources: map[string]runtime.Object{
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
				"cephosdremovetasks": &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{
					*unitinputs.CephOsdRemoveTaskOnValidation,
				}},
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{*unitinputs.CephClusterGenerated.DeepCopy()}},
			},
			apiErrors: map[string]error{
				"get-cephdeploymentmaintenances": errors.New("failed to get CephDeploymentMaintenance"),
			},
			ccsettingsMap: unitinputs.PelagiaConfig.Data,
			expectedPhase: cephlcmv1alpha1.PhaseFailed,
			expectedError: "failed to check CephDeploymentMaintenance state: failed to get CephDeploymentMaintenance lcm-namespace/cephcluster: failed to get CephDeploymentMaintenance",
		},
		{
			name: "task on validation - no hold, storage changes in spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployNonMosk.DeepCopy()
				nodes := mc.Spec.Nodes
				nodes[2].Devices = nil
				mc.Spec.Nodes = nodes
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephosdremovetasks":         &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{*unitinputs.CephOsdRemoveTaskOnValidation}},
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
				"cephclusters":               &cephv1.CephClusterList{Items: []cephv1.CephCluster{unitinputs.TestCephCluster}},
			},
			ccsettingsMap: unitinputs.PelagiaConfig.Data,
			expectedPhase: cephlcmv1alpha1.PhaseReady,
		},
		{
			name: "task on validation - hold, only mon role changes in spec",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployNonMosk.DeepCopy()
				nodes := mc.Spec.Nodes
				newNode := nodes[2].DeepCopy()
				newNode.Name = "node-4"
				newNode.Devices = nil
				newNode.Config = nil
				mc.Spec.Nodes = append(mc.Spec.Nodes, *newNode)
				return mc
			}(),
			inputResources: map[string]runtime.Object{
				"cephosdremovetasks": &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{
					*unitinputs.CephOsdRemoveTaskOnValidation,
				}},
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{unitinputs.TestCephCluster}},
			},
			ccsettingsMap:     unitinputs.PelagiaConfig.Data,
			expectedPhase:     cephlcmv1alpha1.PhaseOnHold,
			expectedLcmActive: true,
		},
		{
			name:    "task waiting approve - hold reconcile",
			cephDpl: testCephDpl,
			inputResources: map[string]runtime.Object{
				"cephosdremovetasks": &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{
					*unitinputs.CephOsdRemoveTaskOnApproveWaiting,
				}},
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{unitinputs.TestCephCluster}},
			},
			ccsettingsMap:     unitinputs.PelagiaConfig.Data,
			expectedPhase:     cephlcmv1alpha1.PhaseOnHold,
			expectedLcmActive: true,
		},
		{
			name:    "task waiting ceph operator - hold reconcile",
			cephDpl: testCephDpl,
			inputResources: map[string]runtime.Object{
				"cephosdremovetasks": &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{
					*unitinputs.CephOsdRemoveTaskOnApproved,
				}},
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{unitinputs.TestCephCluster}},
			},
			ccsettingsMap:     unitinputs.PelagiaConfig.Data,
			expectedPhase:     cephlcmv1alpha1.PhaseOnHold,
			expectedLcmActive: true,
		},
		{
			name:    "task processing - hold reconcile",
			cephDpl: testCephDpl,
			inputResources: map[string]runtime.Object{
				"cephosdremovetasks": &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{
					*unitinputs.CephOsdRemoveTaskProcessing,
				}},
				"cephclusters": &cephv1.CephClusterList{Items: []cephv1.CephCluster{unitinputs.TestCephCluster}},
			},
			ccsettingsMap:     unitinputs.PelagiaConfig.Data,
			expectedPhase:     cephlcmv1alpha1.PhaseOnHold,
			expectedLcmActive: true,
		},
		{
			name:    "task failed - required user action",
			cephDpl: testCephDpl,
			inputResources: map[string]runtime.Object{
				"cephosdremovetasks": &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{
					*unitinputs.CephOsdRemoveTaskFailed,
				}},
			},
			ccsettingsMap:     unitinputs.PelagiaConfig.Data,
			expectedPhase:     cephlcmv1alpha1.PhaseOnHold,
			expectedLcmActive: true,
		},
		{
			name:    "task failed - resolved and no required user action, no maintenance",
			cephDpl: testCephDpl,
			inputResources: map[string]runtime.Object{
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
				"cephosdremovetasks": &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{
					func() cephlcmv1alpha1.CephOsdRemoveTask {
						task := unitinputs.CephOsdRemoveTaskFailed.DeepCopy()
						task.Spec.Resolved = true
						return *task
					}(),
				}},
			},
			ccsettingsMap: unitinputs.PelagiaConfig.Data,
			expectedPhase: cephlcmv1alpha1.PhaseReady,
		},
		{
			name:    "cephdeploymentmaintenance is acting, maintenance in action",
			cephDpl: testCephDpl,
			inputResources: map[string]runtime.Object{
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListActing,
				"cephosdremovetasks":         &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{}},
			},
			ccsettingsMap:     unitinputs.PelagiaConfig.Data,
			expectedPhase:     cephlcmv1alpha1.PhaseMaintenance,
			expectedLcmActive: true,
		},
		{
			name:    "reconcile cephdeployment external - skip tasks handling",
			cephDpl: &unitinputs.CephDeployExternal,
			inputResources: map[string]runtime.Object{
				"cephdeploymentmaintenances": unitinputs.CephDeploymentMaintenanceListIdle,
				"cephosdremovetasks":         &cephlcmv1alpha1.CephOsdRemoveTaskList{Items: []cephlcmv1alpha1.CephOsdRemoveTask{}},
			},
			ccsettingsMap: unitinputs.PelagiaConfig.Data,
			expectedPhase: cephlcmv1alpha1.PhaseReady,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, test.ccsettingsMap)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", []string{"cephclusters"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.CephLcmclientset, "list", []string{"cephosdremovetasks"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.CephLcmclientset, "get", []string{"cephdeploymentmaintenances"}, test.inputResources, test.apiErrors)

			nodesList, err := lcmcommon.GetExpandedCephDeploymentNodeList(c.context, c.api.Client, test.cephDpl.Spec)
			assert.Nil(t, err)
			c.cdConfig.nodesListExpanded = nodesList

			ready, phase, err := c.checkLcmState()
			assert.Equal(t, test.expectedLcmActive, ready)
			assert.Equal(t, test.expectedPhase, phase)
			if test.expectedError == "" {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			}
			// clean reactions before next test
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
			faketestclients.CleanupFakeClientReactions(c.api.CephLcmclientset)
		})
	}
}

func TestApplyConfiguration(t *testing.T) {
	ccsettingsMap := unitinputs.PelagiaConfig.Data

	fullCephDplSpec := unitinputs.CephDeployMosk.DeepCopy()
	fullCephDplSpec.Spec.RBDMirror = unitinputs.CephDeployEnsureRbdMirror.Spec.RBDMirror.DeepCopy()
	fullCephDplSpec.Spec.SharedFilesystem = unitinputs.CephDeployNonMosk.Spec.SharedFilesystem.DeepCopy()
	nodesForApply := corev1.NodeList{
		Items: []corev1.Node{
			unitinputs.GetAvailableNode("node-1"),
			unitinputs.GetAvailableNode("node-2"),
			unitinputs.GetAvailableNode("node-3"),
		},
	}
	externalCephDplButWithCommonFields := fullCephDplSpec.DeepCopy()
	externalCephDplButWithCommonFields.ObjectMeta = unitinputs.CephDeployExternal.ObjectMeta
	externalCephDplButWithCommonFields.Spec.External = true
	externalCephDplButWithCommonFields.Spec.ObjectStorage = unitinputs.CephDeployExternalRgw.Spec.ObjectStorage.DeepCopy()

	inputResourcesForApply := map[string]runtime.Object{
		"nodes": &nodesForApply,
		"ingresses": &networkingv1.IngressList{
			Items: []networkingv1.Ingress{
				*unitinputs.RgwIngress.DeepCopy(),
			},
		},
		"networkpolicies": &networkingv1.NetworkPolicyList{
			Items: []networkingv1.NetworkPolicy{
				unitinputs.NetworkPolicyMds, unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw,
			},
		},
		"configmaps": &corev1.ConfigMapList{
			Items: []corev1.ConfigMap{
				*unitinputs.RookCephMonEndpoints.DeepCopy(),
				*unitinputs.BaseRookConfigOverride.DeepCopy(),
			},
		},
		"secrets": &corev1.SecretList{
			Items: []corev1.Secret{
				*unitinputs.RgwSSLCertSecret.DeepCopy(),
				*unitinputs.IngressRuleSecret.DeepCopy(),
				*unitinputs.CephRBDMirrorSecret1.DeepCopy(),
				*unitinputs.CephRBDMirrorSecret2.DeepCopy(),
				*unitinputs.OpenstackRgwCredsSecretNoBarbican.DeepCopy(),
				*unitinputs.RookCephMonSecret.DeepCopy(),
				*unitinputs.RookCephRgwMetricsSecret.DeepCopy(),
			},
		},
		"services":       &corev1.ServiceList{},
		"pods":           unitinputs.ToolBoxPodList,
		"storageclasses": &storagev1.StorageClassList{},
		"cephblockpools": &cephv1.CephBlockPoolList{
			Items: append(unitinputs.OpenstackCephBlockPoolsListReady.DeepCopy().Items,
				unitinputs.GetCephBlockPoolWithStatus(unitinputs.CephBlockPoolReplicated, true), *unitinputs.BuiltinMgrPool.DeepCopy(), *unitinputs.BuiltinRgwRootPool.DeepCopy()),
		},
		"cephclients": &cephv1.CephClientList{
			Items: []cephv1.CephClient{
				*unitinputs.GetCephClientWithStatus(unitinputs.CephClientCinder, true),
				*unitinputs.GetCephClientWithStatus(unitinputs.CephClientNova, true),
				*unitinputs.GetCephClientWithStatus(unitinputs.CephClientGlance, true),
				*unitinputs.GetCephClientWithStatus(unitinputs.CephClientManila, true),
			},
		},
		"cephclusters": &cephv1.CephClusterList{
			Items: []cephv1.CephCluster{*unitinputs.CephClusterOpenstack()},
		},
		"cephfilesystems": unitinputs.CephFSListReady.DeepCopy(),
		"cephrbdmirrors": &cephv1.CephRBDMirrorList{
			Items: []cephv1.CephRBDMirror{
				*unitinputs.CephRBDMirrorWithStatus(unitinputs.CephRBDMirror, "Ready"),
			},
		},
		"cephobjectstores": &cephv1.CephObjectStoreList{
			Items: []cephv1.CephObjectStore{
				*unitinputs.CephObjectStoreBase.DeepCopy(),
			},
		},
		"cephobjectstoreusers": unitinputs.CephObjectStoreUserListMetrics.DeepCopy(),
	}

	inputResourcesForExternalApply := map[string]runtime.Object{
		"configmaps": &corev1.ConfigMapList{},
		"secrets": &corev1.SecretList{
			Items: []corev1.Secret{
				*unitinputs.ExternalConnectionSecretWithAdminAndRgw.DeepCopy(),
				*unitinputs.RgwSSLCertSecret.DeepCopy(),
				*unitinputs.CephRBDMirrorSecret1.DeepCopy(),
				*unitinputs.CephRBDMirrorSecret2.DeepCopy(),
			},
		},
		"nodes": &corev1.NodeList{
			Items: []corev1.Node{
				unitinputs.GetAvailableNode("node-1"),
				unitinputs.GetAvailableNode("node-2"),
				unitinputs.GetAvailableNode("node-3"),
			},
		},
		"storageclasses": &storagev1.StorageClassList{},
		"cephclients":    &cephv1.CephClientList{},
		"cephclusters": &cephv1.CephClusterList{
			Items: []cephv1.CephCluster{
				*unitinputs.CephClusterExternal.DeepCopy(),
			},
		},
		"cephrbdmirrors": &cephv1.CephRBDMirrorList{
			Items: []cephv1.CephRBDMirror{
				*unitinputs.CephRBDMirrorWithStatus(unitinputs.CephRBDMirror, "Ready"),
			},
		},
		"cephobjectstores": &cephv1.CephObjectStoreList{
			Items: []cephv1.CephObjectStore{
				*unitinputs.CephObjectStoreExternal.DeepCopy(),
			},
		},
		"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{
			Items: []cephv1.CephObjectStoreUser{},
		},
	}

	tests := []struct {
		name           string
		cephDpl        *cephlcmv1alpha1.CephDeployment
		cephVersion    *lcmcommon.CephVersion
		ccsettingsMap  map[string]string
		inProgressMsg  string
		failedMsg      string
		apiErrors      map[string]error
		inputResources map[string]runtime.Object
	}{
		{
			name: "apply cephdeployment - failed on labeling + annotating nodes and network policies",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := unitinputs.CephDeployMosk.DeepCopy()
				mc.Generation = int64(10)
				mc.Status.Phase = cephlcmv1alpha1.PhaseReady
				mc.Status.Message = "Ceph cluster configuration successfully applied"
				return mc
			}(),
			ccsettingsMap: ccsettingsMap,
			failedMsg:     "configuration apply is failed: failed to ensure label nodes, annotate nodes, network policies",
		},
		{
			name:    "apply cephdeployment - apply configuration is failed during all steps",
			cephDpl: fullCephDplSpec.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"nodes": nodesForApply.DeepCopy(),
				"configmaps": &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{
						*unitinputs.RookCephMonEndpoints.DeepCopy(),
					},
				},
				"networkpolicies": &networkingv1.NetworkPolicyList{
					Items: []networkingv1.NetworkPolicy{
						unitinputs.NetworkPolicyMds, unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw,
					},
				},
			},
			ccsettingsMap: ccsettingsMap,
			apiErrors: map[string]error{
				"get-cephclusters":     errors.New("get cluster api error"),
				"cluster-not-verified": errors.New("not verified"),
			},
			inProgressMsg: "configuration apply is in progress: label nodes",
			failedMsg:     "configuration apply is failed: failed to ensure cephcluster, cephblockpools, shared filesystems, storageclasses, cephclients, ceph object storage, RBD Mirroring, Openstack secret, ingress proxy, cluster state",
		},
		{
			name:    "apply cephdeployment - apply configuration is started and waiting for netpol",
			cephDpl: fullCephDplSpec.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"nodes":           &nodesForApply,
				"configmaps":      &corev1.ConfigMapList{},
				"networkpolicies": &networkingv1.NetworkPolicyList{},
			},
			ccsettingsMap: ccsettingsMap,
			inProgressMsg: "configuration apply is in progress: label nodes, network policies",
		},
		{
			name:    "apply cephdeployment - apply configuration is started after netpol",
			cephDpl: fullCephDplSpec.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"nodes":      &nodesForApply,
				"configmaps": &corev1.ConfigMapList{},
				"ingresses":  &networkingv1.IngressList{},
				"networkpolicies": &networkingv1.NetworkPolicyList{
					Items: []networkingv1.NetworkPolicy{
						unitinputs.NetworkPolicyMds, unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw,
					},
				},
				"services":             &corev1.ServiceList{},
				"secrets":              &corev1.SecretList{},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephblockpools":       &cephv1.CephBlockPoolList{},
				"cephclients":          &cephv1.CephClientList{},
				"cephclusters":         &cephv1.CephClusterList{},
				"cephfilesystems":      &cephv1.CephFilesystemList{},
				"cephrbdmirrors":       &cephv1.CephRBDMirrorList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{},
			},
			ccsettingsMap: ccsettingsMap,
			inProgressMsg: "configuration apply is in progress: cephcluster, cephblockpools, shared filesystems, cephclients, ceph object storage, RBD Mirroring, ingress proxy, cluster state",
			failedMsg:     "configuration apply is failed: failed to ensure storageclasses, Openstack secret",
		},
		{
			name:    "apply cephdeployment - apply configuration is started without netpol",
			cephDpl: fullCephDplSpec.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"nodes":                nodesForApply.DeepCopy(),
				"configmaps":           &corev1.ConfigMapList{},
				"ingresses":            &networkingv1.IngressList{},
				"networkpolicies":      &networkingv1.NetworkPolicyList{},
				"services":             &corev1.ServiceList{},
				"secrets":              &corev1.SecretList{},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephblockpools":       &cephv1.CephBlockPoolList{},
				"cephclients":          &cephv1.CephClientList{},
				"cephclusters":         &cephv1.CephClusterList{},
				"cephfilesystems":      &cephv1.CephFilesystemList{},
				"cephrbdmirrors":       &cephv1.CephRBDMirrorList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{},
			},
			inProgressMsg: "configuration apply is in progress: label nodes, cephcluster, cephblockpools, shared filesystems, cephclients, ceph object storage, RBD Mirroring, ingress proxy, cluster state",
			failedMsg:     "configuration apply is failed: failed to ensure storageclasses, Openstack secret",
		},
		{
			name:           "apply cephdeployment - apply configuration is in progress",
			cephDpl:        fullCephDplSpec.DeepCopy(),
			inputResources: inputResourcesForApply,
			ccsettingsMap:  ccsettingsMap,
			apiErrors:      map[string]error{"cluster-not-verified": errors.New("not verified")},
			inProgressMsg:  "configuration apply is in progress: cephcluster, shared filesystems, storageclasses, ceph object storage, Openstack secret, ingress proxy",
			failedMsg:      "configuration apply is failed: failed to ensure cluster state",
		},
		{
			name:           "apply cephdeployment - nothing to do",
			cephDpl:        fullCephDplSpec.DeepCopy(),
			inputResources: inputResourcesForApply,
			ccsettingsMap:  ccsettingsMap,
			inProgressMsg:  "",
		},
		{
			name: "apply cephdeployment - new ingress, nothing to do",
			cephDpl: func() *cephlcmv1alpha1.CephDeployment {
				mc := fullCephDplSpec.DeepCopy()
				mc.Spec.IngressConfig = unitinputs.CephIngressConfig.DeepCopy()
				return mc
			}(),
			inputResources: inputResourcesForApply,
			ccsettingsMap:  ccsettingsMap,
			inProgressMsg:  "",
		},
		{
			name:    "apply reconcile cephdeployment external - apply configuration is started",
			cephDpl: externalCephDplButWithCommonFields.DeepCopy(),
			inputResources: map[string]runtime.Object{
				"configmaps": &corev1.ConfigMapList{},
				"secrets": &corev1.SecretList{
					Items: []corev1.Secret{*unitinputs.ExternalConnectionSecretWithAdminAndRgw.DeepCopy()},
				},
				"nodes": &corev1.NodeList{
					Items: []corev1.Node{
						unitinputs.GetAvailableNode("node-1"),
						unitinputs.GetAvailableNode("node-2"),
						unitinputs.GetAvailableNode("node-3"),
					},
				},
				"storageclasses":       &storagev1.StorageClassList{},
				"cephclients":          &cephv1.CephClientList{},
				"cephclusters":         &cephv1.CephClusterList{},
				"cephrbdmirrors":       &cephv1.CephRBDMirrorList{},
				"cephobjectstores":     &cephv1.CephObjectStoreList{},
				"cephobjectstoreusers": &cephv1.CephObjectStoreUserList{},
				"networkpolicies": &networkingv1.NetworkPolicyList{
					Items: []networkingv1.NetworkPolicy{
						unitinputs.NetworkPolicyMds, unitinputs.NetworkPolicyMgr, unitinputs.NetworkPolicyMon, unitinputs.NetworkPolicyOsd, unitinputs.NetworkPolicyRgw,
					},
				},
			},
			ccsettingsMap: ccsettingsMap,
			inProgressMsg: "configuration apply is in progress: cephcluster, storageclasses, ceph object storage, RBD Mirroring",
		},
		{
			name:           "apply reconcile cephdeployment external - apply configuration is in progress",
			cephDpl:        externalCephDplButWithCommonFields.DeepCopy(),
			inputResources: inputResourcesForExternalApply,
			ccsettingsMap:  ccsettingsMap,
			inProgressMsg:  "configuration apply is in progress: cephcluster, storageclasses, ceph object storage",
		},
		{
			name:           "apply reconcile cephdeployment external - apply configuration has no changes",
			cephDpl:        externalCephDplButWithCommonFields.DeepCopy(),
			inputResources: inputResourcesForExternalApply,
			ccsettingsMap:  ccsettingsMap,
			inProgressMsg:  "",
		},
	}

	oldTimeFunc := lcmcommon.GetCurrentTimeString
	oldUnixTimeFunc := lcmcommon.GetCurrentUnixTimeString
	oldGenerateCrtFunc := lcmcommon.GenerateSelfSignedCert
	oldCephCmdFunc := lcmcommon.RunPodCommandWithValidation
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(&deployConfig{cephDpl: test.cephDpl}, test.ccsettingsMap)

			if test.cephVersion == nil {
				c.cdConfig.currentCephVersion = lcmcommon.Reef
			} else {
				c.cdConfig.currentCephVersion = test.cephVersion
			}
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "list", []string{"nodes"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "get", []string{"configmaps", "nodes", "pods", "secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "create", []string{"configmaps", "secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "update", []string{"configmaps", "nodes", "secrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.CoreV1(), "delete", []string{"services"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "list", []string{"storageclasses"}, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "create", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.StorageV1(), "update", []string{"storageclasses"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "get", []string{"ingresses", "networkpolicies"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "create", []string{"ingresses", "networkpolicies"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "delete", []string{"ingresses", "networkpolicies"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Kubeclientset.NetworkingV1(), "update", []string{"ingresses", "networkpolicies"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "list", cephAPIResources, test.inputResources, nil)
			faketestclients.FakeReaction(c.api.Rookclientset, "get", cephAPIResources, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "create", cephAPIResources, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(c.api.Rookclientset, "update", cephAPIResources, test.inputResources, test.apiErrors)
			//--- common function actions ---//
			lcmcommon.GenerateSelfSignedCert = func(_, _ string, _ []string) (string, string, string, error) {
				return "fake-key", "fake-crt", "fake-ca", nil
			}

			lcmcommon.RunPodCommandWithValidation = func(e lcmcommon.ExecConfig) (string, string, error) {
				if strings.Contains(e.Command, "radosgw-admin") {
					return unitinputs.CephZoneGroupInfoHostnamesFromIngress, "", nil
				} else if strings.Contains(e.Command, "/usr/local/bin/zonegroup_hostnames_update.sh") {
					return "", "", nil
				} else if strings.Contains(e.Command, "ceph auth get-key client.cinder") {
					return "cinder", "", nil
				} else if strings.Contains(e.Command, "ceph auth get-key client.nova") {
					return "nova", "", nil
				} else if strings.Contains(e.Command, "ceph auth get-key client.glance") {
					return "glance", "", nil
				} else if strings.Contains(e.Command, "ceph auth get-key") {
					return "fake", "", nil
				} else if strings.Contains(e.Command, "ceph osd pool ls -f json") {
					return unitinputs.CephOsdLspools, "", nil
				} else if e.Command == "ceph config get mgr mgr/progress/allow_pg_recovery_event" {
					return "false", "", nil
				} else if e.Command == "ceph config get osd bdev_enable_discard" {
					return "true", "", nil
				} else if e.Command == "ceph config get osd bdev_async_discard_threads" {
					return "true", "", nil
				} else if e.Command == "ceph config dump --format json" {
					dump := `[{
    "section": "client.rgw.rgw.store.a",
    "name": "rgw_keystone_admin_password",
    "value": "auth-password",
    "level": "advanced",
    "can_update_at_runtime": false,
    "mask": ""
},
{
    "section": "osd",
    "name": "bdev_async_discard_threads",
    "value": "1",
    "level": "advanced",
    "can_update_at_runtime": true,
    "mask": ""
},
{
    "section": "osd",
    "name": "bdev_enable_discard",
    "value": "true",
    "level": "advanced",
    "can_update_at_runtime": true,
    "mask": ""
}]`
					return dump, "", nil
				} else if strings.HasPrefix(e.Command, "ceph config set client.rgw.rgw.store.a rgw_keystone_admin_password") {
					return "", "", nil
				} else if strings.HasPrefix(e.Command, "ceph config set osd bdev_enable_discard true") {
					return "", "", nil
				} else if strings.HasPrefix(e.Command, "ceph config set osd bdev_async_discard_threads 1") {
					return "", "", nil
				} else if strings.HasPrefix(e.Command, "ceph fs subvolumegroup -f json") {
					return `[{"name":"csi"}]`, "", nil
				} else if strings.HasPrefix(e.Command, "ceph mgr module ls") {
					if test.apiErrors["cluster-not-verified"] != nil {
						return unitinputs.MgrModuleLsNoPrometheus, "", nil
					}
					return unitinputs.MgrModuleLsWithPrometheus, "", nil
				}
				return "", "", errors.New("unexpected run ceph command call " + e.Command)
			}
			lcmcommon.GetCurrentTimeString = func() string {
				return "2021-08-15T14:30:45+04:00"
			}
			lcmcommon.GetCurrentUnixTimeString = func() string {
				return "1675587456"
			}

			nodesList, err := lcmcommon.GetExpandedCephDeploymentNodeList(c.context, c.api.Client, test.cephDpl.Spec)
			assert.Nil(t, err)
			c.cdConfig.nodesListExpanded = nodesList

			applyInProgress, applyErr := c.applyConfiguration()
			assert.Equal(t, test.inProgressMsg, applyInProgress)
			assert.Equal(t, test.failedMsg, applyErr)
			// clean reactions before next test
			faketestclients.CleanupFakeClientReactions(c.api.Rookclientset)
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.CoreV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.StorageV1())
			faketestclients.CleanupFakeClientReactions(c.api.Kubeclientset.NetworkingV1())
		})
	}
	lcmcommon.GetCurrentTimeString = oldTimeFunc
	lcmcommon.GetCurrentUnixTimeString = oldUnixTimeFunc
	lcmcommon.GenerateSelfSignedCert = oldGenerateCrtFunc
	lcmcommon.RunPodCommandWithValidation = oldCephCmdFunc
	unsetTimestampsVar()
}
