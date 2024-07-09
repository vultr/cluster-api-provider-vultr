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
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/pkg/errors"
	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta1"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/cluster-api-provider-vultr/cloud/services"
	"github.com/vultr/cluster-api-provider-vultr/util/reconciler"
	capierrors "sigs.k8s.io/cluster-api/errors"
)

// VultrMachineReconciler reconciles a VultrMachine object
type VultrMachineReconciler struct {
	client.Client
	Recorder         record.EventRecorder
	ReconcileTimeout time.Duration
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrmachines/finalizers,verbs=update

// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;watch;list
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines,verbs=get;watch;list
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch

func (r *VultrMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()

	log := ctrl.LoggerFrom(ctx)

	// Fetch the VultrMachine.
	vultrMachine := &infrav1.VultrMachine{}
	if err := r.Get(ctx, req.NamespacedName, vultrMachine); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
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

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, nil
	}

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

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, vultrCluster) {
		log.Info("VultrMachine or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
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

	// Create the machine scope
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
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

	r.Recorder.Event(vultrmachine, corev1.EventTypeNormal, "InstanceServiceInitializing", "Initializing instance service")
	instancesvc := services.NewService(ctx, clusterScope)
	r.Recorder.Event(vultrmachine, corev1.EventTypeNormal, "InstanceServiceInitialized", "Instance service initialized")

	machineID := machineScope.GetInstanceID()
	r.Recorder.Eventf(vultrmachine, corev1.EventTypeNormal, "InstanceRetrieving", "Retrieving instance with ID %s", machineID)
	instance, err := instancesvc.GetInstance(machineScope.GetInstanceID())
	if err != nil {
		return reconcile.Result{}, err
	}

	if instance == nil {
		r.Recorder.Eventf(vultrmachine, corev1.EventTypeNormal, "InstanceCreating", "Instance is nil attempting create %v", instance)
		instance, err = instancesvc.CreateInstance(machineScope)
		if err != nil {
			err = errors.Errorf("Failed to create instance instance for VultrMachine %s/%s: %v", vultrmachine.Namespace, vultrmachine.Name, err)
			r.Recorder.Event(vultrmachine, corev1.EventTypeWarning, "InstanceCreatingError", err.Error())
			machineScope.SetInstanceServerState(infrav1.ServerStateError)
			return reconcile.Result{}, err
		}
		r.Recorder.Eventf(vultrmachine, corev1.EventTypeNormal, "InstanceCreated", "Created new instance instance - %s", instance.Label)
	}

	r.Recorder.Eventf(vultrmachine, corev1.EventTypeNormal, "SetProviderID", "Setting Instance Provider ID %s", instance.Label)
	machineScope.SetProviderID(instance.ID)
	r.Recorder.Eventf(vultrmachine, corev1.EventTypeNormal, "SetInstanceStatus", "Setting Instance Status %s", instance.Label)
	machineScope.SetInstanceStatus(infrav1.SubscriptionStatus(instance.Status))

	if strings.Contains(instance.Label, "control-plane") {
		r.Recorder.Eventf(vultrmachine, corev1.EventTypeNormal, "AddInstanceToVLB", "Instance %s is a control plane node, adding to VLB", instance.ID)
		err := instancesvc.AddInstanceToVLB(clusterScope.APIServerLoadbalancersRef().ResourceID, instance.ID)
		if err != nil {
			r.Recorder.Eventf(vultrmachine, corev1.EventTypeWarning, "AddInstanceToVLBFailed", "Failed to add instance %s to VLB: %v", instance.ID, err)
			return reconcile.Result{}, errors.Wrap(err, "failed to add instance to VLB")
		}
		r.Recorder.Eventf(vultrmachine, corev1.EventTypeNormal, "AddInstanceToVLBSuccess", "Successfully added instance %s to VLB", instance.ID)
	}

	r.Recorder.Eventf(vultrmachine, corev1.EventTypeNormal, "GetInstanceAddress", "Getting address for instance %s", instance.ID)
	addrs, err := instancesvc.GetInstanceAddress(instance)
	if err != nil {
		r.Recorder.Eventf(vultrmachine, corev1.EventTypeWarning, "GetInstanceAddressFailed", "Failed to get address for instance %s: %v", instance.ID, err)
		machineScope.SetFailureMessage(errors.New("failed to get instance address"))
		return reconcile.Result{}, err
	}
	r.Recorder.Eventf(vultrmachine, corev1.EventTypeNormal, "GetInstanceAddressSuccess", "Successfully retrieved address for instance %s: %v", instance.ID, addrs)
	machineScope.SetAddresses(addrs)

	// Proceed to reconcile the VultrMachine state based on SubscriptionStatus.
	switch infrav1.SubscriptionStatus(instance.Status) {
	case infrav1.SubscriptionStatusPending:
		machineScope.Info("Machine instance is pending", "instance-id", machineScope.GetInstanceID())
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	case infrav1.SubscriptionStatusActive:
		machineScope.Info("Machine instance is active", "instance-id", machineScope.GetInstanceID())
		machineScope.SetReady()
		return reconcile.Result{}, nil
	default:
		machineScope.SetFailureReason(capierrors.UpdateMachineError)
		machineScope.SetFailureMessage(errors.Errorf("Instance status %q is unexpected", instance.Status))
		return reconcile.Result{}, nil
	}
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
