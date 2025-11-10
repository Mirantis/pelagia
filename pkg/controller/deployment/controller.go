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
	"reflect"
	"strings"

	"github.com/rs/zerolog"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmclient "github.com/Mirantis/pelagia/pkg/client/clientset/versioned"

	"github.com/google/go-cmp/cmp"
	claimClient "github.com/kube-object-storage/lib-bucket-provisioner/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	rookclient "github.com/rook/rook/pkg/client/clientset/versioned"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	lcmconfig "github.com/Mirantis/pelagia/pkg/controller/config"
)

const (
	ControllerName = "pelagia-deployment-controller"
	finalizer      = "cephdeployment.lcm.mirantis.com/finalizer"
)

// Add creates a new CephDeployment Controller and adds it to the Manager. The Manager will set fields on the Controller
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
	CephLcmclientset, _ := lcmclient.NewForConfig(config)
	ClaimClientset, _ := claimClient.NewForConfig(config)
	kubeclientset, _ := kubernetes.NewForConfig(config)

	return &ReconcileCephDeployment{
		Config:           config,
		Client:           mgr.GetClient(),
		Kubeclientset:    kubeclientset,
		Rookclientset:    RookClientset,
		CephLcmclientset: CephLcmclientset,
		Claimclientset:   ClaimClientset,
		Scheme:           mgr.GetScheme(),
	}
}

func cephDplPredicate[T *cephlcmv1alpha1.CephDeployment]() predicate.TypedFuncs[T] {
	return predicate.TypedFuncs[T]{
		UpdateFunc: func(e event.TypedUpdateEvent[T]) bool {
			resourceQtyComparer := cmp.Comparer(func(x, y resource.Quantity) bool { return x.Cmp(y) == 0 })

			oldObject := (*cephlcmv1alpha1.CephDeployment)(e.ObjectOld)
			newObject := (*cephlcmv1alpha1.CephDeployment)(e.ObjectNew)

			diff := cmp.Diff(oldObject.Spec, newObject.Spec, resourceQtyComparer)
			if diff != "" {
				log.Info().Str(lcmcommon.LoggerObjectField, fmt.Sprintf("cephdeployment '%s/%s'", oldObject.Namespace, oldObject.Name)).Msgf("spec has changed for %q. diff=%s", newObject.Name, diff)
				return true
			}
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

	// Watch for changes to primary resource CephDeployment
	err = c.Watch(source.Kind(
		mgr.GetCache(),
		&cephlcmv1alpha1.CephDeployment{},
		&handler.TypedEnqueueRequestForObject[*cephlcmv1alpha1.CephDeployment]{},
		cephDplPredicate[*cephlcmv1alpha1.CephDeployment]()))
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileCephDeployment implements reconcile.Reconciler
var (
	_   reconcile.Reconciler = &ReconcileCephDeployment{}
	log                      = lcmcommon.InitLogger(true)
)

// ReconcileCephDeployment reconciles a CephDeployment object
type ReconcileCephDeployment struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	Client           client.Client
	Kubeclientset    kubernetes.Interface
	Rookclientset    rookclient.Interface
	CephLcmclientset lcmclient.Interface
	Claimclientset   claimClient.Interface
	Scheme           *runtime.Scheme
	Config           *rest.Config
}

func (r *ReconcileCephDeployment) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	lcmConfig := lcmconfig.GetConfiguration(request.Namespace)
	sublog := log.With().Str(lcmcommon.LoggerObjectField, fmt.Sprintf("cephdeployment '%v'", request.NamespacedName)).Logger().Level(lcmConfig.DeployParams.LogLevel)
	sublog.Debug().Msg("reconcile started")
	cephDplList, err := r.CephLcmclientset.LcmV1alpha1().CephDeployments(request.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		sublog.Error().Err(err).Msgf("failed to list CephDeployments %s namespace", request.Namespace)
		return reconcile.Result{RequeueAfter: requeueAfterInterval},
			errors.Wrapf(err, "failed to list CephDeployments %s namespace", request.Namespace)
	}
	if len(cephDplList.Items) == 0 {
		return reconcile.Result{}, nil
	}
	if len(cephDplList.Items) > 1 {
		msg := fmt.Sprintf("incorrect number of CephDeployments in %s namespace", request.Namespace)
		sublog.Error().Msg(msg)
		r.setCephDeploymentPhaseFailed(ctx, sublog, request.Name, request.Namespace, cephlcmv1alpha1.CephDeploymentStatus{Phase: cephlcmv1alpha1.PhaseFailed, Message: msg})
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}
	cephDpl := &cephDplList.Items[0]
	if cephDpl.Name != request.Name {
		sublog.Error().Msgf("incorrect CephDeployment %s/%s, expected %s/%s", cephDpl.Namespace, cephDpl.Name, request.Namespace, request.Name)
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, errors.New("incorrect CephDeployment object")
	}

	cephDplConfig := &cephDeploymentConfig{
		context:   ctx,
		api:       r,
		log:       &sublog,
		lcmConfig: &lcmConfig,
		cdConfig:  deployConfig{cephDpl: cephDpl},
	}

	// check first deprecated fields
	err = cephDplConfig.ensureDeprecatedFields()
	if err != nil {
		cephDpl.Status.Phase = cephlcmv1alpha1.PhaseFailed
		cephDpl.Status.Message = fmt.Sprintf("failed to ensure deprecated fields for CephDeployment %s/%s", cephDpl.Namespace, cephDpl.Name)
		sublog.Error().Err(err).Msg(cephDpl.Status.Message)
		r.setCephDeploymentPhaseFailed(ctx, sublog, cephDpl.Name, cephDpl.Namespace, cephDpl.Status)
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}

	expandedNodes, err := lcmcommon.GetExpandedCephDeploymentNodeList(ctx, r.Client, cephDpl.Spec)
	if err != nil {
		cephDpl.Status.Phase = cephlcmv1alpha1.PhaseFailed
		cephDpl.Status.Message = fmt.Sprintf("failed to expand node list for CephDeployment %s/%s", cephDpl.Namespace, cephDpl.Name)
		sublog.Error().Err(err).Msg(cephDpl.Status.Message)
		r.setCephDeploymentPhaseFailed(ctx, sublog, cephDpl.Name, cephDpl.Namespace, cephDpl.Status)
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}
	cephDplConfig.cdConfig.nodesListExpanded = expandedNodes

	// Is cephDpl going to delete? If yes, delete cephcluster before
	if cephDpl.GetDeletionTimestamp() != nil {
		if cephDpl.Spec.ExtraOpts == nil || !cephDpl.Spec.ExtraOpts.PreventClusterDestroy {
			if lcmcommon.Contains(cephDpl.GetFinalizers(), finalizer) {
				delMsgOk := "Ceph cluster deletion is in progress"
				if cephDpl.Status.Phase != cephlcmv1alpha1.PhaseDeleting {
					sublog.Info().Msgf("switching to remove phase for CephDeployment %s/%s", cephDpl.Namespace, cephDpl.Name)
					cephDpl.Status.Phase = cephlcmv1alpha1.PhaseDeleting
					cephDpl.Status.Message = delMsgOk
					err = r.updateCephDeploymentStatus(ctx, sublog, cephDpl.Name, cephDpl.Namespace, cephDpl.Status)
					if err != nil {
						sublog.Error().Err(err).Msgf("failed to update CephDeployment %s/%s status", cephDpl.Namespace, cephDpl.Name)
						return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
					}
					return reconcile.Result{Requeue: true}, nil
				}
				deleted, delErr := cephDplConfig.cleanCephDeployment()
				if delErr != nil {
					sublog.Error().Err(delErr).Msgf("failed to delete CephDeployment %s/%s", cephDpl.Namespace, cephDpl.Name)
					cephDpl.Status.Message = "Ceph cluster is failing to remove"
				} else {
					if deleted {
						sublog.Info().Msgf("Finished CephDeployment resource cleanup for %s/%s", cephDpl.Namespace, cephDpl.Name)
						// Remove finalizer. Once all finalizers have been removed, the object will be deleted.
						if cephDplConfig.updateFinalizer(false) == nil {
							return reconcile.Result{}, nil
						}
						cephDpl.Status.Message = "Ceph cluster is removed, failed to cleanup CephDeployment"
					} else {
						cephDpl.Status.Message = delMsgOk
					}
				}
				err = r.updateCephDeploymentStatus(ctx, sublog, cephDpl.Name, cephDpl.Namespace, cephDpl.Status)
				if err != nil {
					sublog.Error().Err(err).Msgf("failed to update CephDeployment %s/%s status", cephDpl.Namespace, cephDpl.Name)
					return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
				}
				return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
			}
			return reconcile.Result{}, nil
		}
		sublog.Warn().Msgf("looks like CephDeployment '%s/%s' is going to be removed, but spec option preventClusterDestroy is set to true",
			cephDpl.Namespace, cephDpl.Name)
	}

	// run spec validation every time and requeue reconcile if updated
	sublog.Debug().Msgf("running validation of CephDeployment '%s/%s' spec", cephDpl.Namespace, cephDpl.Name)
	validationResult := cephDplConfig.validate()
	if !reflect.DeepEqual(cephDpl.Status.Validation, validationResult) {
		cephDpl.Status.Validation = validationResult
		if validationResult.Result == cephlcmv1alpha1.ValidationFailed {
			cephDpl.Status.Phase = cephlcmv1alpha1.PhaseFailed
		}
		err = r.updateCephDeploymentStatus(ctx, sublog, cephDpl.Name, cephDpl.Namespace, cephDpl.Status)
		if err != nil {
			sublog.Error().Err(err).Msg("failed to write CephDeployment status")
			return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
		}
		return reconcile.Result{Requeue: true}, nil
	}
	if cephDpl.Status.Validation.Result == cephlcmv1alpha1.ValidationFailed {
		sublog.Error().Msgf("validation of CephDeployment spec is failed, fix it first: %s", strings.Join(cephDpl.Status.Validation.Messages, ","))
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}

	// Add finalizer if there is none in instance
	if !lcmcommon.Contains(cephDpl.GetFinalizers(), finalizer) {
		if cephDplConfig.updateFinalizer(true) != nil {
			return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
		}
		return reconcile.Result{Requeue: true}, nil
	}

	cephRuntimeVersion, cephImageToUse, cephStatusVersion, err := cephDplConfig.verifyCephVersions()
	if err != nil {
		sublog.Error().Err(err).Msg("failed to verify Ceph version")
		cephDpl.Status.Phase = cephlcmv1alpha1.PhaseFailed
		cephDpl.Status.Message = err.Error()
		r.setCephDeploymentPhaseFailed(ctx, sublog, cephDpl.Name, cephDpl.Namespace, cephDpl.Status)
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}
	cephDplConfig.cdConfig.currentCephVersion = cephRuntimeVersion
	if cephDpl.Status.ClusterVersion != cephStatusVersion {
		cephDpl.Status.ClusterVersion = cephStatusVersion
		err = r.updateCephDeploymentStatus(ctx, sublog, cephDpl.Name, cephDpl.Namespace, cephDpl.Status)
		if err != nil {
			sublog.Error().Err(err).Msg("failed to update CephDeployment cluster version status")
			return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
		}
		return reconcile.Result{Requeue: true}, nil
	}
	cephDplConfig.cdConfig.currentCephImage = cephImageToUse

	objRefs, objRefsErr := cephDplConfig.createSubObjects()
	if objRefsErr != "" {
		sublog.Error().Msg("failed to create sub resources")
		cephDpl.Status.Phase = cephlcmv1alpha1.PhaseFailed
		cephDpl.Status.Message = fmt.Sprintf("Ceph cluster %s", objRefsErr)
		r.setCephDeploymentPhaseFailed(ctx, sublog, cephDpl.Name, cephDpl.Namespace, cephDpl.Status)
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}
	cephDpl.Status.ObjectsRefs = objRefs

	err = cephDplConfig.verifySetup()
	if err != nil {
		sublog.Error().Err(err).Msg("failed to verify Ceph setup")
		cephDpl.Status.Phase = cephlcmv1alpha1.PhaseFailed
		cephDpl.Status.Message = err.Error()
		r.setCephDeploymentPhaseFailed(ctx, sublog, cephDpl.Name, cephDpl.Namespace, cephDpl.Status)
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}

	lcmPhaseActive, lcmPhase, err := cephDplConfig.checkLcmState()
	if err != nil {
		sublog.Error().Err(err).Msg("failed to check lifecycle state")
		cephDpl.Status.Phase = cephlcmv1alpha1.PhaseFailed
		cephDpl.Status.Message = err.Error()
		r.setCephDeploymentPhaseFailed(ctx, sublog, cephDpl.Name, cephDpl.Namespace, cephDpl.Status)
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}
	// update status if needed and wait
	if lcmPhaseActive {
		if lcmPhase != cephDpl.Status.Phase {
			cephDpl.Status.Phase = lcmPhase
			switch lcmPhase {
			case cephlcmv1alpha1.PhaseOnHold:
				cephDpl.Status.Message = "Ceph cluster is under request processing"
			case cephlcmv1alpha1.PhaseMaintenance:
				cephDpl.Status.Message = "Cluster maintenance (upgrade) detected, reconcile is paused"
			}
			err = r.updateCephDeploymentStatus(ctx, sublog, cephDpl.Name, cephDpl.Namespace, cephDpl.Status)
			if err != nil {
				sublog.Error().Err(err).Msg("failed to write CephDeployment status")
			}
		}
		return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
	}

	// do not set deploying phase if current one ready - set validation phase
	// setting deploying state means definitely some ops required
	// and previous validation phase was not finished/completed
	// if previous phase deploying - keep status as is till the next update
	// after configuration apply and expose it to operator
	if cephDpl.Status.Phase == cephlcmv1alpha1.PhaseReady {
		cephDpl.Status.Phase = cephlcmv1alpha1.PhaseValidation
		cephDpl.Status.Message = "Ceph cluster configuration is verifying"
	} else if cephDpl.Status.Phase != cephlcmv1alpha1.PhaseDeploying {
		cephDpl.Status.Phase = cephlcmv1alpha1.PhaseDeploying
		cephDpl.Status.Message = "Ceph cluster is deploying"
	}
	err = r.updateCephDeploymentStatus(ctx, sublog, cephDpl.Name, cephDpl.Namespace, cephDpl.Status)
	if err != nil {
		sublog.Error().Err(err).Msg("failed to write CephDeployment status")
	}
	applyInProgress, applyFailed := cephDplConfig.applyConfiguration()
	if applyInProgress != "" || applyFailed != "" || objRefsErr != "" {
		cephDpl.Status.Phase = cephlcmv1alpha1.PhaseDeploying
		msgs := []string{}
		if objRefsErr != "" {
			msgs = append(msgs, objRefsErr)
		}
		if applyInProgress != "" {
			msgs = append(msgs, applyInProgress)
		}
		if applyFailed != "" {
			msgs = append(msgs, applyFailed)
		}
		cephDpl.Status.Message = fmt.Sprintf("Ceph cluster %s", strings.Join(msgs, "; "))
	} else {
		cephDpl.Status.Phase = cephlcmv1alpha1.PhaseReady
		cephDpl.Status.Message = "Ceph cluster configuration successfully applied"
	}
	err = r.updateCephDeploymentStatus(ctx, sublog, cephDpl.Name, cephDpl.Namespace, cephDpl.Status)
	if err != nil {
		sublog.Error().Err(err).Msg("failed to write CephDeployment status")
	}
	sublog.Debug().Msgf("reconcile for CephDeployment %q is finished", request.String())
	return reconcile.Result{RequeueAfter: requeueAfterInterval}, nil
}

func (c *cephDeploymentConfig) createSubObjects() ([]v1.ObjectReference, string) {
	ownerRefs, err := lcmcommon.GetObjectOwnerRef(c.cdConfig.cephDpl, c.api.Scheme)
	if err != nil {
		msg := fmt.Sprintf("failed to prepare ownerRefs for CephDeployment %s/%s related child resources", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
		c.log.Error().Err(err).Msg(msg)
		return nil, msg
	}

	getRef := func(name, namespace, kind string) v1.ObjectReference {
		return v1.ObjectReference{
			APIVersion: cephlcmv1alpha1.SchemeGroupVersion.String(),
			Kind:       kind,
			Name:       name,
			Namespace:  namespace,
		}
	}

	refs := []v1.ObjectReference{}
	issues := []string{}
	_, err = c.api.CephLcmclientset.LcmV1alpha1().CephDeploymentHealths(c.cdConfig.cephDpl.Namespace).Get(c.context, c.cdConfig.cephDpl.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			cephhealth := &cephlcmv1alpha1.CephDeploymentHealth{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       c.cdConfig.cephDpl.Namespace,
					Name:            c.cdConfig.cephDpl.Name,
					OwnerReferences: ownerRefs,
				},
			}
			_, err = c.api.CephLcmclientset.LcmV1alpha1().CephDeploymentHealths(cephhealth.Namespace).Create(c.context, cephhealth, metav1.CreateOptions{})
			if err != nil {
				msg := "failed to create CephDeploymentHealth"
				c.log.Error().Err(err).Msg(msg)
				issues = append(issues, msg)
			} else {
				c.log.Info().Msgf("Created CephDeploymentHealth %s/%s resource", cephhealth.Namespace, cephhealth.Name)
				refs = append(refs, getRef(cephhealth.Name, cephhealth.Namespace, "CephDeploymentHealth"))
			}
		} else {
			msg := "failed to check CephDeploymentHealth presence"
			c.log.Error().Err(err).Msg(msg)
			issues = append(issues, msg)
		}
	} else {
		refs = append(refs, getRef(c.cdConfig.cephDpl.Name, c.cdConfig.cephDpl.Namespace, "CephDeploymentHealth"))
	}

	_, err = c.api.CephLcmclientset.LcmV1alpha1().CephDeploymentSecrets(c.cdConfig.cephDpl.Namespace).Get(c.context, c.cdConfig.cephDpl.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			cephdeploymentsecrets := &cephlcmv1alpha1.CephDeploymentSecret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       c.cdConfig.cephDpl.Namespace,
					Name:            c.cdConfig.cephDpl.Name,
					OwnerReferences: ownerRefs,
				},
			}
			_, err = c.api.CephLcmclientset.LcmV1alpha1().CephDeploymentSecrets(cephdeploymentsecrets.Namespace).Create(c.context, cephdeploymentsecrets, metav1.CreateOptions{})
			if err != nil {
				msg := "failed to create CephDeploymentSecret"
				c.log.Error().Err(err).Msg(msg)
				issues = append(issues, msg)
			} else {
				c.log.Info().Msgf("Created CephDeploymentSecret %s/%s resource", cephdeploymentsecrets.Namespace, cephdeploymentsecrets.Name)
				refs = append(refs, getRef(cephdeploymentsecrets.Name, cephdeploymentsecrets.Namespace, "CephDeploymentSecret"))
			}
		} else {
			msg := "failed to check CephDeploymentSecret presence"
			c.log.Error().Err(err).Msg(msg)
			issues = append(issues, msg)
		}
	} else {
		refs = append(refs, getRef(c.cdConfig.cephDpl.Name, c.cdConfig.cephDpl.Namespace, "CephDeploymentSecret"))
	}

	_, err = c.api.CephLcmclientset.LcmV1alpha1().CephDeploymentMaintenances(c.cdConfig.cephDpl.Namespace).Get(c.context, c.cdConfig.cephDpl.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			cephDplMaintenance := &cephlcmv1alpha1.CephDeploymentMaintenance{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       c.cdConfig.cephDpl.Namespace,
					Name:            c.cdConfig.cephDpl.Name,
					OwnerReferences: ownerRefs,
				},
			}
			_, err = c.api.CephLcmclientset.LcmV1alpha1().CephDeploymentMaintenances(cephDplMaintenance.Namespace).Create(c.context, cephDplMaintenance, metav1.CreateOptions{})
			if err != nil {
				msg := "failed to create CephDeploymentMaintenance"
				c.log.Error().Err(err).Msg(msg)
				issues = append(issues, msg)
			} else {
				c.log.Info().Msgf("Created CephDeploymentMaintenance %s/%s resource", cephDplMaintenance.Namespace, cephDplMaintenance.Name)
				refs = append(refs, getRef(cephDplMaintenance.Name, cephDplMaintenance.Namespace, "CephDeploymentMaintenance"))
			}
		} else {
			msg := "failed to check CephDeploymentMaintenance presence"
			c.log.Error().Err(err).Msg(msg)
			issues = append(issues, msg)
		}
	} else {
		refs = append(refs, getRef(c.cdConfig.cephDpl.Name, c.cdConfig.cephDpl.Namespace, "CephDeploymentMaintenance"))
	}
	return refs, strings.Join(issues, ", ")
}

func (c *cephDeploymentConfig) verifySetup() error {
	// ensure daemonset node labels
	c.ensureDaemonsetLabels()

	// check to avoid further actions if no current ceph version detected
	// the only case is when ceph-tools was broken, but ceph cluster is ok
	// so just throw error and re-run reconcile - since ensure ceph-tools
	// is passed to that moment, next reconcile should set required var.
	if c.cdConfig.currentCephVersion == nil {
		return errors.Errorf("current Ceph version is not detected")
	}

	c.log.Debug().Msgf("running Ceph cluster version %s %s.%s", c.cdConfig.currentCephVersion.Name, c.cdConfig.currentCephVersion.MajorVersion, c.cdConfig.currentCephVersion.MinorVersion)
	// ensure rook image is actual in Rook apps
	err := c.ensureRookImage()
	if err != nil {
		return errors.Wrap(err, "failed to ensure consistent Rook image version")
	}
	// update ceph cluster image if needed, because during
	// usual ensure cluster can be not ready and any change will be skipped
	err = c.ensureCephClusterVersion()
	if err != nil {
		return errors.Wrap(err, "failed to ensure consistent Ceph cluster version")
	}
	return nil
}

func (c *cephDeploymentConfig) checkLcmState() (bool, cephlcmv1alpha1.CephDeploymentPhase, error) {
	if !c.cdConfig.cephDpl.Spec.External {
		c.log.Debug().Msg("ensure CephOsdRemoveTasks")
		taskList, err := c.api.CephLcmclientset.LcmV1alpha1().CephOsdRemoveTasks(c.cdConfig.cephDpl.Namespace).List(c.context, metav1.ListOptions{})
		if err != nil {
			return false, cephlcmv1alpha1.PhaseFailed, errors.Wrapf(err, "failed to list CephOsdRemoveTasks in %s namespace", c.cdConfig.cephDpl.Namespace)
		}
		for _, taskItem := range taskList.Items {
			if taskItem.Status != nil {
				if taskItem.Status.Phase == cephlcmv1alpha1.TaskPhaseValidating {
					c.log.Info().Msgf("found CephOsdRemoveTask '%s/%s' in validation phase, holding reconcile for correct task completion", taskItem.Namespace, taskItem.Name)
					// check spec before, to be sure, ceph cluster spec is updated with latest otherwise contine reconcile
					if c.cdConfig.cephDpl.Status.Phase != cephlcmv1alpha1.PhaseOnHold {
						aligned, err := c.checkStorageSpecIsAligned()
						if err != nil {
							return false, cephlcmv1alpha1.PhaseFailed, errors.Wrap(err, "failed to check Ceph cluster state")
						}
						if !aligned {
							c.log.Info().Msgf("CephOsdRemoveTask '%s/%s' in validation phase, but Ceph cluster storage spec is not aligned with current CephDeployment nodes, continue reconcile",
								taskItem.Namespace, taskItem.Name)
							break
						}
					}
					return true, cephlcmv1alpha1.PhaseOnHold, nil
				} else if taskItem.Status.Phase == cephlcmv1alpha1.TaskPhaseWaitingOperator || taskItem.Status.Phase == cephlcmv1alpha1.TaskPhaseApproveWaiting {
					c.log.Info().Msgf("found CephOsdRemoveTask '%s/%s' in waiting phase, holding reconcile for correct task completion", taskItem.Namespace, taskItem.Name)
					return true, cephlcmv1alpha1.PhaseOnHold, nil
				} else if taskItem.Status.Phase == cephlcmv1alpha1.TaskPhaseProcessing {
					c.log.Info().Msgf("found CephOsdRemoveTask '%s/%s' in processing phase, waiting until it completed", taskItem.Namespace, taskItem.Name)
					return true, cephlcmv1alpha1.PhaseOnHold, nil
				} else if taskItem.Status.Phase == cephlcmv1alpha1.TaskPhaseFailed {
					lastCondition := len(taskItem.Status.Conditions) - 1
					// check phase before current - if processing - then something not ok
					if lastCondition > 0 && taskItem.Status.Conditions[lastCondition-1].Phase == cephlcmv1alpha1.TaskPhaseProcessing {
						if taskItem.Status.Phase == cephlcmv1alpha1.TaskPhaseFailed && taskItem.Spec != nil && taskItem.Spec.Resolved {
							continue
						}
						c.log.Error().Msgf("found CephOsdRemoveTask '%s/%s' in '%s' phase after failed processing. Inspect and remove if not relevant or mark resolved",
							taskItem.Namespace, taskItem.Name, taskItem.Status.Phase)
						return true, cephlcmv1alpha1.PhaseOnHold, nil
					}
				}
			}
		}
	}

	isActing, err := c.isMaintenanceActing()
	if err != nil {
		return false, cephlcmv1alpha1.PhaseFailed, errors.Wrap(err, "failed to check CephDeploymentMaintenance state")
	}

	// if maintenance acting - switch to maintenance
	if isActing {
		c.log.Info().Msg("CephDeploymentMaintenance is acting")
		c.log.Info().Msgf("reconcile CephDeployment %s/%s cancelled, set maintenance mode", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
		return true, cephlcmv1alpha1.PhaseMaintenance, nil
	}

	// doesnot mean that status is actually ready - just marker that lcm actions are not acting
	return false, cephlcmv1alpha1.PhaseReady, nil
}

func (c *cephDeploymentConfig) applyConfiguration() (string, string) {
	errCollector := make([]string, 0)
	changedCollector := make([]string, 0)
	msgTmpl := "failed to ensure"
	// helper func to avoid copy-paste on issue processing
	handleEnsureResult := func(resourceChanged bool, err error, ensureResource string) {
		if err != nil {
			c.log.Error().Err(err).Msgf("%s %s", msgTmpl, ensureResource)
			errCollector = append(errCollector, ensureResource)
		} else {
			if resourceChanged {
				changedCollector = append(changedCollector, ensureResource)
			}
		}
	}
	var err error
	var changed bool
	// Ensure node labels and topology
	if !c.cdConfig.cephDpl.Spec.External {
		changed, err = c.ensureLabelNodes()
		handleEnsureResult(changed, err, "label nodes")
	}

	// ensure nodes annotations if any
	if !c.cdConfig.cephDpl.Spec.External {
		changed, err = c.ensureNodesAnnotation()
		handleEnsureResult(changed, err, "annotate nodes")
	}

	// ensure network policies
	netPoolChanged := false
	if !c.cdConfig.cephDpl.Spec.External {
		netPoolChanged, err = c.ensureNetworkPolicy()
		handleEnsureResult(netPoolChanged, err, "network policies")
	}

	// continue if labeling/netpool are not failed and no netpool changes
	if len(errCollector) == 0 && !netPoolChanged {
		// Ensure ceph cluster processing
		changed, err = c.ensureCluster()
		handleEnsureResult(changed, err, "cephcluster")

		// Ensure ceph block pools processing for non-external cluster
		if !c.cdConfig.cephDpl.Spec.External {
			changed, err = c.ensurePools()
			handleEnsureResult(changed, err, "cephblockpools")
		}

		// Ensure shared filesystems (CephFS) for non-external cluster
		if !c.cdConfig.cephDpl.Spec.External {
			changed, err = c.ensureSharedFilesystem()
			handleEnsureResult(changed, err, "shared filesystems")
		}

		// Ensure storage classes of ceph pools
		changed, err = c.ensureStorageClasses()
		handleEnsureResult(changed, err, "storageclasses")

		// Ensure ceph clients processing
		changed, err = c.ensureCephClients()
		handleEnsureResult(changed, err, "cephclients")

		// Ensure ceph object storage processing
		changed, err = c.ensureObjectStorage()
		handleEnsureResult(changed, err, "ceph object storage")

		// Ensure RBD Mirror processing
		changed, err = c.ensureRBDMirroring()
		handleEnsureResult(changed, err, "RBD Mirroring")

		// Ensure openstack shared secret processing for non-external cluster
		if !c.cdConfig.cephDpl.Spec.External {
			changed, err = c.ensureOpenstackSecret()
			handleEnsureResult(changed, err, "Openstack secret")
		}

		// Ensure Ingress proxy for non-external
		if !c.cdConfig.cephDpl.Spec.External {
			changed, err = c.ensureIngressProxy()
			handleEnsureResult(changed, err, "ingress proxy")
		}

		// Ensure overal cluster state
		if !c.cdConfig.cephDpl.Spec.External {
			changed, err = c.ensureClusterState()
			handleEnsureResult(changed, err, "cluster state")
		}
	}

	applyRes := ""
	if len(changedCollector) > 0 {
		applyRes = fmt.Sprintf("configuration apply is in progress: %s", strings.Join(changedCollector, ", "))
		c.log.Info().Msg(applyRes)
	}
	errRes := ""
	if len(errCollector) > 0 {
		errRes = fmt.Sprintf("configuration apply is failed: %s %s", msgTmpl, strings.Join(errCollector, ", "))
		c.log.Error().Msg(errRes)
	}
	return applyRes, errRes
}

func (c *cephDeploymentConfig) updateFinalizer(add bool) error {
	op := "add"
	if add {
		c.log.Info().Msgf("adding Finalizer for CephDeployment %s/%s", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
		controllerutil.AddFinalizer(c.cdConfig.cephDpl, finalizer)
	} else {
		op = "remove"
		c.log.Info().Msgf("removing finalizer for CephDeployment %s/%s", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
		controllerutil.RemoveFinalizer(c.cdConfig.cephDpl, finalizer)
	}

	// Update CR
	_, err := c.api.CephLcmclientset.LcmV1alpha1().CephDeployments(c.cdConfig.cephDpl.Namespace).Update(c.context, c.cdConfig.cephDpl, metav1.UpdateOptions{})
	if err != nil {
		c.log.Error().Err(err).Msgf("failed to %s finalizer for CephDeployment %s/%s", op, c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
	}
	return err
}

func (r *ReconcileCephDeployment) setCephDeploymentPhaseFailed(ctx context.Context, log zerolog.Logger, name, namespace string, status cephlcmv1alpha1.CephDeploymentStatus) {
	if currentFailTry < failTriesLeft {
		currentFailTry++
	} else {
		err := r.updateCephDeploymentStatus(ctx, log, name, namespace, status)
		if err != nil {
			log.Error().Err(err).Msgf("failed to update CephDeployment %s/%s status", namespace, name)
		}
	}
}

func (r *ReconcileCephDeployment) updateCephDeploymentStatus(ctx context.Context, log zerolog.Logger, name, namespace string, status cephlcmv1alpha1.CephDeploymentStatus) error {
	currentFailTry = 0
	cephDpl := &cephlcmv1alpha1.CephDeployment{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, cephDpl)
	if err != nil {
		return errors.Wrapf(err, "failed to get CephDeployment '%s/%s' to update status", namespace, name)
	}
	// do not allow change status to failed if already set maintenance or request processing to avoid
	// issues with related controllers, because they are relying on corresponding .status.phase
	// just update timestamp
	logMsg := fmt.Sprintf("updating status for CephDeployment %s/%s", cephDpl.Namespace, cephDpl.Name)
	if status.Phase == cephlcmv1alpha1.PhaseFailed && (cephDpl.Status.Phase == cephlcmv1alpha1.PhaseOnHold || cephDpl.Status.Phase == cephlcmv1alpha1.PhaseMaintenance) {
		status = cephDpl.Status
	}
	if status.ClusterVersion != "" {
		logMsg = fmt.Sprintf("%s, cluster version '%s'", logMsg, status.ClusterVersion)
	}
	if status.Phase != "" {
		logMsg = fmt.Sprintf("%s, set phase: %v", logMsg, status.Phase)
	}
	status.LastRun = lcmcommon.GetCurrentTimeString()
	log.Info().Msg(logMsg)
	err = cephlcmv1alpha1.UpdateCephDeploymentStatus(ctx, cephDpl, status, r.Client)
	if err != nil {
		return errors.Wrapf(err, "failed to update CephDeployment %s/%s status", cephDpl.Namespace, cephDpl.Name)
	}
	return nil
}

func (c *cephDeploymentConfig) cleanCephDeployment() (bool, error) {
	cleanupFinished := true
	errCollector := make([]string, 0)

	runRemoveState := func(step string, removeStep func() (bool, error)) {
		removed, err := removeStep()
		if err != nil {
			errMsg := fmt.Sprintf("failed to remove %s", step)
			c.log.Error().Err(err).Msg(errMsg)
			errCollector = append(errCollector, errMsg)
			cleanupFinished = false
		} else {
			if !removed {
				c.log.Info().Msgf("deletion of %s is in progress", step)
			}
			cleanupFinished = cleanupFinished && removed
		}
	}

	c.log.Info().Msgf("deleting resources for CephDeployment %s/%s", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name)
	// Delete cephdeploymentsecret to avoid not needed secret reconcilation
	runRemoveState(fmt.Sprintf("CephDeploymentSecret '%s/%s'", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name), func() (bool, error) {
		err := c.api.CephLcmclientset.LcmV1alpha1().CephDeploymentSecrets(c.cdConfig.cephDpl.Namespace).Delete(c.context, c.cdConfig.cephDpl.Name, metav1.DeleteOptions{})
		if err != nil && apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	// Delete maintenance crd object to avoid not needed secret reconcilation
	runRemoveState(fmt.Sprintf("CephDeploymentMaintenance '%s/%s'", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name), func() (bool, error) {
		err := c.api.CephLcmclientset.LcmV1alpha1().CephDeploymentMaintenances(c.cdConfig.cephDpl.Namespace).Delete(c.context, c.cdConfig.cephDpl.Name, metav1.DeleteOptions{})
		if err != nil && apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	// Delete openstack secret
	if c.cdConfig.cephDpl.Spec.ExtraOpts != nil && c.cdConfig.cephDpl.Spec.ExtraOpts.DisableOsKeys {
		c.log.Warn().Msgf("openstack secret %s/%s ensure disabled, skip deleting. Do not forget to remove it manually",
			c.lcmConfig.DeployParams.OpenstackCephSharedNamespace, openstackSharedSecret)
	} else {
		runRemoveState("openstack shared secret", func() (bool, error) {
			return c.deleteOpenstackSecret()
		})
	}
	// Delete object storage stuff
	runRemoveState("object storage", func() (bool, error) {
		return c.deleteObjectStorage()
	})
	if !c.cdConfig.cephDpl.Spec.External {
		// Delete ingress proxy
		runRemoveState("ingress proxy", func() (bool, error) {
			return c.deleteIngressProxy()
		})
	}
	// Delete RBD Mirror
	runRemoveState("rbd mirror", func() (bool, error) {
		return c.deleteRBDMirroring()
	})
	// Delete ceph clients
	runRemoveState("ceph clients", func() (bool, error) {
		return c.deleteCephClients()
	})
	if !c.cdConfig.cephDpl.Spec.External {
		// Delete ceph block pools
		runRemoveState("ceph block pools", func() (bool, error) {
			return c.deletePools()
		})

		// Delete ceph shared filesystems
		runRemoveState("ceph shared filesystem", func() (bool, error) {
			return c.deleteSharedFilesystems()
		})
	}
	// Delete storage classes
	runRemoveState("storage classes", func() (bool, error) {
		return c.deleteStorageClasses()
	})
	if c.cdConfig.cephDpl.Spec.External {
		runRemoveState("external resources", func() (bool, error) {
			return c.deleteExternalConnectionSecret()
		})
	}

	// if all extra resources removed, continue to main cluster resources cleanup
	if cleanupFinished {
		// Delete cephdeploymenthealth to avoid not needed secret reconcilation
		runRemoveState(fmt.Sprintf("CephDeploymentHealth '%s/%s'", c.cdConfig.cephDpl.Namespace, c.cdConfig.cephDpl.Name), func() (bool, error) {
			err := c.api.CephLcmclientset.LcmV1alpha1().CephDeploymentHealths(c.cdConfig.cephDpl.Namespace).Delete(c.context, c.cdConfig.cephDpl.Name, metav1.DeleteOptions{})
			if err != nil && apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		})
		// Delete ceph cluster processing
		runRemoveState("ceph cluster", func() (bool, error) {
			return c.deleteCluster()
		})
		if !c.cdConfig.cephDpl.Spec.External {
			// Delete network policies
			runRemoveState("network policies", func() (bool, error) {
				return c.cleanupNetworkPolicy()
			})
			// Delete labels from nodes
			runRemoveState("node ceph labels", func() (bool, error) {
				return c.deleteLabelNodes()
			})
			// Delete annotations from nodes
			runRemoveState("node ceph annotations", func() (bool, error) {
				return c.deleteNodesAnnotations()
			})
		}
		runRemoveState("daemonset ceph labels", func() (bool, error) {
			return c.deleteDaemonSetLabels()
		})
	}

	if len(errCollector) > 0 {
		return false, errors.Errorf("deletion is not completed for CephDeployment: %s", strings.Join(errCollector, ", "))
	}
	return cleanupFinished, nil
}
