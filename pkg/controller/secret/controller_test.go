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

package secret

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	lcmconfig "github.com/Mirantis/pelagia/pkg/controller/config"
	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
	fscheme "github.com/Mirantis/pelagia/test/unit/scheme"
)

func FakeReconciler() *ReconcileCephSecrets {
	return &ReconcileCephSecrets{
		Client:           faketestclients.GetClient(nil),
		Kubeclientset:    faketestclients.GetFakeKubeclient(),
		Rookclientset:    faketestclients.GetFakeRookclient(),
		Cephdplclientset: faketestclients.GetFakeLcmclient(),
		Scheme:           fscheme.Scheme,
	}
}

func fakeSecretConfig(sconfig *secretsConfig) *cephDeploymentSecretConfig {
	lcmConfig := lcmconfig.GetConfiguration("rook-ceph")
	sublog := log.With().Str(lcmcommon.LoggerObjectField, "cephdeploymentsecret").Logger().Level(lcmConfig.DeployParams.LogLevel)
	sc := secretsConfig{
		cephDpl: &cephlcmv1alpha1.CephDeployment{ObjectMeta: unitinputs.LcmObjectMeta},
	}
	if sconfig != nil {
		sc = *sconfig
	}
	return &cephDeploymentSecretConfig{
		context:       context.TODO(),
		api:           FakeReconciler(),
		log:           &sublog,
		lcmConfig:     &lcmConfig,
		secretsConfig: sc,
	}
}

func TestCephDeploymentSecretReconcile(t *testing.T) {
	req := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: unitinputs.LcmObjectMeta.Namespace, Name: unitinputs.LcmObjectMeta.Name}}
	resInterval := reconcile.Result{RequeueAfter: requeueAfterInterval}
	resNoRequeue := reconcile.Result{}
	tests := []struct {
		name               string
		inputResources     map[string]runtime.Object
		apiErrors          map[string]error
		updateStatusFailed bool
		expectedStatus     *cephlcmv1alpha1.CephDeploymentSecretStatus
		expectedError      string
		expectedResult     reconcile.Result
	}{
		{
			name: "reconcile cephdeploymentsecret - not found",
			inputResources: map[string]runtime.Object{
				"cephdeploymentsecrets": &cephlcmv1alpha1.CephDeploymentSecretList{},
			},
			expectedResult: resNoRequeue,
		},
		{
			name:           "reconcile cephdeploymentsecret - failed to get",
			expectedResult: resInterval,
			expectedError:  "failed to get CephDeploymentSecret lcm-namespace/cephcluster: failed to get resource(s) kind of 'cephdeploymentsecrets': list object is not specified in test",
		},
		{
			name: "reconcile cephdeploymentsecret - cephdeployment not found, skip secret processing",
			inputResources: map[string]runtime.Object{
				"cephdeploymentsecrets": &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.EmptyCephSecret.DeepCopy()}},
				"cephdeployments":       &cephlcmv1alpha1.CephDeploymentList{},
			},
			expectedStatus: &cephlcmv1alpha1.CephDeploymentSecretStatus{
				State:            cephlcmv1alpha1.HealthStateFailed,
				LastSecretCheck:  time.Date(2021, 8, 15, 14, 30, 47, 0, time.Local).Format(time.RFC3339),
				LastSecretUpdate: time.Date(2021, 8, 15, 14, 30, 47, 0, time.Local).Format(time.RFC3339),
				Messages:         []string{"failed to get CephDeployment lcm-namespace/cephcluster: cephdeployments \"cephcluster\" not found"},
			},
			expectedResult: resNoRequeue,
		},
		{
			name: "reconcile cephdeploymentsecret - failed to get cephdeployment",
			inputResources: map[string]runtime.Object{
				"cephdeploymentsecrets": &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.EmptyCephSecret.DeepCopy()}},
			},
			expectedStatus: &cephlcmv1alpha1.CephDeploymentSecretStatus{
				State:            cephlcmv1alpha1.HealthStateFailed,
				LastSecretCheck:  time.Date(2021, 8, 15, 14, 30, 48, 0, time.Local).Format(time.RFC3339),
				LastSecretUpdate: time.Date(2021, 8, 15, 14, 30, 48, 0, time.Local).Format(time.RFC3339),
				Messages:         []string{"failed to get CephDeployment lcm-namespace/cephcluster: failed to get resource(s) kind of 'cephdeployments': list object is not specified in test"},
			},
			expectedResult: resInterval,
		},
		{
			name: "reconcile cephdeploymentsecret - cephdeployment not found and status update failed",
			inputResources: map[string]runtime.Object{
				"cephdeploymentsecrets": &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.EmptyCephSecret.DeepCopy()}},
			},
			updateStatusFailed: true,
			expectedResult:     resInterval,
		},
		{
			name: "reconcile cephdeploymentsecret - update ownerRefs",
			inputResources: map[string]runtime.Object{
				"cephdeploymentsecrets": &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.EmptyCephSecret.DeepCopy()}},
				"cephdeployments":       &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.CephDeployNonMoskForSecret}},
			},
			expectedResult: reconcile.Result{Requeue: true},
		},
		{
			name: "reconcile cephdeploymentsecret - update ownerRefs failed",
			inputResources: map[string]runtime.Object{
				"cephdeploymentsecrets": &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.EmptyCephSecret.DeepCopy()}},
				"cephdeployments":       &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.CephDeployNonMoskForSecret}},
			},
			apiErrors: map[string]error{"update-cephdeploymentsecrets": errors.New("update failed")},
			expectedStatus: &cephlcmv1alpha1.CephDeploymentSecretStatus{
				State:            cephlcmv1alpha1.HealthStateFailed,
				LastSecretCheck:  time.Date(2021, 8, 15, 14, 30, 51, 0, time.Local).Format(time.RFC3339),
				LastSecretUpdate: time.Date(2021, 8, 15, 14, 30, 51, 0, time.Local).Format(time.RFC3339),
				Messages:         []string{"failed to update CephDeploymentSecret: update failed"},
			},
			expectedResult: resInterval,
		},
		{
			name: "reconcile cephdeploymentsecret - secrets info updated no issues",
			inputResources: map[string]runtime.Object{
				"cephdeploymentsecrets": &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.GetNewCephSecret()}},
				"cephdeployments":       &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.CephDeployNonMoskForSecret}},
				"secrets":               &corev1.SecretList{Items: []corev1.Secret{unitinputs.CephAdminKeyringSecret}},
			},
			expectedStatus: unitinputs.CephSecretReady.Status,
			expectedResult: resInterval,
		},
		{
			name: "reconcile cephdeploymentsecret - secrets info updated has issues",
			inputResources: map[string]runtime.Object{
				"cephdeploymentsecrets": &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.CephSecretNotReady.DeepCopy()}},
				"cephdeployments":       &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.CephDeployNonMoskForSecret}},
				"secrets":               &corev1.SecretList{Items: []corev1.Secret{}},
			},
			expectedStatus: unitinputs.CephSecretNotReady.Status,
			expectedResult: resInterval,
		},
		{
			name: "reconcile cephdeploymentsecret - update failed",
			inputResources: map[string]runtime.Object{
				"cephdeploymentsecrets": &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.CephSecretReady.DeepCopy()}},
				"cephdeployments":       &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.CephDeployNonMoskForSecret}},
				"secrets":               &corev1.SecretList{Items: []corev1.Secret{}},
			},
			updateStatusFailed: true,
			expectedResult:     resInterval,
		},
		{
			name: "reconcile cephdeploymentsecret - only check state update",
			inputResources: map[string]runtime.Object{
				"cephdeploymentsecrets": &cephlcmv1alpha1.CephDeploymentSecretList{Items: []cephlcmv1alpha1.CephDeploymentSecret{*unitinputs.CephSecretNotReady.DeepCopy()}},
				"cephdeployments":       &cephlcmv1alpha1.CephDeploymentList{Items: []cephlcmv1alpha1.CephDeployment{unitinputs.CephDeployNonMoskForSecret}},
				"secrets":               &corev1.SecretList{Items: []corev1.Secret{}},
			},
			expectedStatus: func() *cephlcmv1alpha1.CephDeploymentSecretStatus {
				secretStatus := unitinputs.CephSecretNotReady.Status.DeepCopy()
				secretStatus.LastSecretCheck = time.Date(2021, 8, 15, 14, 30, 55, 0, time.Local).Format(time.RFC3339)
				return secretStatus
			}(),
			expectedResult: resInterval,
		},
	}
	oldTimeFunc := lcmcommon.GetCurrentTimeString
	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := FakeReconciler()
			lcmcommon.GetCurrentTimeString = func() string {
				return time.Date(2021, 8, 15, 14, 30, 45+idx, 0, time.Local).Format(time.RFC3339)
			}

			r.Client = faketestclients.GetClient(nil)
			checkStatus := false
			if test.inputResources["cephdeploymentsecrets"] != nil {
				list := test.inputResources["cephdeploymentsecrets"].(*cephlcmv1alpha1.CephDeploymentSecretList)
				if len(list.Items) == 1 && !test.updateStatusFailed {
					checkStatus = true
					r.Client = faketestclients.GetClient(faketestclients.GetClientBuilder().WithStatusSubresource(&list.Items[0]).WithObjects(&list.Items[0]))
				}
			}
			faketestclients.FakeReaction(r.Cephdplclientset, "get", []string{"cephdeploymentsecrets", "cephdeployments"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Cephdplclientset, "update", []string{"cephdeploymentsecrets"}, test.inputResources, test.apiErrors)
			faketestclients.FakeReaction(r.Kubeclientset.CoreV1(), "get", []string{"secrets"}, test.inputResources, test.apiErrors)

			res, err := r.Reconcile(context.TODO(), req)
			assert.Equal(t, test.expectedResult, res)
			if test.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
			}
			cephSecret := &cephlcmv1alpha1.CephDeploymentSecret{}
			err = r.Client.Get(context.TODO(), req.NamespacedName, cephSecret)
			if checkStatus {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedStatus, cephSecret.Status)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, "cephdeploymentsecrets.lcm.mirantis.com \"cephcluster\" not found", err.Error())
			}
			// clean reactions
			faketestclients.CleanupFakeClientReactions(r.Cephdplclientset)
			faketestclients.CleanupFakeClientReactions(r.Kubeclientset.CoreV1())
		})
	}
	lcmcommon.GetCurrentTimeString = oldTimeFunc
}
