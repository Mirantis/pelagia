/*
Copyright 2025 The Mirantis Authors.

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

package osdremove

import (
	"context"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
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

const ControllerName = "pelagia-osdremove-task-controller"

// Add creates new LCM Task and Config controllers and adds to the Manager.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	lcmconfig.ParamsToControl = lcmconfig.ControlParamsTask
	err := lcmconfig.Add(mgr)
	if err != nil {
		return errors.Wrap(err, "failed to add lcm config controller")
	}
	reconciler, err := newReconciler(mgr)
	if err != nil {
		return errors.Wrap(err, "failed to create lcm osdremove task reconciler")
	}
	return add(mgr, reconciler)
}

// newReconciler returns a new reconcile.Reconciler
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

	return &ReconcileCephOsdRemoveTask{
		Config:        config,
		Client:        mgr.GetClient(),
		Lcmclientset:  LcmClientset,
		Kubeclientset: KubeClientset,
		Rookclientset: RookClientset,
		Scheme:        mgr.GetScheme(),
	}, nil
}

func checkTaskActive(taskStatus *lcmv1alpha1.CephOsdRemoveTaskStatus) bool {
	if taskStatus != nil {
		if taskStatus.Phase == lcmv1alpha1.TaskPhaseCompleted ||
			taskStatus.Phase == lcmv1alpha1.TaskPhaseCompletedWithWarnings ||
			taskStatus.Phase == lcmv1alpha1.TaskPhaseFailed ||
			taskStatus.Phase == lcmv1alpha1.TaskPhaseValidationFailed ||
			taskStatus.Phase == lcmv1alpha1.TaskPhaseAborted {
			return false
		}
	}
	return true
}

func cephTaskPredicate[T *lcmv1alpha1.CephOsdRemoveTask]() predicate.TypedFuncs[T] {
	return predicate.TypedFuncs[T]{
		CreateFunc: func(e event.TypedCreateEvent[T]) bool {
			obj := (*lcmv1alpha1.CephOsdRemoveTask)(e.Object)
			return checkTaskActive(obj.Status)
		},
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

	// Watch for changes to primary resource CephOsdRemoveTask
	err = c.Watch(source.Kind(
		mgr.GetCache(),
		&lcmv1alpha1.CephOsdRemoveTask{},
		&handler.TypedEnqueueRequestForObject[*lcmv1alpha1.CephOsdRemoveTask]{},
		cephTaskPredicate[*lcmv1alpha1.CephOsdRemoveTask]()))
	if err != nil {
		return err
	}

	return nil
}

var (
	// blank assignment to verify that ReconcileCephOsdRemoveTask implements reconcile.Reconciler
	_ reconcile.Reconciler = &ReconcileCephOsdRemoveTask{}
	// init logger
	log = lcmcommon.InitLogger(true)
)

// ReconcileCephOsdRemoveTask reconciles a CephRequest object
type ReconcileCephOsdRemoveTask struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	Config        *rest.Config
	Client        client.Client
	Kubeclientset kubernetes.Interface
	Rookclientset rookclient.Interface
	Lcmclientset  lcmclient.Interface
	Scheme        *runtime.Scheme
}

func getOldestCephOsdRemoveTaskName(cephTasks []lcmv1alpha1.CephOsdRemoveTask) string {
	i := -1
	for idx, curTask := range cephTasks {
		// ignore all completed and failed requests
		if !checkTaskActive(curTask.Status) {
			continue
		}
		if i == -1 {
			i = idx
			continue
		}
		reqTime := curTask.GetCreationTimestamp()
		prevTime := cephTasks[i].GetCreationTimestamp()
		if (&reqTime).Before(&prevTime) {
			i = idx
		}
	}
	if i == -1 {
		return ""
	}
	return cephTasks[i].Name
}

func (r *ReconcileCephOsdRemoveTask) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	lcmConfig := lcmconfig.GetConfiguration(request.Namespace)
	sublog := log.With().Str(lcmcommon.LoggerObjectField, fmt.Sprintf("cephosdremovetask '%v'", request.NamespacedName)).Logger().Level(lcmConfig.TaskParams.LogLevel)
	sublog.Info().Msg("reconcile started")
	// Find requested resource and raise error if it's not exists or some error occurred
	cephTask, err := r.Lcmclientset.LcmV1alpha1().CephOsdRemoveTasks(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		sublog.Error().Err(err).Msg("")
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, err
	}

	// Initiate CephOsdRemoveTask with Pending phase if necessary (if it has no status yet)
	if cephTask.Status == nil {
		sublog.Info().Msgf("initiating with '%v' phase", lcmv1alpha1.TaskPhasePending)
		err = r.updateCephOsdRemoveTaskStatus(ctx, request, prepareInitStatus(cephTask))
		if err != nil {
			sublog.Error().Err(err).Msg("")
			return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
		}
		return reconcile.Result{Requeue: true}, nil
	}

	cephTask.Status.PhaseInfo = ""
	deploymentHealthList, err := r.Lcmclientset.LcmV1alpha1().CephDeploymentHealths(request.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		sublog.Error().Err(err).Msg("")
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}
	//we should have only one CephDeploymentHealth per namespace
	if len(deploymentHealthList.Items) != 1 {
		if len(deploymentHealthList.Items) > 1 {
			if checkTaskActive(cephTask.Status) {
				errMsg := "multiple CephDeploymentHealth objects found in namespace"
				sublog.Error().Msgf("aborting, %s", errMsg)
				err = r.updateCephOsdRemoveTaskStatus(ctx, request, prepareAbortStatus(cephTask.Status, errMsg))
			}
		} else {
			sublog.Info().Msg("stale, no related CephDeploymentHealth resource found in namespace, removing")
			err = r.Lcmclientset.LcmV1alpha1().CephOsdRemoveTasks(request.Namespace).Delete(ctx, request.Name, metav1.DeleteOptions{})
		}
		if err != nil {
			sublog.Error().Err(err).Msg("")
			return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
		}
		return reconcile.Result{}, nil
	}

	cephDeploymentHealth := &deploymentHealthList.Items[0]
	ownerRefs, err := lcmcommon.GetObjectOwnerRef(cephDeploymentHealth, r.Scheme)
	if err != nil {
		sublog.Error().Err(errors.Wrap(err, "owner refs set failed")).Msg("")
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}

	if !reflect.DeepEqual(ownerRefs, cephTask.OwnerReferences) {
		sublog.Info().Msg("updating owner references")
		cephTask.OwnerReferences = ownerRefs
		_, err := r.Lcmclientset.LcmV1alpha1().CephOsdRemoveTasks(cephTask.Namespace).Update(ctx, cephTask, metav1.UpdateOptions{})
		if err != nil {
			sublog.Error().Err(err).Msg("")
			return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// do not abort on api error, since cephdeploymenthealth contains cephcluster status
	// so it may be simple API throttling error or so, re-run
	cephCluster, err := r.Rookclientset.CephV1().CephClusters(lcmConfig.RookNamespace).Get(ctx, cephDeploymentHealth.Name, metav1.GetOptions{})
	if err != nil {
		sublog.Error().Err(err).Msg("")
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}
	if cephCluster.Spec.External.Enable {
		abortReason := "detected external CephCluster configuration"
		sublog.Info().Msgf("aborting, %s", abortReason)
		err = r.updateCephOsdRemoveTaskStatus(ctx, request, prepareAbortStatus(cephTask.Status, abortReason))
		if err != nil {
			sublog.Error().Err(err).Msg("")
			return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
		}
		return reconcile.Result{}, nil
	}
	if cephCluster.Status.CephStatus == nil || cephCluster.Status.CephStatus.FSID == "" {
		msg := "CephCluster is not deployed yet, no fsid provided"
		sublog.Warn().Msg(msg)
		cephTask.Status.PhaseInfo = msg
		err = r.updateCephOsdRemoveTaskStatus(ctx, request, cephTask.Status)
		if err != nil {
			sublog.Error().Err(err).Msg("")
		}
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}
	if cephDeploymentHealth.Status.HealthReport == nil || cephDeploymentHealth.Status.HealthReport.RookCephObjects == nil ||
		cephDeploymentHealth.Status.HealthReport.RookCephObjects.CephCluster == nil {
		msg := "related CephDeploymentHealth has no CephCluster status yet"
		sublog.Warn().Msg(msg)
		cephTask.Status.PhaseInfo = msg
		err = r.updateCephOsdRemoveTaskStatus(ctx, request, cephTask.Status)
		if err != nil {
			sublog.Error().Err(err).Msg("")
		}
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}
	if cephDeploymentHealth.Status.HealthReport.OsdAnalysis == nil || cephDeploymentHealth.Status.HealthReport.OsdAnalysis.CephClusterSpecGeneration == nil {
		msg := "related CephDeploymentHealth has no CephCluster osd storage analysis yet"
		sublog.Warn().Msg(msg)
		cephTask.Status.PhaseInfo = msg
		err = r.updateCephOsdRemoveTaskStatus(ctx, request, cephTask.Status)
		if err != nil {
			sublog.Error().Err(err).Msg("")
		}
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}

	taskList, err := r.Lcmclientset.LcmV1alpha1().CephOsdRemoveTasks(request.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		sublog.Error().Err(err).Msg("")
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}
	// check that we are picking up first created not closed task, to avoid race between multiple tasks in ns
	if oldestTaskName := getOldestCephOsdRemoveTaskName(taskList.Items); oldestTaskName != request.Name {
		sublog.Info().Msgf("paused, found older not completed CephOsdRemoveTask '%s/%s", request.Namespace, oldestTaskName)
		cephTask.Status.PhaseInfo = "waiting for older CephOsdRemoveTask completion"
		err = r.updateCephOsdRemoveTaskStatus(ctx, request, cephTask.Status)
		if err != nil {
			sublog.Error().Err(err).Msg("")
		}
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}

	removeConfig := &cephOsdRemoveConfig{
		context:   ctx,
		api:       r,
		log:       &sublog,
		lcmConfig: &lcmConfig,
		taskConfig: taskConfig{
			task:                  cephTask,
			cephCluster:           cephCluster,
			cephHealthOsdAnalysis: cephDeploymentHealth.Status.HealthReport.OsdAnalysis,
		},
	}

	newStatus := removeConfig.handleTask()
	if !reflect.DeepEqual(newStatus, cephTask.Status) {
		err = r.updateCephOsdRemoveTaskStatus(ctx, request, newStatus)
		if err != nil {
			sublog.Error().Err(err).Msg("")
			return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
		}
		if !checkTaskActive(newStatus) {
			sublog.Info().Msg("finished processing")
			return reconcile.Result{}, nil
		}
	}
	if removeConfig.taskConfig.requeueNow {
		return reconcile.Result{Requeue: true}, nil
	}
	sublog.Info().Msg("processing is not finished yet")
	return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
}

func (r *ReconcileCephOsdRemoveTask) updateCephOsdRemoveTaskStatus(ctx context.Context, req reconcile.Request, status *lcmv1alpha1.CephOsdRemoveTaskStatus) error {
	cephTask := &lcmv1alpha1.CephOsdRemoveTask{}
	err := r.Client.Get(ctx, req.NamespacedName, cephTask)
	if err != nil {
		return errors.Wrapf(err, "failed to get CephOsdRemoveTask '%s' to update status", req.NamespacedName)
	}
	err = lcmv1alpha1.UpdateCephOsdRemoveTaskStatus(cephTask, status, r.Client)
	if err != nil {
		return errors.Wrapf(err, "failed to update CephOsdRemoveTask '%s' status with '%v' phase", req.NamespacedName, status.Phase)
	}
	return nil
}
