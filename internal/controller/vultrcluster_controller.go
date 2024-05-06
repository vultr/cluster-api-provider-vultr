/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pkg/errors"
	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
)

// VultrClusterReconciler reconciles a VultrCluster object
type VultrClusterReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ReconcileTimeout time.Duration
	Recorder         record.EventRecorder
	VultrApiKey      string
}

//+kubebuilder:rbac:groups=infra.cluster.x-k8s.io,resources=vultrclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.cluster.x-k8s.io,resources=vultrclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infra.cluster.x-k8s.io,resources=vultrclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VultrCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *VultrClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	// ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	// defer cancel()

	log := ctrl.LoggerFrom(ctx)

	// Fetch the VultrCluster.
	vultrCluster := &infrav1.VultrCluster{}
	err := r.Get(ctx, req.NamespacedName, vultrCluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("VultrCluster resource not found or already deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Unable to fetch VultrCluster resource")
		return ctrl.Result{}, err
	}

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, vultrCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get owner cluster: %w", err)
	}
	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}
	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, vultrCluster) {
		log.Info("VultrCluster or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	// Create the cluster scope.
	clusterScope, err := scope.NewClusterScope(ctx, r.VultrApiKey, scope.ClusterScopeParams{
		Client:       r.Client,
		Logger:       log,
		Cluster:      cluster,
		VultrCluster: vultrCluster,
	})
	if err != nil {
		return ctrl.Result{}, errors.Errorf("failed to create scope: %v", err)
	}

	// Always close the scope when exiting this function so we can persist any changes.
	defer func() {
		err := clusterScope.Close()
		if err != nil && reterr == nil {
			reterr = err
		}
	}()

	//Handle deleted clusters
	if !vultrCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, clusterScope)
	}

	// Handle non-deleted clusters
	return r.reconcile(ctx, clusterScope)
	//return ctrl.Result{}, nil

}

func (r *VultrClusterReconciler) reconcile(ctx context.Context, clusterScope *scope.ClusterScope) (res ctrl.Result, reterr error) {
	res = ctrl.Result{}
	// process controlplane endpoint
	if clusterScope.VultrCluster.Spec.ControlPlaneEndpoint.Host == "" {

		//TODO

	}

	clusterScope.VultrCluster.Status.Ready = true
	conditions.MarkTrue(clusterScope.VultrCluster, clusterv1.ReadyCondition)

	return res, nil

}

func (r *VultrClusterReconciler) reconcileDelete(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Info("Reconciling delete VultrCluster")
	vultrcluster := clusterScope.VultrCluster

	//TODO

	// Cluster is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(vultrcluster, infrav1.ClusterFinalizer)
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VultrClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.VultrCluster{}).
		Complete(r)
}
