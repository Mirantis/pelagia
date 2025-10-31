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

package health

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	claimClient "github.com/kube-object-storage/lib-bucket-provisioner/pkg/client/clientset/versioned"
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

	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmclient "github.com/Mirantis/pelagia/pkg/client/clientset/versioned"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	lcmconfig "github.com/Mirantis/pelagia/pkg/controller/config"
)

const ControllerName = "pelagia-health-controller"

// Add creates new LCM Health and Config controllers and adds to the Manager.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	// set required params to control for Health controller
	lcmconfig.ParamsToControl = lcmconfig.ControlParamsHealth
	err := lcmconfig.Add(mgr)
	if err != nil {
		return errors.Wrap(err, "failed to add lcm config controller")
	}
	reconciler, err := newReconciler(mgr)
	if err != nil {
		return errors.Wrap(err, "failed to create lcm health reconciler")
	}
	return add(mgr, reconciler)
}

// newReconciler returns a new reconcile.Reconciler or error
func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rest config")
	}
	RookClientset, err := rookclient.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rook client")
	}
	LcmClientset, err := lcmclient.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get lcm client")
	}
	ClaimClientset, err := claimClient.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get bucket proviosioner client")
	}
	KubeClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s client")
	}

	return &ReconcileCephDeploymentHealth{
		Config:         config,
		Client:         mgr.GetClient(),
		Lcmclientset:   LcmClientset,
		Kubeclientset:  KubeClientset,
		Rookclientset:  RookClientset,
		Claimclientset: ClaimClientset,
		Scheme:         mgr.GetScheme(),
	}, nil
}

func cephHealthPredicate[T *lcmv1alpha1.CephDeploymentHealth]() predicate.TypedFuncs[T] {
	return predicate.TypedFuncs[T]{
		UpdateFunc:  func(_ event.TypedUpdateEvent[T]) bool { return false },
		DeleteFunc:  func(_ event.TypedDeleteEvent[T]) bool { return false },
		GenericFunc: func(_ event.TypedGenericEvent[T]) bool { return false },
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CephDeploymentHealth
	err = c.Watch(source.Kind(
		mgr.GetCache(),
		&lcmv1alpha1.CephDeploymentHealth{},
		&handler.TypedEnqueueRequestForObject[*lcmv1alpha1.CephDeploymentHealth]{},
		cephHealthPredicate[*lcmv1alpha1.CephDeploymentHealth]()))
	if err != nil {
		return err
	}

	return nil
}

var (
	// blank assignment to verify that ReconcileCephDeploymentHealth implements reconcile.Reconciler
	_                    reconcile.Reconciler = &ReconcileCephDeploymentHealth{}
	log                  zerolog.Logger       = lcmcommon.InitLogger(true)
	requeueAfterInterval                      = 30 * time.Second
)

// ReconcileCephDeploymentHealth reconciles a CephDeploymentHealth object
type ReconcileCephDeploymentHealth struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	Config         *rest.Config
	Client         client.Client
	Lcmclientset   lcmclient.Interface
	Kubeclientset  kubernetes.Interface
	Rookclientset  rookclient.Interface
	Claimclientset claimClient.Interface
	Scheme         *runtime.Scheme
}

func (r *ReconcileCephDeploymentHealth) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	lcmConfig := lcmconfig.GetConfiguration(request.Namespace)
	sublog := log.With().Str(lcmcommon.LoggerObjectField, fmt.Sprintf("cephdeploymenthealth '%v'", request.NamespacedName)).Logger().Level(lcmConfig.HealthParams.LogLevel)
	sublog.Debug().Msg("reconcile started")
	_, err := r.Lcmclientset.LcmV1alpha1().CephDeploymentHealths(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		sublog.Error().Err(err).Msg("")
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, err
	}

	// init health config
	newHealthConfig := &cephDeploymentHealthConfig{
		context:   ctx,
		api:       r,
		lcmConfig: &lcmConfig,
		log:       &sublog,
		healthConfig: healthConfig{
			name:        request.Name,
			namespace:   request.Namespace,
			cephCluster: nil,
			rgwOpts:     rgwOpts{},
			sharedFilesystemOpts: sharedFilesystemOpts{
				mdsDaemonsDesired: map[string]map[string]int{},
			},
		},
	}

	newHealthStatus, verificationIssues := newHealthConfig.cephDeploymentVerification()
	if len(verificationIssues) > 0 {
		sublog.Error().Msgf("issues found during ceph deployment verification: [%s]", strings.Join(verificationIssues, ", "))
	}

	r.updateCephDeploymentHealthStatus(ctx, sublog, request, newHealthStatus, verificationIssues)
	sublog.Debug().Msg("reconcile finished")
	return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
}

func (r *ReconcileCephDeploymentHealth) updateCephDeploymentHealthStatus(ctx context.Context, objlog zerolog.Logger, req reconcile.Request, healthReport *lcmv1alpha1.CephDeploymentHealthReport, reportIssues []string) {
	var err error
	deploymentHealth := &lcmv1alpha1.CephDeploymentHealth{}
	err = r.Client.Get(ctx, req.NamespacedName, deploymentHealth)
	if err == nil {
		// save current timings to new status to avoid extra diff, then put correct timings
		timeNow := lcmcommon.GetCurrentTimeString()
		newStatus := lcmv1alpha1.CephDeploymentHealthStatus{
			State:            lcmv1alpha1.HealthStateOk,
			HealthReport:     healthReport,
			LastHealthCheck:  deploymentHealth.Status.LastHealthCheck,
			LastHealthUpdate: deploymentHealth.Status.LastHealthUpdate,
		}
		if len(reportIssues) > 0 {
			newStatus.Issues = reportIssues
			newStatus.State = lcmv1alpha1.HealthStateFailed
		}
		if !reflect.DeepEqual(deploymentHealth.Status, newStatus) {
			objlog.Debug().Msgf("updating health status with new health info")
			lcmcommon.ShowObjectDiff(objlog, deploymentHealth.Status, newStatus)
			newStatus.LastHealthUpdate = timeNow
		} else {
			objlog.Debug().Msgf("updating health status with new check timestamps")
		}
		newStatus.LastHealthCheck = timeNow
		err = lcmv1alpha1.UpdateCephHealthDeploymentStatus(ctx, deploymentHealth, newStatus, r.Client)
	}
	if err != nil {
		objlog.Error().Err(errors.Wrap(err, "failed to update status")).Msg("")
	}
}
