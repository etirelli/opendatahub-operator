/*
Copyright 2023.

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

package datasciencecluster

import (
	"context"

	"github.com/opendatahub-io/opendatahub-operator/components/profiles"

	corev1 "k8s.io/api/core/v1"
	authv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/go-logr/logr"
	addonv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	operators "github.com/operator-framework/api/pkg/operators/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dsc "github.com/opendatahub-io/opendatahub-operator/apis/datasciencecluster/v1alpha1"
	"github.com/opendatahub-io/opendatahub-operator/controllers/status"
	"github.com/opendatahub-io/opendatahub-operator/pkg/deploy"
)

const (
	finalizer                 = "dsc-finalizer.operator.opendatahub.io"
	finalizerMaxRetries       = 10
	odhGeneratedResourceLabel = "opendatahub.io/generated-resource" // hardcoded for now, but we can make it configurable later
)

// DataScienceClusterReconciler reconciles a DataScienceCluster object
type DataScienceClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
	// Recorder to generate events
	Recorder              record.EventRecorder
	ApplicationsNamespace string
}

//+kubebuilder:rbac:groups=datasciencecluster.opendatahub.io,resources=datascienceclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=datasciencecluster.opendatahub.io,resources=datascienceclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=datasciencecluster.opendatahub.io,resources=datascienceclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=services;namespaces;serviceaccounts;secrets;configmaps;operatorgroups,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=addons.managed.openshift.io,resources=addons,verbs=get;list
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings;roles;clusterrolebindings;clusterroles,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DataScienceClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Reconciling DataScienceCluster resources", "Request.Namespace", req.Namespace, "Request.Name", req.Name)

	instance := &dsc.DataScienceCluster{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil && apierrs.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// Start reconciling
	if instance.Status.Conditions == nil {
		reason := status.ReconcileInit
		message := "Initializing DataScienceCluster resource"
		status.SetProgressingCondition(&instance.Status.Conditions, reason, message)
		instance.Status.Phase = status.PhaseProgressing
		err = r.Client.Status().Update(ctx, instance)
		if err != nil {
			r.Log.Error(err, "Failed to add conditions to status of DataScienceCluster resource.", "DataScienceCluster", req.Namespace, "Request.Name", req.Name)
			// r.Recorder.Eventf(instance, corev1.EventTypeWarning, "DataScienceClusterReconcileError",
			// 	"Failed to add conditions to DataScienceCluster instance %s", instance.Name)
			return ctrl.Result{}, err
		}
	}

	// Check if the DataScienceCluster was deleted
	deleted := instance.GetDeletionTimestamp() != nil
	finalizers := sets.NewString(instance.GetFinalizers()...)
	if deleted {
		if !finalizers.Has(finalizer) {
			r.Log.Info("DataScienceCluster instance has been deleted.", "instance", instance.Name)
			return ctrl.Result{}, nil
		}
		r.Log.Info("Deleting DataScienceCluster instance", "instance", instance.Name)

		// 	// TODO: implement delete logic

		// 	// Remove finalizer once DataScienceCluster deletion is completed.
		finalizers.Delete(finalizer)
		instance.SetFinalizers(finalizers.List())
		finalizerError := r.Client.Update(context.TODO(), instance)
		for retryCount := 0; errors.IsConflict(finalizerError) && retryCount < finalizerMaxRetries; retryCount++ {
			// Based on Istio operator at https://github.com/istio/istio/blob/master/operator/pkg/controller/istiocontrolplane/istiocontrolplane_controller.go
			// for finalizer removal errors workaround.
			r.Log.Info("Conflict during finalizer removal, retrying.")
			_ = r.Client.Get(ctx, req.NamespacedName, instance)
			finalizers = sets.NewString(instance.GetFinalizers()...)
			finalizers.Delete(finalizer)
			instance.SetFinalizers(finalizers.List())
			finalizerError = r.Client.Update(ctx, instance)
		}
		if finalizerError != nil {
			r.Log.Error(finalizerError, "error removing finalizer")
			return ctrl.Result{}, finalizerError
		}
		return ctrl.Result{}, nil
	} else if !finalizers.Has(finalizer) {
		// 	// Based on kfdef code at https://github.com/opendatahub-io/opendatahub-operator/blob/master/controllers/kfdef.apps.kubeflow.org/kfdef_controller.go#L191
		// 	// TODO: consider if this piece of logic is really needed or if we can remove it.
		r.Log.Info("Normally this should not happen. Adding the finalizer", finalizer, req)
		finalizers.Insert(finalizer)
		instance.SetFinalizers(finalizers.List())
		err = r.Client.Update(ctx, instance)
		if err != nil {
			r.Log.Error(err, "failed to update DataScienceCluster with finalizer")
			return ctrl.Result{}, err
		}
	}

	// RECONCILE LOGIC
	plan := profiles.CreateReconciliationPlan(instance)

	if r.isManagedService() {
		//Apply osd specific permissions
		err = deploy.DeployManifestsFromPath(instance, r.Client,
			"/opt/odh-manifests/osd-configs",
			r.ApplicationsNamespace, r.Scheme, true)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	if err = reconcileDashboard(instance, r.Client, r.Scheme, plan); err != nil {
		r.Log.Error(err, "failed to reconcile DataScienceCluster (dashboard resources)")
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "DataScienceClusterReconcileError",
			"Failed to reconcile dashboard resources on DataScienceCluster instance %s", instance.Name)
		return ctrl.Result{}, err
	}
	if err = reconcileTraining(instance, r.Client, r.Scheme, plan); err != nil {
		r.Log.Error(err, "failed to reconcile DataScienceCluster (training resources)")
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "DataScienceClusterReconcileError",
			"Failed to reconcile training resources on DataScienceCluster instance %s", instance.Name)
		return ctrl.Result{}, err
	}
	if err = reconcileServing(instance, r.Client, r.Scheme, plan); err != nil {
		r.Log.Error(err, "failed to reconcile DataScienceCluster (serving resources)")
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "DataScienceClusterReconcileError",
			"Failed to reconcile serving resources on DataScienceCluster instance %s", instance.Name)
		return ctrl.Result{}, err
	}
	if err = reconcileWorkbench(instance, r.Client, r.Scheme, plan); err != nil {
		r.Log.Error(err, "failed to reconcile DataScienceCluster (workbench resources)")
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "DataScienceClusterReconcileError",
			"Failed to reconcile common workbench on DataScienceCluster instance %s", instance.Name)
		return ctrl.Result{}, err
	}

	// finalize reconciliation
	status.SetCompleteCondition(&instance.Status.Conditions, status.ReconcileCompleted, "DataScienceCluster resource reconciled successfully.")
	err = r.Client.Update(ctx, instance)
	if err != nil {
		r.Log.Error(err, "failed to update DataScienceCluster after successfuly completed reconciliation")
		return ctrl.Result{}, err
	} else {
		r.Log.Info("DataScienceCluster Deployment Completed.")
		// r.Recorder.Eventf(instance, corev1.EventTypeNormal, "DataScienceClusterCreationSuccessful",
		// 	"DataScienceCluster instance %s created and deployed successfully", instance.Name)
	}

	instance.Status.Phase = status.PhaseReady
	err = r.Client.Status().Update(ctx, instance)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DataScienceClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dsc.DataScienceCluster{}).
		Owns(&operators.OperatorGroup{}).
		Owns(&authv1.Role{}).
		Owns(&authv1.RoleBinding{}).
		Owns(&authv1.ClusterRole{}).
		Owns(&authv1.ClusterRoleBinding{}).
		Owns(&corev1.Namespace{}).
		Complete(r)
}

// TODO: should we generalize this function and move it to a common place?
func (r *DataScienceClusterReconciler) isManagedService() bool {
	expectedAddon := &addonv1alpha1.Addon{}
	err := r.Client.Get(context.TODO(), client.ObjectKey{Name: "managed-odh"}, expectedAddon)
	if err != nil {
		if apierrs.IsNotFound(err) {
			return false
		} else {
			r.Log.Error(err, "error getting Addon instance for managed service")
			return false
		}
	}
	return true
}

// TODO: Move this to component package
func reconcileWorkbench(instance *dsc.DataScienceCluster, client client.Client, scheme *runtime.Scheme, plan *profiles.ReconciliationPlan) error {
	err := deploy.DeployManifestsFromPath(instance, client,
		"/opt/odh-manifests/odh-dashboard/base",
		"opendatahub",
		scheme, plan.Components[profiles.WorkbenchesComponent])
	return err
}

func reconcileServing(instance *dsc.DataScienceCluster, client client.Client, scheme *runtime.Scheme, plan *profiles.ReconciliationPlan) error {
	panic("unimplemented")
}

func reconcileTraining(instance *dsc.DataScienceCluster, client client.Client, scheme *runtime.Scheme, plan *profiles.ReconciliationPlan) error {
	panic("unimplemented")
}

func reconcileDashboard(instance *dsc.DataScienceCluster, client client.Client, scheme *runtime.Scheme, plan *profiles.ReconciliationPlan) error {
	panic("unimplemented")
}
