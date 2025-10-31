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
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	rookclient "github.com/rook/rook/pkg/client/clientset/versioned"
	"github.com/rs/zerolog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmclient "github.com/Mirantis/pelagia/pkg/client/clientset/versioned"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	lcmconfig "github.com/Mirantis/pelagia/pkg/controller/config"
)

const (
	ControllerName = "pelagia-secret-controller"
)

// Add creates a new Ceph secrets controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	lcmconfig.ParamsToControl = lcmconfig.ControlParamsCephDpl
	err := lcmconfig.Add(mgr)
	if err != nil {
		return err
	}
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	config, _ := rest.InClusterConfig()
	RookClientset, _ := rookclient.NewForConfig(config)
	CephdplClientset, _ := lcmclient.NewForConfig(config)
	kubeclientset, _ := kubernetes.NewForConfig(config)

	return &ReconcileCephSecrets{Client: mgr.GetClient(), Kubeclientset: kubeclientset, Rookclientset: RookClientset, Cephdplclientset: CephdplClientset, Scheme: mgr.GetScheme()}
}

func cephDplSecretPredicate[T *cephlcmv1alpha1.CephDeploymentSecret]() predicate.TypedFuncs[T] {
	return predicate.TypedFuncs[T]{
		CreateFunc: func(_ event.TypedCreateEvent[T]) bool { return true },
		UpdateFunc: func(_ event.TypedUpdateEvent[T]) bool { return false },
		DeleteFunc: func(_ event.TypedDeleteEvent[T]) bool { return false },
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CephDeploymentSecret
	err = c.Watch(source.Kind(
		mgr.GetCache(),
		&cephlcmv1alpha1.CephDeploymentSecret{},
		&handler.TypedEnqueueRequestForObject[*cephlcmv1alpha1.CephDeploymentSecret]{},
		cephDplSecretPredicate[*cephlcmv1alpha1.CephDeploymentSecret]()))
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCephSecrets implements reconcile.Reconciler
var (
	_   reconcile.Reconciler = &ReconcileCephSecrets{}
	log                      = lcmcommon.InitLogger(true)
)

// ReconcileCephSecrets reconciles a CephDeploymentSecret object
type ReconcileCephSecrets struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	Client           client.Client
	Kubeclientset    kubernetes.Interface
	Rookclientset    rookclient.Interface
	Cephdplclientset lcmclient.Interface
	Scheme           *runtime.Scheme
}

func (r *ReconcileCephSecrets) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	lcmConfig := lcmconfig.GetConfiguration(request.Namespace)
	sublog := log.With().Str(lcmcommon.LoggerObjectField, fmt.Sprintf("cephdeploymentsecret '%v'", request.NamespacedName)).Logger().Level(lcmConfig.DeployParams.LogLevel)
	sublog.Debug().Msg("reconcile started")

	cdSecret, err := r.Cephdplclientset.LcmV1alpha1().CephDeploymentSecrets(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		errMsg := errors.Wrapf(err, "failed to get CephDeploymentSecret %s/%s", request.Namespace, request.Name)
		sublog.Error().Err(errMsg).Msg("")
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, errMsg
	}

	cephDpl, err := r.Cephdplclientset.LcmV1alpha1().CephDeployments(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		r.setFailedState(ctx, sublog, request.Namespace, request.Name, fmt.Sprintf("failed to get CephDeployment %s/%s: %v", request.Namespace, request.Name, err))
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}

	ownerRefs, err := lcmcommon.GetObjectOwnerRef(cephDpl, r.Scheme)
	if err != nil {
		msg := fmt.Sprintf("failed to get ownerRefs for CephDeploymentSecret '%s/%s' associated with CephDeployment '%s/%s': %v",
			cdSecret.Namespace, cdSecret.Name, cephDpl.Namespace, cephDpl.Name, err)
		r.setFailedState(ctx, sublog, request.Namespace, request.Name, msg)
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}
	if !reflect.DeepEqual(ownerRefs, cdSecret.OwnerReferences) {
		sublog.Info().Msgf("updating owner references for CephDeploymentSecret '%s/%s'", cdSecret.Namespace, cdSecret.Name)
		cdSecret.OwnerReferences = ownerRefs
		_, err = r.Cephdplclientset.LcmV1alpha1().CephDeploymentSecrets(cdSecret.Namespace).Update(ctx, cdSecret, metav1.UpdateOptions{})
		if err != nil {
			r.setFailedState(ctx, sublog, request.Namespace, request.Name, fmt.Sprintf("failed to update CephDeploymentSecret: %v", err))
			return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
		}
		return reconcile.Result{Requeue: true}, nil
	}

	secretConfig := &cephDeploymentSecretConfig{
		context:       ctx,
		api:           r,
		log:           &sublog,
		lcmConfig:     &lcmConfig,
		secretsConfig: secretsConfig{cephDpl: cephDpl},
	}

	newSecretsInfo, issues := secretConfig.getSecretsStatusInfo()
	if len(issues) > 0 {
		sublog.Error().Msgf("issues found during secrets processing: [%s]", strings.Join(issues, ", "))
	}

	err = r.updateCephDeploymentSecretStatus(ctx, sublog, request.Namespace, request.Name, buildCephDeploymentSecretStatus(newSecretsInfo, issues))
	if err != nil {
		sublog.Error().Err(err).Msgf("failed to update CephDeploymentSecret '%s/%s' status", cdSecret.Namespace, cdSecret.Name)
	}
	return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
}

func (r *ReconcileCephSecrets) setFailedState(ctx context.Context, log zerolog.Logger, namespace, name string, msg string) {
	log.Error().Msg(msg)
	statusErr := r.updateCephDeploymentSecretStatus(ctx, log, namespace, name, buildCephDeploymentSecretStatus(nil, []string{msg}))
	if statusErr != nil {
		log.Error().Err(statusErr).Msg("")
	}
}

func buildCephDeploymentSecretStatus(secretsInfo *cephlcmv1alpha1.CephDeploymentSecretsInfo, issues []string) *cephlcmv1alpha1.CephDeploymentSecretStatus {
	newStatus := &cephlcmv1alpha1.CephDeploymentSecretStatus{
		State:       cephlcmv1alpha1.HealthStateOk,
		SecretsInfo: secretsInfo,
	}
	if len(issues) > 0 {
		newStatus.Messages = issues
		newStatus.State = cephlcmv1alpha1.HealthStateFailed
	}
	return newStatus
}

func (r *ReconcileCephSecrets) updateCephDeploymentSecretStatus(ctx context.Context, objlog zerolog.Logger, namespace, name string, status *cephlcmv1alpha1.CephDeploymentSecretStatus) error {
	cephDplSecret := &cephlcmv1alpha1.CephDeploymentSecret{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, cephDplSecret)
	if err != nil {
		return errors.Wrapf(err, "failed to get CephDeploymentSecret '%s/%s' to update status", namespace, name)
	}
	// save current timings to new status to avoid extra diff, then put correct timings
	timeNow := lcmcommon.GetCurrentTimeString()
	if cephDplSecret.Status != nil {
		status.LastSecretCheck = cephDplSecret.Status.LastSecretCheck
		status.LastSecretUpdate = cephDplSecret.Status.LastSecretUpdate
	}
	if !reflect.DeepEqual(cephDplSecret.Status, status) {
		objlog.Debug().Msgf("updating status with new secrets info")
		lcmcommon.ShowObjectDiff(objlog, cephDplSecret.Status, status)
		status.LastSecretUpdate = timeNow
	} else {
		objlog.Debug().Msgf("updating status with new check timestamps")
	}
	status.LastSecretCheck = timeNow
	err = cephlcmv1alpha1.UpdateCephDeploymentSecretStatus(ctx, cephDplSecret, status, r.Client)
	if err != nil {
		return errors.Wrapf(err, "failed to update CephDeploymentSecret %s/%s status", cephDplSecret.Namespace, cephDplSecret.Name)
	}
	return nil
}
