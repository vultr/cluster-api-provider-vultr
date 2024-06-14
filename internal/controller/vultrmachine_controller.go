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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/cluster-api-provider-vultr/cloud/services"
)

// VultrMachineReconciler reconciles a VultrMachine object
type VultrMachineReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	Log              logr.Logger
	VultrApiKey      string
	WatchFilterValue string
	Recorder         record.EventRecorder
}

//+kubebuilder:rbac:groups=infra.cluster.x-k8s.io,resources=vultrmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.cluster.x-k8s.io,resources=vultrmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infra.cluster.x-k8s.io,resources=vultrmachines/finalizers,verbs=update

// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;watch;list
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines,verbs=get;watch;list
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VultrMachine object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile

func (r *VultrMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := r.Log.WithValues("vultrmachine", req.NamespacedName)

	// Fetch the VultrMachine.
	vultrMachine := &infrav1.VultrMachine{}
	err := r.Get(ctx, req.NamespacedName, vultrMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			//return ctrl.Result{}, nil
			log.Error(err, "Failed to fetch VultrMachine")
		}
		return ctrl.Result{}, err
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, vultrMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		log.Info("Machine Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	log = r.Log.WithValues("VultrMachine", machine.Name)

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, nil
	}

	log = r.Log.WithValues("cluster", cluster.Name)

	// Fetch the VultrCluster.
	vultrCluster := &infrav1.VultrCluster{}
	vultrClusterName := client.ObjectKey{
		Namespace: vultrMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, vultrClusterName, vultrCluster); err != nil {
		log.Info("VultrCluster is not available yet.")
		return ctrl.Result{}, nil
	}

	log = r.Log.WithValues("vultrCluster", vultrCluster.Name)

	// // Retrieve the API key from environment variables
	// apiKey := os.Getenv("VULTR_API_KEY")
	// if apiKey == "" {
	// 	return ctrl.Result{}, errors.New("environment variable VULTR_API_KEY is required")
	// }

	// Create the cluster scope
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

	// Create the machine scope
	machineScope, err := scope.NewMachineScope(ctx, r.VultrApiKey, scope.MachineScopeParams{
		Client:       r.Client,
		Logger:       log,
		Cluster:      cluster,
		Machine:      machine,
		VultrCluster: vultrCluster,
		VultrMachine: vultrMachine,
	})
	if err != nil {
		return ctrl.Result{}, errors.Errorf("failed to create machine scope: %v", err)
	}

	defer func() {
		err := machineScope.Close()
		if err != nil && reterr == nil {
			reterr = err
		}
	}()

	if !vultrMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope, clusterScope)
	}

	return r.reconcileNormal(ctx, machineScope, clusterScope)
	//return ctrl.Result{}, nil

}

func (r *VultrMachineReconciler) reconcileNormal(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineScope.Info("Reconciling VultrMachine")
	vultrmachine := machineScope.VultrMachine

	if vultrmachine.Status.FailureReason != nil || vultrmachine.Status.FailureMessage != nil {
		machineScope.Info("Error state detected, skipping reconciliation")
		return reconcile.Result{}, nil
	}

	// If the VultrMachine doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(machineScope.VultrMachine, infrav1.MachineFinalizer)

	if !machineScope.Cluster.Status.InfrastructureReady {
		machineScope.Info("Cluster infrastructure is not ready yet")
		return reconcile.Result{}, nil
	}

	// Make sure bootstrap data is available and populated.
	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap data secret reference is not yet available")
		return reconcile.Result{}, nil
	}

	instancesvc := services.NewService(ctx, clusterScope)
	instance, err := instancesvc.GetInstance(machineScope.GetInstanceID())
	if err != nil {
		return reconcile.Result{}, err
	}

	if instance == nil {
		instance, err = instancesvc.CreateInstance(machineScope)
		if err != nil {
			err = errors.Errorf("Failed to create instance instance for VultrMachine %s/%s: %v", vultrmachine.Namespace, vultrmachine.Name, err)
			r.Recorder.Event(vultrmachine, corev1.EventTypeWarning, "InstanceCreatingError", err.Error())
			machineScope.SetInstanceServerState(infrav1.ServerStateError)
			return reconcile.Result{}, err
		}
		r.Recorder.Eventf(vultrmachine, corev1.EventTypeNormal, "InstanceCreated", "Created new instance instance - %s", instance.Label)
	}

	// Register the finalizer immediately to avoid orphaning Vultr resources on delete.
	if err := machineScope.PatchObject(ctx); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *VultrMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineScope.Info("Reconciling delete VultrMachine")
	vultrmachine := machineScope.VultrMachine

	vultrcomputesvc := services.NewService(ctx, clusterScope)
	vultrInstance, err := vultrcomputesvc.GetInstance(machineScope.GetInstanceID())
	if err != nil {
		return reconcile.Result{}, err
	}

	if vultrInstance != nil {
		if err := vultrcomputesvc.DeleteInstance(machineScope.GetInstanceID()); err != nil {
			return reconcile.Result{}, err
		}
	} else {
		clusterScope.V(2).Info("Unable to locate instance")
		r.Recorder.Eventf(vultrmachine, corev1.EventTypeWarning, "NoInstanceFound", "Skip deleting")
	}

	r.Recorder.Eventf(vultrmachine, corev1.EventTypeNormal, "InstanceDeleted", "Deleted a instance - %s", machineScope.Name())
	controllerutil.RemoveFinalizer(vultrmachine, infrav1.MachineFinalizer)
	return reconcile.Result{}, nil
}

// // SetupWithManager sets up the controller with the Manager.
// func (r *VultrMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
// 	vultrMachineMapper, err := clusterutil.ClusterToTypedObjectsMapper(r.Client, &infrav1.VultrMachineList{}, mgr.GetScheme())
// 	if err != nil {
// 		return fmt.Errorf("failed to create mapper for VultrMachines: %w", err)
// 	}

// 	err = ctrl.NewControllerManagedBy(mgr).
// 		For(&infrav1.VultrMachine{}).
// 		Watches(
// 			&clusterv1.Machine{},
// 			handler.EnqueueRequestsFromMapFunc(clusterutil.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("VultrMachine"))),
// 		).
// 		Watches(
// 			&infrav1.VultrCluster{},
// 			handler.EnqueueRequestsFromMapFunc(r.VultrClusterToVultrMachines(mgr.GetLogger())),
// 		).
// 		Watches(
// 			&clusterv1.Cluster{},
// 			handler.EnqueueRequestsFromMapFunc(vultrMachineMapper),
// 			builder.WithPredicates(predicates.ClusterUnpausedAndInfrastructureReady(mgr.GetLogger())),
// 		).
// 		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(mgr.GetLogger(), r.WatchFilterValue)).
// 		Complete(r)
// 	if err != nil {
// 		return fmt.Errorf("failed to build controller: %w", err)
// 	}

// 	return nil
// }

func (r *VultrMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, _ controller.Options) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.VultrMachine{}).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))). // don't queue reconcile if resource is paused
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("VultrMachine"))),
		).
		Watches(
			&infrav1.VultrCluster{},
			handler.EnqueueRequestsFromMapFunc(r.VultrClusterToVultrMachines(ctx)),
		).
		Build(r)
	if err != nil {
		return errors.Wrapf(err, "error creating controller")
	}

	clusterToObjectFunc, err := util.ClusterToTypedObjectsMapper(r.Client, &infrav1.VultrMachineList{}, mgr.GetScheme())
	if err != nil {
		return errors.Wrapf(err, "failed to create mapper for Cluster to VultrMachines")
	}

	// Add a watch on clusterv1.Cluster object for unpause & ready notifications.
	if err := c.Watch(
		source.Kind(mgr.GetCache(), &clusterv1.Cluster{}),
		handler.EnqueueRequestsFromMapFunc(clusterToObjectFunc),
		predicates.ClusterUnpausedAndInfrastructureReady(ctrl.LoggerFrom(ctx)),
	); err != nil {
		return errors.Wrapf(err, "failed adding a watch for ready clusters")
	}

	return nil
}

// VultrClusterToVultrMachines convert the cluster to machines spec.
func (r *VultrMachineReconciler) VultrClusterToVultrMachines(ctx context.Context) handler.MapFunc {
	log := ctrl.LoggerFrom(ctx)
	return func(ctx context.Context, o client.Object) []ctrl.Request {
		result := []ctrl.Request{}

		c, ok := o.(*infrav1.VultrCluster)
		if !ok {
			log.Error(errors.Errorf("expected a VultrCluster but got a %T", o), "failed to get VultrMachine for VultrCluster")
			return nil
		}

		cluster, err := util.GetOwnerCluster(ctx, r.Client, c.ObjectMeta)
		switch {
		case apierrors.IsNotFound(err) || cluster == nil:
			return result
		case err != nil:
			log.Error(err, "failed to get owning cluster")
			return result
		}

		labels := map[string]string{clusterv1.ClusterNameLabel: cluster.Name}
		machineList := &clusterv1.MachineList{}
		if err := r.List(ctx, machineList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
			log.Error(err, "failed to list Machines")
			return nil
		}
		for _, m := range machineList.Items {
			if m.Spec.InfrastructureRef.Name == "" {
				continue
			}
			name := client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.InfrastructureRef.Name}
			result = append(result, ctrl.Request{NamespacedName: name})
		}

		return result
	}
}
