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

package lcmconfig

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/resource"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

const (
	ControllerName   = "pelagia-config-controller"
	LcmConfigMapName = "pelagia-lcmconfig"
)

// Add creates a new Health Config Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCephDeploymentHealthConfig{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}
}

func cmPredicate[T *corev1.ConfigMap]() predicate.TypedFuncs[T] {
	return predicate.TypedFuncs[T]{
		CreateFunc: func(e event.TypedCreateEvent[T]) bool {
			cm := (*corev1.ConfigMap)(e.Object)
			return cm.Name == LcmConfigMapName
		},
		UpdateFunc: func(e event.TypedUpdateEvent[T]) bool {
			newC := (*corev1.ConfigMap)(e.ObjectNew)
			oldC := (*corev1.ConfigMap)(e.ObjectOld)
			if newC.Name == LcmConfigMapName && oldC.Name == LcmConfigMapName {
				resourceQtyComparer := cmp.Comparer(func(x, y resource.Quantity) bool { return x.Cmp(y) == 0 })

				diff := cmp.Diff(oldC.Data, newC.Data, resourceQtyComparer)
				if diff != "" {
					log.Info().Str(lcmcommon.LoggerObjectField, fmt.Sprintf("configmap '%s/%s'", newC.Namespace, newC.Name)).Msgf("detected changes:\n%s", diff)
					return true
				}
			}
			return false
		},
		DeleteFunc: func(e event.TypedDeleteEvent[T]) bool {
			cm := (*corev1.ConfigMap)(e.Object)
			return cm.Name == LcmConfigMapName
		},
		GenericFunc: func(_ event.TypedGenericEvent[T]) bool {
			return false
		},
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(source.Kind(
		mgr.GetCache(),
		&corev1.ConfigMap{},
		&handler.TypedEnqueueRequestForObject[*corev1.ConfigMap]{},
		cmPredicate()))
	if err != nil {
		return err
	}

	return nil
}

var (
	_ reconcile.Reconciler = &ReconcileCephDeploymentHealthConfig{}
	// init logger
	log = lcmcommon.InitLogger(true)
)

type ReconcileCephDeploymentHealthConfig struct {
	Client client.Client
	Scheme *runtime.Scheme
}

func (r *ReconcileCephDeploymentHealthConfig) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	sublog := log.With().Str(lcmcommon.LoggerObjectField, fmt.Sprintf("configmap '%v'", request.NamespacedName)).Logger()
	sublog.Info().Msgf("loading %v configuration", ParamsToControl)
	lcmConfig := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, request.NamespacedName, lcmConfig)
	if err != nil {
		if apierrors.IsNotFound(err) {
			sublog.Warn().Msg("not found, looks like it's removed, default configuration will be used instead")
			dropConfiguration(request.Namespace)
			return reconcile.Result{}, nil
		}
		sublog.Error().Err(err).Msg("")
		return reconcile.Result{RequeueAfter: 15 * time.Second}, err
	}
	loadConfiguration(sublog, request.Namespace, lcmConfig.Data)
	return reconcile.Result{}, nil
}
