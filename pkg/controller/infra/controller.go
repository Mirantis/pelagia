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

package infra

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	rookclient "github.com/rook/rook/pkg/client/clientset/versioned"
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

const ControllerName = "pelagia-infra-controller"

// Add creates new LCM Infra and Config controllers and adds to the Manager.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	// set required params to control
	lcmconfig.ParamsToControl = lcmconfig.ControlParamsHealth
	err := lcmconfig.Add(mgr)
	if err != nil {
		return errors.Wrap(err, "failed to add lcm config controller")
	}
	reconciler, err := newReconciler(mgr)
	if err != nil {
		return errors.Wrap(err, "failed to create lcm infra reconciler")
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
	KubeClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s client")
	}

	return &ReconcileLcmResources{
		Client:        mgr.GetClient(),
		Lcmclientset:  LcmClientset,
		Kubeclientset: KubeClientset,
		Rookclientset: RookClientset,
		Scheme:        mgr.GetScheme(),
	}, nil
}

func cephHealthPredicate[T *lcmv1alpha1.CephDeploymentHealth]() predicate.TypedFuncs[T] {
	return predicate.TypedFuncs[T]{
		CreateFunc:  func(_ event.TypedCreateEvent[T]) bool { return true },
		UpdateFunc:  func(_ event.TypedUpdateEvent[T]) bool { return false },
		DeleteFunc:  func(_ event.TypedDeleteEvent[T]) bool { return true },
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
	// blank assignment to verify that ReconcileLcmResources implements reconcile.Reconciler
	_ reconcile.Reconciler = &ReconcileLcmResources{}
	// init logger
	log = lcmcommon.InitLogger(true)
)

// ReconcileLcmResources reconciles a CephDeploymentHealth object
type ReconcileLcmResources struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	Client        client.Client
	Lcmclientset  lcmclient.Interface
	Kubeclientset kubernetes.Interface
	Rookclientset rookclient.Interface
	Scheme        *runtime.Scheme
}

func (r *ReconcileLcmResources) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	lcmConfig := lcmconfig.GetConfiguration(request.Namespace)
	sublog := log.With().Str(lcmcommon.LoggerObjectField, fmt.Sprintf("namespace '%v'", request.Namespace)).Logger()
	deploymentHealth, err := r.Lcmclientset.LcmV1alpha1().CephDeploymentHealths(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			sublog.Info().Msg("CephDeploymentHealth is removed, nothing to setup")
			return reconcile.Result{}, nil
		}
		sublog.Error().Err(err).Msg("")
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, err
	}
	// add owner refs for disk-daemon
	lcmOwnerRefs, err := lcmcommon.GetObjectOwnerRef(deploymentHealth, r.Scheme)
	if err != nil {
		err = errors.Wrap(err, "failed to get owner refs for CephDeploymentHealth")
		sublog.Error().Err(err).Msg("")
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}
	controllerImage, found := os.LookupEnv(controllerImageVar)
	if !found || controllerImage == "" {
		sublog.Error().Msgf("required env var '%s' is not set or empty", controllerImageVar)
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}

	cephCluster, err := r.Rookclientset.CephV1().CephClusters(lcmConfig.RookNamespace).Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		err := errors.Wrapf(err, "failed to get cephcluster '%s/%s'", lcmConfig.RookNamespace, request.Name)
		sublog.Error().Err(err).Msg("")
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}
	// add owner refs for tools box
	cephOwnerRefs, err := lcmcommon.GetObjectOwnerRef(cephCluster, r.Scheme)
	if err != nil {
		err = errors.Wrap(err, "failed to get owner refs for CephCluster")
		sublog.Error().Err(err).Msg("")
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}

	// init config
	config := &cephDeploymentInfraConfig{
		context:   ctx,
		api:       r,
		lcmConfig: &lcmConfig,
		log:       &sublog,
		infraConfig: infraConfig{
			name:            request.Name,
			namespace:       request.Namespace,
			lcmOwnerRefs:    lcmOwnerRefs,
			cephOwnerRefs:   cephOwnerRefs,
			externalCeph:    cephCluster.Spec.External.Enable,
			controllerImage: controllerImage,
		},
	}
	if cephCluster.Status.CephVersion != nil && cephCluster.Status.CephVersion.Image != "" {
		config.infraConfig.cephImage = cephCluster.Status.CephVersion.Image
	}
	if !cephCluster.Spec.External.Enable && cephCluster.Spec.Placement != nil {
		if v, ok := cephCluster.Spec.Placement[cephv1.KeyOSD]; ok {
			config.infraConfig.osdPlacement = v
		}
	}

	err = config.ensureToolBox()
	if err != nil {
		sublog.Error().Err(err).Msg("")
	}
	err = config.ensureDiskDaemon()
	if err != nil {
		sublog.Error().Err(err).Msg("")
	}
	err = config.checkRookOperatorReplicas()
	if err != nil {
		sublog.Error().Err(err).Msg("")
	}
	return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
}
