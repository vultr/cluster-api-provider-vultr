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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pkg/errors"
	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/cluster-api-provider-vultr/cloud/services"
	"github.com/vultr/cluster-api-provider-vultr/util/reconciler"
)

// VultrClusterReconciler reconciles a VultrCluster object
type VultrClusterReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ReconcileTimeout time.Duration
	Recorder         record.EventRecorder
	WatchFilterValue string
}

// SetupWithManager sets up the controller with the Manager.
func (r *VultrClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	gvk := infrav1.GroupVersion.WithKind("VultrCluster")

	err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.VultrCluster{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(mgr.GetLogger(), r.WatchFilterValue)).
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(
				clusterutil.ClusterToInfrastructureMapFunc(context.TODO(), gvk, mgr.GetClient(), &infrav1.VultrCluster{}),
			),
			builder.WithPredicates(predicates.ClusterUnpausedAndInfrastructureReady(mgr.GetLogger())),
		).Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

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
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()

	log := ctrl.LoggerFrom(ctx)
	logger := ctrl.LoggerFrom(ctx).WithName("VultrClusterReconciler").WithValues("name", req.NamespacedName.String())

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
	cluster, err := clusterutil.GetOwnerCluster(ctx, r.Client, vultrCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get owner cluster: %w", err)
	}
	if cluster == nil {
		logger.Info("Cluster Controller has not yet set OwnerRef", "OwnerReferences", vultrCluster.OwnerReferences)
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, vultrCluster) {
		log.Info("VultrCluster or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create the cluster scope.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:       r.Client,
		Logger:       log,
		Cluster:      cluster,
		VultrCluster: vultrCluster,
	})
	if err != nil {
		return ctrl.Result{}, errors.Errorf("failed to create scope: %v", err)
	}

	// Always close the scope when exiting this function so we can persist any VultrMachine changes.
	defer func() {
		if err := clusterScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Handle deleted clusters
	if !vultrCluster.DeletionTimestamp.IsZero() {
		_, err := r.reconcileDelete(ctx, clusterScope)
		return ctrl.Result{}, err
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, clusterScope)

}

func (r *VultrClusterReconciler) reconcileNormal(ctx context.Context, clusterScope *scope.ClusterScope) (res ctrl.Result, reterr error) {
	clusterScope.Info("Reconciling VultrCluster")
	vultrcluster := clusterScope.VultrCluster
	// If the VultrCluster doesn't have finalizer, add it.
	if !controllerutil.ContainsFinalizer(vultrcluster, infrav1.ClusterFinalizer) {
		controllerutil.AddFinalizer(vultrcluster, infrav1.ClusterFinalizer)

		return ctrl.Result{Requeue: true}, nil
	}

	vlbservice := services.NewService(ctx, clusterScope)
	apiServerLoadbalancer := clusterScope.APIServerLoadbalancers()

	apiServerLoadbalancerRef := clusterScope.APIServerLoadbalancersRef()
	vlbID := apiServerLoadbalancerRef.ResourceID

	if apiServerLoadbalancer.ID != "" {
		vlbID = apiServerLoadbalancer.ID
	}

	loadbalancer, err := vlbservice.GetLoadBalancer(vlbID)
	if err != nil {
		return reconcile.Result{}, err
	}

	if loadbalancer == nil {
		loadbalancer, err = vlbservice.CreateLoadBalancer(apiServerLoadbalancer)
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "failed to create load balancers for VultrCluster %s/%s", vultrcluster.Namespace, vultrcluster.Name)
		}

		r.Recorder.Eventf(vultrcluster, corev1.EventTypeNormal, "LoadBalancerCreated", "Created new load balancers - %s", loadbalancer.Label)
	}

	apiServerLoadbalancerRef.ResourceID = loadbalancer.ID
	apiServerLoadbalancerRef.ResourceSubscriptionStatus = infrav1.SubscriptionStatus(loadbalancer.Status)
	apiServerLoadbalancer.ID = loadbalancer.ID

	if apiServerLoadbalancerRef.ResourcePowerStatus != infrav1.PowerStatusRunning && loadbalancer.IPV4 == "" {
		clusterScope.Info("Waiting on API server Global IP Address")
		return reconcile.Result{RequeueAfter: 15 * time.Second}, nil
	}

	r.Recorder.Eventf(vultrcluster, corev1.EventTypeNormal, "LoadBalancerReady", "LoadBalancer got an IP Address - %s", loadbalancer.IPV4)

	controlPlaneEndpoint := loadbalancer.IPV4

	clusterScope.SetControlPlaneEndpoint(clusterv1.APIEndpoint{
		Host: controlPlaneEndpoint,
		Port: int32(apiServerLoadbalancer.HealthCheck.Port),
	})

	clusterScope.Info("Set VultrCluster status to ready")
	clusterScope.SetReady()
	clusterScope.VultrCluster.Status.Ready = true
	r.Recorder.Eventf(vultrcluster, corev1.EventTypeNormal, "VultrClusterReady", "VultrCluster %s - has ready status", clusterScope.Name())
	return reconcile.Result{}, nil
}

func (r *VultrClusterReconciler) reconcileDelete(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Info("Reconciling delete VultrCluster")
	vultrcluster := clusterScope.VultrCluster

	vlbservice := services.NewService(ctx, clusterScope)
	apiServerLoadbalancerRef := clusterScope.APIServerLoadbalancersRef()
	vlbID := apiServerLoadbalancerRef.ResourceID

	loadbalancer, err := vlbservice.GetLoadBalancer(vlbID)
	if err != nil {
		return reconcile.Result{}, err
	}

	if loadbalancer == nil {
		clusterScope.V(2).Info("Unable to locate load balancer")
		r.Recorder.Eventf(vultrcluster, corev1.EventTypeWarning, "NoLoadBalancerFound", "Unable to find matching load balancer")
		controllerutil.RemoveFinalizer(vultrcluster, infrav1.ClusterFinalizer)
		return reconcile.Result{}, nil
	}

	if err := vlbservice.DeleteLoadBalancer(loadbalancer.ID); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "error deleting load balancer for VultrCluster %s/%s", vultrcluster.Namespace, vultrcluster.Name)
	}

	r.Recorder.Eventf(vultrcluster, corev1.EventTypeNormal, "LoadBalancerDeleted", "Deleted LoadBalancer - %s", loadbalancer.Label)

	// Cluster is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(vultrcluster, infrav1.ClusterFinalizer)
	return reconcile.Result{}, nil
}
