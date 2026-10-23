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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util"
	conditions "sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/paused"
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
	"github.com/vultr/govultr/v3"
)

const (
	// loadBalancerRequeue is how long to wait before retrying a load balancer
	// that is not accepting members yet.
	loadBalancerRequeue = 15 * time.Second
	// notFoundRequeue is how long to wait before looking again for an
	// instance Vultr reported missing; a single 404 is never acted on.
	notFoundRequeue = 30 * time.Second
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

	deleting := !vultrMachine.DeletionTimestamp.IsZero()

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

	// Surface and honour pausing of the Cluster or of the VultrMachine itself.
	if isPaused, requeue, err := paused.EnsurePausedCondition(ctx, r.Client, cluster, vultrMachine); err != nil || isPaused || requeue {
		return ctrl.Result{}, err
	}

	// Fetch the VultrCluster.
	vultrCluster := &v1beta2.VultrCluster{}
	vultrClusterName := client.ObjectKey{
		Namespace: vultrMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	// A missing VultrCluster does not block deletion: deleting the instance only
	// needs the API client.
	if err := r.Get(ctx, vultrClusterName, vultrCluster); err != nil && !scope.MachineOnly() && !deleting {
		log.Info("VultrCluster is not available yet.")
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

	if deleting {
		return r.reconcileDelete(ctx, machineScope, clusterScope)
	}

	return r.reconcileNormal(ctx, machineScope, clusterScope)

}

func (r *VultrMachineReconciler) reconcileNormal(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineScope.Info("Reconciling VultrMachine")
	vultrMachine := machineScope.VultrMachine

	// Early exit only for terminal failures; anything else is retried.
	if machineScope.HasTerminalFailure() {
		machineScope.Info("Machine infrastructure has failed, skipping reconciliation")
		return reconcile.Result{}, nil
	}

	// If the VultrMachine doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(machineScope.VultrMachine, v1beta2.MachineFinalizer)

	if !conditions.IsTrue(machineScope.Cluster, clusterv1.InfrastructureReadyCondition) {
		machineScope.Info("Cluster infrastructure is not ready yet")
		machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.InstanceWaitingForClusterInfrastructureReason, "Waiting for cluster infrastructure")
		return reconcile.Result{}, nil
	}

	// Ensure bootstrap data is available before creating instance
	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap data secret reference is not yet available")
		machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.InstanceWaitingForBootstrapDataReason, "Waiting for bootstrap data")
		return reconcile.Result{}, nil
	}

	instanceSvc := services.NewService(ctx, clusterScope)

	instance, err := r.findOrCreateInstance(instanceSvc, machineScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if instance == nil {
		// Reported missing; look again rather than acting on one answer.
		return reconcile.Result{RequeueAfter: notFoundRequeue}, nil
	}

	// Persist the ID first so a later error cannot orphan the instance.
	if machineScope.GetInstanceID() != instance.ID {
		machineScope.SetProviderID(instance.ID)
		if err := machineScope.PatchObject(ctx); err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "failed to persist provider ID for instance %s", instance.ID)
		}
	}

	// Set instance details on Machine
	machineScope.SetInstanceStatus(v1beta2.SubscriptionStatus(instance.Status))
	machineScope.SetCPU(instance.VCPUCount)
	machineScope.SetRAM(instance.RAM)
	machineScope.SetStorage(instance.Disk)
	r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "SetInstanceStatus", "Set", "Setting instance status %s", instance.Label)

	// Control-plane instances join the API server load balancer; requeue
	// until it accepts the member.
	if util.IsControlPlaneMachine(machineScope.Machine) {
		lbID := clusterScope.APIServerLoadbalancersRef().ResourceID
		if lbID == "" {
			machineScope.Info("Waiting for the load balancer to be created")
			machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.InstanceWaitingForLoadBalancerReason, "Waiting for the load balancer to be created")
			return reconcile.Result{RequeueAfter: loadBalancerRequeue}, nil
		}
		member, err := instanceSvc.EnsureLoadBalancerMember(lbID, instance.ID)
		if err != nil {
			r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeWarning, "LoadBalancerMemberFailed", "Failed",
				"Failed to add instance %s to load balancer %s: %v", instance.ID, lbID, err)
			return reconcile.Result{}, err
		}
		if !member {
			machineScope.Info("Waiting for the load balancer to become active", "load-balancer-id", lbID)
			machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.InstanceWaitingForLoadBalancerReason, "Waiting for the load balancer to become active")
			return reconcile.Result{RequeueAfter: loadBalancerRequeue}, nil
		}
	}

	// Retrieve instance addresses
	r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "GetInstanceAddress", "Getting",
		"Getting address for instance %s", instance.ID)
	addrs, err := instanceSvc.GetInstanceAddress(instance)
	if err != nil {
		r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeWarning, "GetInstanceAddressFailed", "GetFailed",
			"Failed to get address for instance %s: %v", instance.ID, err)
		machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.InstanceAddressFailedReason, err.Error())
		return reconcile.Result{}, err
	}
	machineScope.SetAddresses(addrs)
	r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "GetInstanceAddressSuccess", "Retrieved",
		"Successfully retrieved address for instance %s: %v", instance.ID, addrs)

	// Set the Ready condition based on instance status
	switch v1beta2.SubscriptionStatus(instance.Status) {
	case v1beta2.SubscriptionStatusPending:
		machineScope.Info("Machine instance is pending", "instance-id", instance.ID)
		machineScope.SetReadyCondition(metav1.ConditionUnknown, v1beta2.InstancePendingReason, "Instance is provisioning")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil

	case v1beta2.SubscriptionStatusActive:
		machineScope.Info("Machine instance is active", "instance-id", instance.ID)
		machineScope.SetReadyCondition(metav1.ConditionTrue, v1beta2.InstanceActiveReason, "Instance is active")
		machineScope.SetReady()
		vultrMachine.Status.Initialization.Provisioned = true
		return reconcile.Result{}, nil

	default:
		errMsg := fmt.Sprintf("Instance status %q is unexpected", instance.Status)
		machineScope.Info("Machine instance status is unexpected", "instance-id", instance.ID, "status", instance.Status)
		machineScope.SetTerminalFailure(v1beta2.InstanceUnexpectedStatusReason, errMsg)
		return reconcile.Result{}, nil
	}
}

// findOrCreateInstance finds the instance by providerID, then by label and
// tags, and creates it only when the machine has no providerID. A machine whose
// known instance is reported missing is never re-created: it is marked
// InstanceNotFound and checked again, and MachineHealthCheck decides when to
// replace it. Returns nil, nil in that case.
func (r *VultrMachineReconciler) findOrCreateInstance(instanceSvc *services.Service, machineScope *scope.MachineScope) (*govultr.Instance, error) {
	vultrMachine := machineScope.VultrMachine

	if id := machineScope.GetInstanceID(); id != "" {
		instance, err := instanceSvc.GetInstance(id)
		if err != nil {
			return nil, err
		}
		if instance == nil {
			msg := fmt.Sprintf("instance %s not found, checking again", id)
			if !conditions.IsFalse(vultrMachine, v1beta2.MachineReadyCondition) || conditions.GetReason(vultrMachine, v1beta2.MachineReadyCondition) != v1beta2.InstanceNotFoundReason {
				r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeWarning, "InstanceNotFound", "NotFound", msg)
			}
			machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.InstanceNotFoundReason, msg)
			return nil, nil
		}
		return instance, nil
	}

	instance, err := instanceSvc.FindInstanceForMachine(machineScope.Name(), machineScope.Role())
	if err != nil {
		return nil, err
	}
	if instance != nil {
		r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "InstanceAdopted", "Adopted",
			"Adopted existing instance %s", instance.ID)
		return instance, nil
	}

	r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "InstanceCreating", "Creating", "No instance found, attempting create")
	instance, err = instanceSvc.CreateInstance(machineScope)
	if err != nil {
		r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeWarning, "InstanceCreatingError", "CreateFailed",
			"Failed to create instance for VultrMachine %s/%s: %v", vultrMachine.Namespace, vultrMachine.Name, err)
		machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.InstanceCreationFailedReason, err.Error())
		vultrMachine.Status.Initialization.Provisioned = false
		return nil, err
	}
	r.Recorder.Eventf(vultrMachine, nil, corev1.EventTypeNormal, "InstanceCreated", "Created",
		"Created new instance %s (%s)", instance.ID, instance.Label)
	return instance, nil
}

func (r *VultrMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) { //nolint: unparam
	machineScope.Info("Reconciling delete VultrMachine")
	vultrmachine := machineScope.VultrMachine
	machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.InstanceDeletingReason, "Deleting the instance")

	instanceSvc := services.NewService(ctx, clusterScope)

	// Delete by ID when there is one, then list the instances owned by this
	// machine; the finalizer goes only once that list is empty.
	id := machineScope.GetInstanceID()
	deleted := false
	if id != "" {
		var err error
		if deleted, err = instanceSvc.DeleteInstance(id); err != nil {
			r.Recorder.Eventf(vultrmachine, nil, corev1.EventTypeWarning, "InstanceDeleteFailed", "DeleteFailed",
				"Failed to delete instance %s: %v", id, err)
			return reconcile.Result{}, err
		}
		if deleted {
			r.Recorder.Eventf(vultrmachine, nil, corev1.EventTypeNormal, "InstanceDeleted", "Deleted", "Deleted instance %s", id)
		}
	}

	remaining, err := instanceSvc.FindInstanceForMachine(machineScope.Name(), machineScope.Role())
	if err != nil {
		return reconcile.Result{}, err
	}
	if remaining != nil {
		if remaining.ID == id {
			if deleted {
				// The list has not caught up with the delete yet.
				return reconcile.Result{RequeueAfter: notFoundRequeue}, nil
			}
			machineScope.Info("Vultr reported the instance missing but still lists it, checking again", "instance-id", id)
			return reconcile.Result{RequeueAfter: notFoundRequeue}, nil
		}
		if _, err := instanceSvc.DeleteInstance(remaining.ID); err != nil {
			return reconcile.Result{}, err
		}
		r.Recorder.Eventf(vultrmachine, nil, corev1.EventTypeNormal, "InstanceDeleted", "Deleted", "Deleted instance %s", remaining.ID)
		return reconcile.Result{RequeueAfter: notFoundRequeue}, nil
	}

	if !deleted {
		r.Recorder.Eventf(vultrmachine, nil, corev1.EventTypeNormal, "NoInstanceFound", "NotFound", "No instance to delete")
	}
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
