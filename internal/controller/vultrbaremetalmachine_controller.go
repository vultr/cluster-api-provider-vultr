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

	"github.com/pkg/errors"
	"github.com/vultr/govultr/v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	"k8s.io/utils/ptr"
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

	"github.com/vultr/cluster-api-provider-vultr/api/v1beta2"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/cluster-api-provider-vultr/cloud/services"
	"github.com/vultr/cluster-api-provider-vultr/util/reconciler"
)

const (
	// bareMetalPendingRequeue is how often a provisioning bare metal server is polled.
	// Bare metal provisioning takes several minutes.
	bareMetalPendingRequeue = 30 * time.Second
	// bareMetalVPCRequeue is how often a requested VPC attachment is re-checked.
	bareMetalVPCRequeue = 15 * time.Second
	// bareMetalNotFoundRequeue is how long to wait before looking again for a
	// server Vultr reported missing; a single 404 is never acted on.
	bareMetalNotFoundRequeue = 30 * time.Second
	// bareMetalLoadBalancerRequeue is how often a not yet active load balancer is re-checked.
	bareMetalLoadBalancerRequeue = 15 * time.Second
)

// VultrBareMetalMachineReconciler reconciles a VultrBareMetalMachine object.
type VultrBareMetalMachineReconciler struct {
	client.Client
	Recorder         events.EventRecorder
	ReconcileTimeout time.Duration
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrbaremetalmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrbaremetalmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vultrbaremetalmachines/finalizers,verbs=update

func (r *VultrBareMetalMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()

	log := ctrl.LoggerFrom(ctx)

	bareMetalMachine := &v1beta2.VultrBareMetalMachine{}
	if err := r.Get(ctx, req.NamespacedName, bareMetalMachine); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	deleting := !bareMetalMachine.DeletionTimestamp.IsZero()

	machine, err := util.GetOwnerMachine(ctx, r.Client, bareMetalMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		log.Info("Machine Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, nil
	}

	if isPaused, requeue, err := paused.EnsurePausedCondition(ctx, r.Client, cluster, bareMetalMachine); err != nil || isPaused || requeue {
		return ctrl.Result{}, err
	}

	vultrCluster := &v1beta2.VultrCluster{}
	vultrClusterName := client.ObjectKey{
		Namespace: bareMetalMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	// A missing VultrCluster does not block deletion: deleting the server only
	// needs the API client, and the load balancer membership is skipped.
	if err := r.Get(ctx, vultrClusterName, vultrCluster); err != nil && !scope.MachineOnly() && !deleting {
		log.Info("VultrCluster is not available yet.")
		return ctrl.Result{}, nil
	}

	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:       r.Client,
		Logger:       log,
		Cluster:      cluster,
		VultrCluster: vultrCluster,
	})
	if err != nil {
		return ctrl.Result{}, errors.Errorf("failed to create scope: %v", err)
	}

	machineScope, err := scope.NewBareMetalMachineScope(scope.BareMetalMachineScopeParams{
		Client:                r.Client,
		Logger:                log,
		Cluster:               cluster,
		Machine:               machine,
		VultrCluster:          vultrCluster,
		VultrBareMetalMachine: bareMetalMachine,
	})
	if err != nil {
		return ctrl.Result{}, errors.Errorf("failed to create bare metal machine scope: %v", err)
	}

	defer func() {
		if err := machineScope.Close(ctx); err != nil && reterr == nil {
			reterr = err
		}
	}()

	if deleting {
		return r.reconcileDelete(ctx, machineScope, clusterScope)
	}

	return r.reconcileNormal(ctx, machineScope, clusterScope)
}

func (r *VultrBareMetalMachineReconciler) reconcileNormal(ctx context.Context, machineScope *scope.BareMetalMachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineScope.Info("Reconciling VultrBareMetalMachine")
	bareMetalMachine := machineScope.VultrBareMetalMachine

	if machineScope.HasTerminalFailure() {
		machineScope.Info("Machine infrastructure has failed, skipping reconciliation")
		return reconcile.Result{}, nil
	}

	controllerutil.AddFinalizer(bareMetalMachine, v1beta2.BareMetalMachineFinalizer)

	if !ptr.Deref(machineScope.Cluster.Status.Initialization.InfrastructureProvisioned, false) {
		machineScope.Info("Cluster infrastructure is not ready yet")
		machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.BareMetalWaitingForClusterInfrastructureReason, "Waiting for the cluster infrastructure to be provisioned")
		return reconcile.Result{}, nil
	}

	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap data secret reference is not yet available")
		machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.BareMetalWaitingForBootstrapDataReason, "Waiting for the bootstrap data secret")
		return reconcile.Result{}, nil
	}

	svc := services.NewService(ctx, clusterScope)

	server, err := r.findOrCreateServer(ctx, svc, machineScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if server == nil {
		// Reported missing; look again rather than acting on one answer.
		return reconcile.Result{RequeueAfter: bareMetalNotFoundRequeue}, nil
	}

	machineScope.SetProviderID(server.ID)
	machineScope.SetServerStatus(v1beta2.SubscriptionStatus(server.Status))
	machineScope.SetHardware(server.CPUCount, server.RAM, server.Disk)

	switch v1beta2.SubscriptionStatus(server.Status) {
	case v1beta2.SubscriptionStatusPending:
		machineScope.Info("Bare metal server is provisioning", "server-id", server.ID)
		machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.BareMetalServerPendingReason, "Bare metal server is provisioning")
		return reconcile.Result{RequeueAfter: bareMetalPendingRequeue}, nil

	case v1beta2.SubscriptionStatusActive:

	default:
		msg := fmt.Sprintf("bare metal server status %q is unexpected", server.Status)
		machineScope.Info("Bare metal server status is unexpected", "server-id", server.ID, "status", server.Status)
		r.Recorder.Eventf(bareMetalMachine, nil, corev1.EventTypeWarning, "UnexpectedStatus", "Failed", msg)
		machineScope.SetTerminalFailure(v1beta2.BareMetalUnexpectedStatusReason, msg)
		return reconcile.Result{}, nil
	}

	var vpcInfo *govultr.VPCInfo
	if vpcID := bareMetalMachine.Spec.VPCID; vpcID != "" {
		attached, info, err := svc.EnsureBareMetalVPC(server.ID, vpcID)
		if err != nil {
			if services.IsTerminalVPCAttachError(err) {
				r.Recorder.Eventf(bareMetalMachine, nil, corev1.EventTypeWarning, "VPCAttachFailed", "Failed", err.Error())
				machineScope.SetTerminalFailure(v1beta2.BareMetalVPCAttachFailedReason, err.Error())
				return reconcile.Result{}, nil
			}
			return reconcile.Result{}, err
		}
		if !attached {
			machineScope.Info("Waiting for VPC attachment", "server-id", server.ID, "vpc-id", vpcID)
			machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.BareMetalVPCAttachRequestedReason, "Waiting for VPC attachment")
			return reconcile.Result{RequeueAfter: bareMetalVPCRequeue}, nil
		}
		vpcInfo = info
		machineScope.SetVPC(&v1beta2.VultrBareMetalVPCStatus{
			ID:         info.ID,
			IPAddress:  info.IPAddress,
			MACAddress: info.MacAddress,
		})
	}

	if machineScope.IsControlPlane() {
		lbID := clusterScope.APIServerLoadBalancerID()
		if lbID == "" {
			machineScope.Info("Waiting for the load balancer id to be known")
			machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.BareMetalWaitingForLoadBalancerReason, "Waiting for the load balancer to be created")
			return reconcile.Result{RequeueAfter: bareMetalLoadBalancerRequeue}, nil
		}
		member, err := svc.EnsureLoadBalancerMember(lbID, server.ID)
		if err != nil {
			r.Recorder.Eventf(bareMetalMachine, nil, corev1.EventTypeWarning, "LoadBalancerMemberFailed", "Failed",
				"Failed to add bare metal server %s to load balancer %s: %v", server.ID, lbID, err)
			return reconcile.Result{}, err
		}
		if !member {
			machineScope.Info("Waiting for the load balancer to become active", "load-balancer-id", lbID)
			machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.BareMetalWaitingForLoadBalancerReason, "Waiting for the load balancer to become active")
			return reconcile.Result{RequeueAfter: bareMetalLoadBalancerRequeue}, nil
		}
	}

	machineScope.SetAddresses(services.GetBareMetalServerAddresses(server, vpcInfo))

	machineScope.SetReadyCondition(metav1.ConditionTrue, v1beta2.BareMetalServerActiveReason, "Bare metal server is active")
	machineScope.SetReady()
	machineScope.SetProvisioned()

	return reconcile.Result{}, nil
}

// findOrCreateServer returns the bare metal server for the machine, locating it by
// providerID, then by tags (a server whose id was never persisted), and finally
// creating it. It returns nil, nil after recording a terminal failure.
func (r *VultrBareMetalMachineReconciler) findOrCreateServer(ctx context.Context, svc *services.Service, machineScope *scope.BareMetalMachineScope) (*govultr.BareMetalServer, error) {
	bareMetalMachine := machineScope.VultrBareMetalMachine

	if id := machineScope.GetServerID(); id != "" {
		server, err := svc.GetBareMetalServer(id)
		if err != nil {
			return nil, err
		}
		if server == nil {
			msg := fmt.Sprintf("bare metal server %s not found, checking again", id)
			if !conditions.IsFalse(bareMetalMachine, v1beta2.BareMetalMachineReadyCondition) || conditions.GetReason(bareMetalMachine, v1beta2.BareMetalMachineReadyCondition) != v1beta2.BareMetalServerNotFoundReason {
				r.Recorder.Eventf(bareMetalMachine, nil, corev1.EventTypeWarning, "ServerNotFound", "NotFound", msg)
			}
			machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.BareMetalServerNotFoundReason, msg)
			return nil, nil
		}
		return server, nil
	}

	server, err := svc.FindBareMetalServerByTags(machineScope.Name(), machineScope.Role())
	if err != nil {
		return nil, err
	}
	if server != nil {
		r.Recorder.Eventf(bareMetalMachine, nil, corev1.EventTypeNormal, "ServerAdopted", "Adopted",
			"Adopted existing bare metal server %s", server.ID)
		return server, nil
	}

	server, err = svc.CreateBareMetalServer(ctx, machineScope)
	if err != nil {
		r.Recorder.Eventf(bareMetalMachine, nil, corev1.EventTypeWarning, "ServerCreationFailed", "Failed",
			"Failed to create bare metal server: %v", err)
		machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.BareMetalServerCreationFailedReason, err.Error())
		return nil, err
	}
	r.Recorder.Eventf(bareMetalMachine, nil, corev1.EventTypeNormal, "ServerCreated", "Created",
		"Created bare metal server %s (%s)", server.ID, server.Plan)

	return server, nil
}

// vultrAPIErrorStatus extracts the HTTP status from a Vultr API error. govultr
// surfaces API failures as errors whose text is the JSON body returned by the
// API (for example {"error":"...","status":422}); transport errors carry no status.

func (r *VultrBareMetalMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.BareMetalMachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineScope.Info("Reconciling delete VultrBareMetalMachine")
	bareMetalMachine := machineScope.VultrBareMetalMachine
	machineScope.SetReadyCondition(metav1.ConditionFalse, v1beta2.BareMetalDeletingReason, "Deleting the bare metal server")

	svc := services.NewService(ctx, clusterScope)

	// Delete by ID when there is one, then list the servers owned by this
	// machine; the finalizer goes only once that list is empty.
	id := machineScope.GetServerID()
	deleted := false
	if id != "" {
		if machineScope.IsControlPlane() {
			if err := svc.RemoveLoadBalancerMember(clusterScope.APIServerLoadBalancerID(), id); err != nil {
				return reconcile.Result{}, err
			}
		}
		var err error
		if deleted, err = svc.DeleteBareMetalServer(id); err != nil {
			return reconcile.Result{}, err
		}
		if deleted {
			r.Recorder.Eventf(bareMetalMachine, nil, corev1.EventTypeNormal, "ServerDeleted", "Deleted", "Deleted bare metal server %s", id)
		}
	}

	remaining, err := svc.FindBareMetalServerByTags(machineScope.Name(), machineScope.Role())
	if err != nil {
		return reconcile.Result{}, err
	}
	if remaining != nil {
		if remaining.ID == id {
			machineScope.Info("Server still listed after delete, checking again", "server-id", id)
			return reconcile.Result{RequeueAfter: bareMetalNotFoundRequeue}, nil
		}
		if _, err := svc.DeleteBareMetalServer(remaining.ID); err != nil {
			return reconcile.Result{}, err
		}
		r.Recorder.Eventf(bareMetalMachine, nil, corev1.EventTypeNormal, "ServerDeleted", "Deleted", "Deleted bare metal server %s", remaining.ID)
		return reconcile.Result{RequeueAfter: bareMetalNotFoundRequeue}, nil
	}

	if !deleted {
		r.Recorder.Eventf(bareMetalMachine, nil, corev1.EventTypeNormal, "NoServerFound", "NotFound", "No bare metal server to delete")
	}
	controllerutil.RemoveFinalizer(bareMetalMachine, v1beta2.BareMetalMachineFinalizer)
	return reconcile.Result{}, nil
}

func (r *VultrBareMetalMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, _ controller.Options) error {
	clusterToObjectFunc, err := util.ClusterToTypedObjectsMapper(r.Client, &v1beta2.VultrBareMetalMachineList{}, mgr.GetScheme())
	if err != nil {
		return errors.Wrapf(err, "failed to create mapper for Cluster to VultrBareMetalMachines")
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta2.VultrBareMetalMachine{}).
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(v1beta2.GroupVersion.WithKind("VultrBareMetalMachine"))),
		).
		Watches(
			&v1beta2.VultrCluster{},
			handler.EnqueueRequestsFromMapFunc(r.VultrClusterToVultrBareMetalMachines(ctx)),
		).
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(clusterToObjectFunc),
			builder.WithPredicates(predicates.ClusterPausedTransitionsOrInfrastructureProvisioned(mgr.GetScheme(), ctrl.LoggerFrom(ctx))),
		).
		Complete(r)
}

// VultrClusterToVultrBareMetalMachines enqueues the VultrBareMetalMachines of the cluster owning a VultrCluster.
func (r *VultrBareMetalMachineReconciler) VultrClusterToVultrBareMetalMachines(ctx context.Context) handler.MapFunc {
	log := ctrl.LoggerFrom(ctx)
	return func(ctx context.Context, o client.Object) []ctrl.Request {
		result := []ctrl.Request{}

		c, ok := o.(*v1beta2.VultrCluster)
		if !ok {
			log.Error(errors.Errorf("expected a VultrCluster but got a %T", o), "failed to get VultrBareMetalMachines for VultrCluster")
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
			if m.Spec.InfrastructureRef.Name == "" || m.Spec.InfrastructureRef.Kind != "VultrBareMetalMachine" {
				continue
			}
			result = append(result, ctrl.Request{NamespacedName: client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.InfrastructureRef.Name}})
		}

		return result
	}
}
