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
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	conditions "sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pkg/errors"
	"github.com/vultr/cluster-api-provider-vultr/api/v1beta2"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/cluster-api-provider-vultr/cloud/services"
	"github.com/vultr/cluster-api-provider-vultr/util/reconciler"
)

// VultrMachineReconciler reconciles a VultrMachine object
type VultrMachineReconciler struct {
	client.Client
	Recorder         events.EventRecorder
	ReconcileTimeout time.Duration
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrmachines/finalizers,verbs=update

// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;watch;list
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines,verbs=get;watch;list
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch;update
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch

func (r *VultrMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()

	log := ctrl.LoggerFrom(ctx)

	// Fetch the VultrMachine.
	vultrMachine := &v1beta2.VultrMachine{}
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
	vultrCluster := &v1beta2.VultrCluster{}
	vultrClusterName := client.ObjectKey{
		Namespace: vultrMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Get(ctx, vultrClusterName, vultrCluster); err != nil && !scope.MachineOnly() {
		log.Info("VultrCluster is not available yet.")
		return ctrl.Result{}, nil
	}

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, vultrCluster) && !scope.MachineOnly() {
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

	if !vultrMachine.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope, clusterScope)
	}

	return r.reconcileNormal(ctx, machineScope, clusterScope)

}

func (r *VultrMachineReconciler) reconcileNormal(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineScope.Info("Reconciling VultrMachine")
	vultrMachine := machineScope.VultrMachine

	// Early exit if machine infrastructure is in a error state
	if conditions.IsFalse(vultrMachine, clusterv1.InfrastructureReadyCondition) {
		machineScope.Info("Machine infrastructure has failed, skipping reconciliation")
		return reconcile.Result{}, nil
	}

	// If the VultrMachine doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(machineScope.VultrMachine, v1beta2.MachineFinalizer)

	if !conditions.IsTrue(machineScope.Cluster, clusterv1.InfrastructureReadyCondition) {
		machineScope.Info("Cluster infrastructure is not ready yet")
		return reconcile.Result{}, nil
	}

	// Ensure bootstrap data is available before creating instance
	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap data secret reference is not yet available")
		return reconcile.Result{}, nil
	}

	r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "InstanceServiceInitializing", "Initializing", "Initializing instance service")
	instanceSvc := services.NewService(ctx, clusterScope)
	r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "InstanceServiceInitialized", "Initialized", "Instance service initialized")

	machineID := machineScope.GetInstanceID()
	r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "InstanceRetrieving", "Retrieving", "Retrieving instance with ID %s", machineID)
	instance, err := instanceSvc.GetInstance(machineID)
	if err != nil {
		return reconcile.Result{}, err
	}

	if instance == nil {
		r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "InstanceCreating", "Creating", "Instance is nil, attempting create")
		instance, err = instanceSvc.CreateInstance(machineScope)
		instancePayload, _ := json.Marshal(vultrMachine)
		machineScope.Info("Created new instance", "payload", string(instancePayload))
		if err != nil {
			r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeWarning, "InstanceCreatingError", "CreateFailed",
				"Failed to create instance for VultrMachine %s/%s: %v", vultrMachine.Namespace, vultrMachine.Name, err)
			conditions.Set(vultrMachine, metav1.Condition{
				Type:               clusterv1.InfrastructureReadyCondition,
				Status:             metav1.ConditionFalse,
				Reason:             "InstanceCreationFailed",
				Message:            err.Error(),
				LastTransitionTime: metav1.Now(),
			})
			vultrMachine.Status.Initialization.Provisioned = false
			return reconcile.Result{}, err
		}
		r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "InstanceCreated", "Created",
			"Created new instance - %s, payload: %s", instance.Label, string(instancePayload))
	}

	// Set instance details on Machine
	machineScope.SetProviderID(instance.ID)
	machineScope.SetInstanceStatus(v1beta2.SubscriptionStatus(instance.Status))
	machineScope.SetCPU(instance.VCPUCount)
	machineScope.SetRAM(instance.RAM)
	machineScope.SetStorage(instance.Disk)
	r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "SetInstanceStatus", "Set", "Setting instance status %s", instance.Label)

	// Add control-plane nodes to VLB
	if util.IsControlPlaneMachine(machineScope.Machine) {
		r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "AddInstanceToVLB", "Adding",
			"Instance %s is a control plane node, adding to VLB", instance.ID)
		if err := instanceSvc.AddInstanceToVLB(clusterScope.APIServerLoadbalancersRef().ResourceID, instance.ID); err != nil {
			r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeWarning, "AddInstanceToVLBFailed", "AddFailed",
				"Failed to add instance %s to VLB: %v", instance.ID, err)
			return reconcile.Result{}, errors.Wrap(err, "failed to add instance to VLB")
		}
		r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "AddInstanceToVLBSuccess", "Added",
			"Successfully added instance %s to VLB", instance.ID)
	}

	// Retrieve instance addresses
	r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "GetInstanceAddress", "Getting",
		"Getting address for instance %s", instance.ID)
	addrs, err := instanceSvc.GetInstanceAddress(instance)
	if err != nil {
		r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeWarning, "GetInstanceAddressFailed", "GetFailed",
			"Failed to get address for instance %s: %v", instance.ID, err)
		conditions.Set(vultrMachine, metav1.Condition{
			Type:               clusterv1.InfrastructureReadyCondition,
			Status:             metav1.ConditionFalse,
			Reason:             "FailedToGetInstanceAddress",
			Message:            err.Error(),
			LastTransitionTime: metav1.Now(),
		})
		return reconcile.Result{}, err
	}
	machineScope.SetAddresses(addrs)
	r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "GetInstanceAddressSuccess", "Retrieved",
		"Successfully retrieved address for instance %s: %v", instance.ID, addrs)

	// Set InfrastructureReadyCondition based on instance status
	switch v1beta2.SubscriptionStatus(instance.Status) {
	case v1beta2.SubscriptionStatusPending:
		machineScope.Info("Machine instance is pending", "instance-id", machineScope.GetInstanceID())
		conditions.Set(vultrMachine, metav1.Condition{
			Type:               clusterv1.InfrastructureReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             "InstancePending",
			Message:            "Instance is provisioning",
			LastTransitionTime: metav1.Now(),
		})

		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil

	case v1beta2.SubscriptionStatusActive:
		machineScope.Info("Machine instance is active", "instance-id", machineScope.GetInstanceID())
		conditions.Set(vultrMachine, metav1.Condition{
			Type:               clusterv1.InfrastructureReadyCondition,
			Status:             metav1.ConditionTrue,
			Reason:             "InstanceActive",
			Message:            "Instance is active",
			LastTransitionTime: metav1.Now(),
		})
		machineScope.SetReady()
		vultrMachine.Status.Initialization.Provisioned = true
		return reconcile.Result{}, nil

	default:
		errMsg := fmt.Sprintf("Instance status %q is unexpected", instance.Status)
		machineScope.Info("Machine instance status is unexpected", "instance-id", machineScope.GetInstanceID(), "status", instance.Status)
		conditions.Set(vultrMachine, metav1.Condition{
			Type:               clusterv1.InfrastructureReadyCondition,
			Status:             metav1.ConditionFalse,
			Reason:             "UnexpectedStatus",
			Message:            errMsg,
			LastTransitionTime: metav1.Now(),
		})
		return reconcile.Result{}, nil
	}
}

func (r *VultrMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) { //nolint: unparam
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
		r.Recorder.Eventf(vultrmachine, nil, corev1.EventTypeWarning, "NoInstanceFound", "NotFound", "Skip deleting")
	}

	r.Recorder.Eventf(vultrmachine, nil, corev1.EventTypeNormal, "InstanceDeleted", "Deleted", "Deleted a instance - %s", machineScope.Name())
	controllerutil.RemoveFinalizer(vultrmachine, v1beta2.MachineFinalizer)
	return reconcile.Result{}, nil
}
func (r *VultrMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, _ controller.Options) error {
	clusterToObjectFunc, err := util.ClusterToTypedObjectsMapper(r.Client, &v1beta2.VultrMachineList{}, mgr.GetScheme())
	if err != nil {
		return errors.Wrapf(err, "failed to create mapper for Cluster to VultrMachines")
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta2.VultrMachine{}).
		WithEventFilter(predicates.ResourceNotPaused(mgr.GetScheme(), ctrl.LoggerFrom(ctx))).
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(v1beta2.GroupVersion.WithKind("VultrMachine"))),
		).
		Watches(
			&v1beta2.VultrCluster{},
			handler.EnqueueRequestsFromMapFunc(r.VultrClusterToVultrMachines(ctx)),
		).
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(clusterToObjectFunc),
			builder.WithPredicates(predicates.ClusterPausedTransitionsOrInfrastructureProvisioned(mgr.GetScheme(), ctrl.LoggerFrom(ctx))),
		).
		Complete(r)
}

// VultrClusterToVultrMachines convert the cluster to machines spec.
func (r *VultrMachineReconciler) VultrClusterToVultrMachines(ctx context.Context) handler.MapFunc {
	log := ctrl.LoggerFrom(ctx)
	return func(ctx context.Context, o client.Object) []ctrl.Request {
		result := []ctrl.Request{}

		c, ok := o.(*v1beta2.VultrCluster)
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
