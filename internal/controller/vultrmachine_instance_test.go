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
	"errors"
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"github.com/vultr/govultr/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta2"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/cluster-api-provider-vultr/cloud/services"
	"github.com/vultr/cluster-api-provider-vultr/cloud/vultrerrors"
)

const (
	orphanMachineName = "orphan-test-0"
	orphanClusterTag  = "capvultr:cluster-uid:uid-o"
	orphanNameTag     = "name:" + orphanMachineName
	orphanVultrErr    = `{"error":"(REMOTE) Server failed to include a response","status":400}`
	orphanExistingID  = "existing"
	orphanTaggedID    = "tagged"
	orphanActive      = "active"
	orphanUntaggedID  = "untagged"
)

// fakeInstances implements the subset of govultr.InstanceService the reconciler uses.
type fakeInstances struct {
	govultr.InstanceService

	instances map[string]*govultr.Instance
	listed    []govultr.Instance

	getEmpty   bool
	getErr     error
	getCode    int
	createErr  error
	deleteErr  error
	deleteCode int
	listErr    error

	created []*govultr.InstanceCreateReq
	deleted []string
	gets    int
	lists   int
}

func (f *fakeInstances) Get(ctx context.Context, id string) (*govultr.Instance, *http.Response, error) {
	f.gets++
	if f.getErr != nil {
		vultrerrors.Record(ctx, f.getCode)
		return nil, &http.Response{StatusCode: f.getCode}, f.getErr
	}
	if f.getEmpty {
		vultrerrors.Record(ctx, http.StatusOK)
		return nil, &http.Response{StatusCode: http.StatusOK}, nil
	}
	i, ok := f.instances[id]
	if !ok {
		vultrerrors.Record(ctx, http.StatusNotFound)
		return nil, nil, errors.New("error returned from api : instance not found")
	}
	vultrerrors.Record(ctx, http.StatusOK)
	return i, &http.Response{StatusCode: http.StatusOK}, nil
}

func (f *fakeInstances) Create(_ context.Context, req *govultr.InstanceCreateReq) (*govultr.Instance, *http.Response, error) {
	f.created = append(f.created, req)
	if f.createErr != nil {
		return nil, &http.Response{StatusCode: http.StatusInternalServerError}, f.createErr
	}
	i := &govultr.Instance{ID: "created", Status: "pending", Label: req.Label, Hostname: req.Hostname, Tags: req.Tags, MainIP: "203.0.113.10"}
	if f.instances == nil {
		f.instances = map[string]*govultr.Instance{}
	}
	f.instances[i.ID] = i
	return i, &http.Response{StatusCode: http.StatusAccepted}, nil
}

func (f *fakeInstances) Delete(ctx context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	if f.deleteErr != nil {
		vultrerrors.Record(ctx, f.deleteCode)
		return f.deleteErr
	}
	if _, ok := f.instances[id]; !ok {
		vultrerrors.Record(ctx, http.StatusNotFound)
		return errors.New("error returned from api : Instance not found")
	}
	vultrerrors.Record(ctx, http.StatusNoContent)
	delete(f.instances, id)
	return nil
}

func (f *fakeInstances) List(_ context.Context, _ *govultr.ListOptions) ([]govultr.Instance, *govultr.Meta, *http.Response, error) {
	f.lists++
	if f.listErr != nil {
		return nil, nil, nil, f.listErr
	}
	return f.listed, &govultr.Meta{Links: &govultr.Links{}}, &http.Response{StatusCode: http.StatusOK}, nil
}

type fakeSSHKeys struct{ govultr.SSHKeyService }

// fakeLB serves the control plane load balancer; nil lb means "not created yet".
type fakeLB struct {
	govultr.LoadBalancerService
	lb         *govultr.LoadBalancer
	updateErr  error
	updateCode int
	updates    int
}

func (f *fakeLB) Get(_ context.Context, _ string) (*govultr.LoadBalancer, *http.Response, error) {
	return f.lb, &http.Response{StatusCode: http.StatusOK}, nil
}

func (f *fakeLB) Update(ctx context.Context, _ string, req *govultr.LoadBalancerReq) error {
	f.updates++
	if f.updateErr != nil {
		vultrerrors.Record(ctx, f.updateCode)
		return f.updateErr
	}
	f.lb.Instances = req.Instances
	return nil
}

func (f *fakeSSHKeys) Get(_ context.Context, id string) (*govultr.SSHKey, *http.Response, error) {
	return &govultr.SSHKey{ID: id}, &http.Response{StatusCode: http.StatusOK}, nil
}

type orphanFixture struct {
	r            *VultrMachineReconciler
	svc          *services.Service
	clusterScope *scope.ClusterScope
	machineScope *scope.MachineScope
	vultrMachine *infrav1.VultrMachine
}

// newOrphanFixture builds a reconciler, service and scopes for a worker VultrMachine.
func newOrphanFixture(t *testing.T, f *fakeInstances, providerID string, deleting bool) *orphanFixture {
	t.Helper()
	return newFixture(t, f, providerID, deleting, nil)
}

// newFixture is newOrphanFixture with an optional control plane load balancer;
// a non-nil lb makes the Machine a control plane machine.
func newFixture(t *testing.T, f *fakeInstances, providerID string, deleting bool, lb *fakeLB) *orphanFixture {
	t.Helper()
	t.Setenv("VULTR_API_KEY", "test")

	sch := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{corev1.AddToScheme, clusterv1.AddToScheme, infrav1.AddToScheme} {
		if err := add(sch); err != nil {
			t.Fatal(err)
		}
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "bootstrap", Namespace: testNamespace},
		Data:       map[string][]byte{"value": []byte("#cloud-config\nruncmd:\n  - kubeadm join\n")},
	}
	cluster := &clusterv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "orphan", Namespace: testNamespace, UID: types.UID("uid-o")}}
	vultrCluster := &infrav1.VultrCluster{ObjectMeta: metav1.ObjectMeta{Name: "orphan", Namespace: testNamespace}}
	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{Name: orphanMachineName, Namespace: testNamespace},
		Spec:       clusterv1.MachineSpec{Bootstrap: clusterv1.Bootstrap{DataSecretName: ptr.To("bootstrap")}},
	}
	apiClients := scope.VultrAPIClients{Instances: f, SSHKeys: &fakeSSHKeys{}}
	if lb != nil {
		machine.Labels = map[string]string{clusterv1.MachineControlPlaneLabel: ""}
		apiClients.LoadBalancers = lb
		if lb.lb != nil {
			vultrCluster.Status.Network.APIServerLoadbalancersRef.ResourceID = "lb-1"
		}
	}
	vultrMachine := &infrav1.VultrMachine{
		ObjectMeta: metav1.ObjectMeta{Name: orphanMachineName, Namespace: testNamespace, Finalizers: []string{infrav1.MachineFinalizer}},
		Spec:       infrav1.VultrMachineSpec{Region: testRegion, PlanID: "vc2-2c-4gb", Snapshot: "snap-1"},
	}
	if providerID != "" {
		vultrMachine.Spec.ProviderID = ptr.To("vultr://" + providerID)
	}
	if deleting {
		vultrMachine.DeletionTimestamp = ptr.To(metav1.Now())
	}

	c := fake.NewClientBuilder().WithScheme(sch).WithObjects(secret, cluster, vultrCluster, machine, vultrMachine).Build()

	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		VultrAPIClients: apiClients,
		Client:          c,
		Logger:          logr.Discard(),
		Cluster:         cluster,
		VultrCluster:    vultrCluster,
	})
	if err != nil {
		t.Fatal(err)
	}
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:       c,
		Logger:       logr.Discard(),
		Machine:      machine,
		Cluster:      cluster,
		VultrCluster: vultrCluster,
		VultrMachine: vultrMachine,
	})
	if err != nil {
		t.Fatal(err)
	}

	return &orphanFixture{
		r:            &VultrMachineReconciler{Client: c, Recorder: events.NewFakeRecorder(64)},
		svc:          services.NewService(context.Background(), clusterScope),
		clusterScope: clusterScope,
		machineScope: machineScope,
		vultrMachine: vultrMachine,
	}
}

func readyReason(vm *infrav1.VultrMachine) string {
	c := conditions.Get(vm, infrav1.MachineReadyCondition)
	if c == nil {
		return ""
	}
	return c.Reason
}

func TestFindOrCreateInstance(t *testing.T) {
	existing := &govultr.Instance{ID: orphanExistingID, Label: orphanMachineName, Status: orphanActive}
	tagged := govultr.Instance{ID: orphanTaggedID, Label: orphanMachineName, Hostname: orphanMachineName, Tags: []string{orphanClusterTag, orphanNameTag}}

	tests := []struct {
		name         string
		fake         *fakeInstances
		providerID   string
		wantID       string
		wantErr      bool
		wantCreates  int
		wantReason   string
		wantTerminal bool
	}{
		{
			name:       "providerID set and instance exists: use it",
			fake:       &fakeInstances{instances: map[string]*govultr.Instance{orphanExistingID: existing}},
			providerID: orphanExistingID,
			wantID:     orphanExistingID,
		},
		{
			name:       "providerID set and Vultr says 404: report it and re-check, never re-create",
			fake:       &fakeInstances{},
			providerID: orphanExistingID,
			wantReason: infrav1.InstanceNotFoundReason,
		},
		{
			name:       "providerID set and Vultr answers 200 without an instance: retry, never re-create",
			fake:       &fakeInstances{getEmpty: true},
			providerID: orphanExistingID,
			wantErr:    true,
		},
		{
			name:       "providerID set and Vultr answers 400: retry, never re-create",
			fake:       &fakeInstances{getErr: errors.New(orphanVultrErr), getCode: http.StatusBadRequest},
			providerID: orphanExistingID,
			wantErr:    true,
		},
		{
			name:   "no providerID and a tagged instance exists: adopt it, no create",
			fake:   &fakeInstances{listed: []govultr.Instance{tagged}},
			wantID: orphanTaggedID,
		},
		{
			name:        "no providerID and nothing to adopt: create once",
			fake:        &fakeInstances{},
			wantID:      "created",
			wantCreates: 1,
		},
		{
			name:        "create fails: retryable, not terminal",
			fake:        &fakeInstances{createErr: errors.New(`{"error":"Internal server error.","status":500}`)},
			wantErr:     true,
			wantCreates: 1,
			wantReason:  infrav1.InstanceCreationFailedReason,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fx := newOrphanFixture(t, tt.fake, tt.providerID, false)
			got, err := fx.r.findOrCreateInstance(fx.svc, fx.machineScope)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			gotID := ""
			if got != nil {
				gotID = got.ID
			}
			if gotID != tt.wantID {
				t.Errorf("instance = %q, want %q", gotID, tt.wantID)
			}
			if len(tt.fake.created) != tt.wantCreates {
				t.Errorf("creates = %d, want %d", len(tt.fake.created), tt.wantCreates)
			}
			if tt.wantReason != "" && readyReason(fx.vultrMachine) != tt.wantReason {
				t.Errorf("Ready reason = %q, want %q", readyReason(fx.vultrMachine), tt.wantReason)
			}
			if fx.machineScope.HasTerminalFailure() != tt.wantTerminal {
				t.Errorf("terminal = %v, want %v", fx.machineScope.HasTerminalFailure(), tt.wantTerminal)
			}
		})
	}
}

func TestReconcileDeleteInstance(t *testing.T) {
	existing := &govultr.Instance{ID: orphanExistingID, Label: orphanMachineName, Status: orphanActive}
	tagged := govultr.Instance{ID: orphanTaggedID, Label: orphanMachineName, Hostname: orphanMachineName, Tags: []string{orphanClusterTag, orphanNameTag}}
	untagged := govultr.Instance{ID: orphanUntaggedID, Label: orphanMachineName, Hostname: orphanMachineName}
	stillListed := govultr.Instance{ID: orphanExistingID, Label: orphanMachineName, Hostname: orphanMachineName, Tags: []string{orphanClusterTag, orphanNameTag}}

	tests := []struct {
		name          string
		fake          *fakeInstances
		providerID    string
		wantErr       bool
		wantRequeue   bool
		wantDeleted   []string
		wantFinalizer bool
		secondPass    bool // run again with an empty list, as the next reconcile would
	}{
		{
			name:        "providerID set: delete by ID, list is empty, drop the finalizer",
			fake:        &fakeInstances{instances: map[string]*govultr.Instance{orphanExistingID: existing}},
			providerID:  orphanExistingID,
			wantDeleted: []string{orphanExistingID},
		},
		{
			name:        "providerID set, delete says 404 and the list agrees: drop the finalizer",
			fake:        &fakeInstances{},
			providerID:  orphanExistingID,
			wantDeleted: []string{orphanExistingID},
		},
		{
			name:          "providerID set, delete says 404 but the list still shows it: keep the finalizer, re-check",
			fake:          &fakeInstances{listed: []govultr.Instance{stillListed}},
			providerID:    orphanExistingID,
			wantRequeue:   true,
			wantDeleted:   []string{orphanExistingID},
			wantFinalizer: true,
		},
		{
			name:          "providerID set, delete accepted but the list lags: keep the finalizer, re-check",
			fake:          &fakeInstances{instances: map[string]*govultr.Instance{orphanExistingID: existing}, listed: []govultr.Instance{stillListed}},
			providerID:    orphanExistingID,
			wantRequeue:   true,
			wantDeleted:   []string{orphanExistingID},
			wantFinalizer: true,
		},
		{
			name:          "providerID set, delete answered 200 without a body: keep the finalizer and retry",
			fake:          &fakeInstances{deleteErr: errors.New("error returned from api : (REMOTE) Server failed to include a response"), deleteCode: 200},
			providerID:    orphanExistingID,
			wantErr:       true,
			wantDeleted:   []string{orphanExistingID},
			wantFinalizer: true,
		},
		{
			name:          "providerID set, delete failed 500: keep the finalizer and retry",
			fake:          &fakeInstances{deleteErr: errors.New("error returned from api : Internal server error."), deleteCode: 500},
			providerID:    orphanExistingID,
			wantErr:       true,
			wantDeleted:   []string{orphanExistingID},
			wantFinalizer: true,
		},
		{
			name:          "providerID set, list fails after the delete: keep the finalizer and retry",
			fake:          &fakeInstances{instances: map[string]*govultr.Instance{orphanExistingID: existing}, listErr: errors.New("error returned from api : Internal server error.")},
			providerID:    orphanExistingID,
			wantErr:       true,
			wantDeleted:   []string{orphanExistingID},
			wantFinalizer: true,
		},
		{
			name:        "no providerID but a tagged instance exists: delete it, then drop the finalizer next pass",
			fake:        &fakeInstances{instances: map[string]*govultr.Instance{orphanTaggedID: &tagged}, listed: []govultr.Instance{tagged}},
			wantRequeue: true,
			wantDeleted: []string{orphanTaggedID},
			secondPass:  true,
		},
		{
			name:        "no providerID but an untagged instance with our label and hostname exists: delete it",
			fake:        &fakeInstances{instances: map[string]*govultr.Instance{orphanUntaggedID: &untagged}, listed: []govultr.Instance{untagged}},
			wantRequeue: true,
			wantDeleted: []string{orphanUntaggedID},
			secondPass:  true,
		},
		{
			name: "no providerID and no instance: drop the finalizer",
			fake: &fakeInstances{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fx := newOrphanFixture(t, tt.fake, tt.providerID, true)
			res, err := fx.r.reconcileDelete(context.Background(), fx.machineScope, fx.clusterScope)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if (res.RequeueAfter > 0) != tt.wantRequeue {
				t.Errorf("requeue = %v, want %v", res.RequeueAfter, tt.wantRequeue)
			}
			if len(tt.fake.deleted) != len(tt.wantDeleted) || (len(tt.wantDeleted) > 0 && tt.fake.deleted[0] != tt.wantDeleted[0]) {
				t.Errorf("deleted = %v, want %v", tt.fake.deleted, tt.wantDeleted)
			}
			if controllerutil.ContainsFinalizer(fx.vultrMachine, infrav1.MachineFinalizer) != (tt.wantFinalizer || tt.secondPass) {
				t.Errorf("finalizer present = %v after first pass", !tt.wantFinalizer)
			}
			if tt.secondPass {
				tt.fake.listed = nil
				if _, err := fx.r.reconcileDelete(context.Background(), fx.machineScope, fx.clusterScope); err != nil {
					t.Fatal(err)
				}
				if controllerutil.ContainsFinalizer(fx.vultrMachine, infrav1.MachineFinalizer) {
					t.Error("finalizer must be removed once the list is empty")
				}
			}
		})
	}
}

func TestReconcileNormal(t *testing.T) {
	active := &govultr.Instance{ID: orphanExistingID, Label: orphanMachineName, Status: orphanActive, MainIP: "203.0.113.10"}

	tests := []struct {
		name         string
		fake         *fakeInstances
		providerID   string
		clusterReady bool
		noBootstrap  bool
		terminal     bool
		wantErr      bool
		wantRequeue  bool
		wantReason   string
		wantStatus   metav1.ConditionStatus
		wantProvider string
		wantCreates  int
		lb           *fakeLB
		wantMember   bool
	}{
		{
			name:         "terminal failure: nothing is called",
			fake:         &fakeInstances{},
			providerID:   orphanExistingID,
			clusterReady: true,
			terminal:     true,
			wantReason:   infrav1.InstanceUnexpectedStatusReason,
			wantStatus:   metav1.ConditionFalse,
			wantProvider: "vultr://" + orphanExistingID,
		},
		{
			name:       "cluster infrastructure not ready: waiting condition, nothing is called",
			fake:       &fakeInstances{},
			wantReason: infrav1.InstanceWaitingForClusterInfrastructureReason,
			wantStatus: metav1.ConditionFalse,
		},
		{
			name:         "bootstrap data missing: waiting condition, nothing is called",
			fake:         &fakeInstances{},
			clusterReady: true,
			noBootstrap:  true,
			wantReason:   infrav1.InstanceWaitingForBootstrapDataReason,
			wantStatus:   metav1.ConditionFalse,
		},
		{
			name:         "no providerID: create, persist the ID, pending requeue",
			fake:         &fakeInstances{},
			clusterReady: true,
			wantRequeue:  true,
			wantReason:   infrav1.InstancePendingReason,
			wantStatus:   metav1.ConditionUnknown,
			wantProvider: "vultr://created",
			wantCreates:  1,
		},
		{
			name:         "providerID set and active: ready, no create",
			fake:         &fakeInstances{instances: map[string]*govultr.Instance{orphanExistingID: active}},
			providerID:   orphanExistingID,
			clusterReady: true,
			wantReason:   infrav1.InstanceActiveReason,
			wantStatus:   metav1.ConditionTrue,
			wantProvider: "vultr://" + orphanExistingID,
		},
		{
			name:         "control plane, load balancer not created yet: wait, no error",
			fake:         &fakeInstances{instances: map[string]*govultr.Instance{orphanExistingID: active}},
			lb:           &fakeLB{},
			providerID:   orphanExistingID,
			clusterReady: true,
			wantRequeue:  true,
			wantReason:   infrav1.InstanceWaitingForLoadBalancerReason,
			wantStatus:   metav1.ConditionFalse,
			wantProvider: "vultr://" + orphanExistingID,
		},
		{
			name:         "control plane, load balancer pending: wait, no update",
			fake:         &fakeInstances{instances: map[string]*govultr.Instance{orphanExistingID: active}},
			lb:           &fakeLB{lb: &govultr.LoadBalancer{Status: "pending"}},
			providerID:   orphanExistingID,
			clusterReady: true,
			wantRequeue:  true,
			wantReason:   infrav1.InstanceWaitingForLoadBalancerReason,
			wantStatus:   metav1.ConditionFalse,
			wantProvider: "vultr://" + orphanExistingID,
		},
		{
			name:         "control plane, nodes still activating (422): wait, no error",
			fake:         &fakeInstances{instances: map[string]*govultr.Instance{orphanExistingID: active}},
			lb:           &fakeLB{lb: &govultr.LoadBalancer{Status: orphanActive}, updateErr: errors.New("error returned from api : Load Balancer Nodes Still Activating..."), updateCode: 422},
			providerID:   orphanExistingID,
			clusterReady: true,
			wantRequeue:  true,
			wantReason:   infrav1.InstanceWaitingForLoadBalancerReason,
			wantStatus:   metav1.ConditionFalse,
			wantProvider: "vultr://" + orphanExistingID,
		},
		{
			name:         "control plane, load balancer active: member added, ready",
			fake:         &fakeInstances{instances: map[string]*govultr.Instance{orphanExistingID: active}},
			lb:           &fakeLB{lb: &govultr.LoadBalancer{Status: orphanActive}},
			providerID:   orphanExistingID,
			clusterReady: true,
			wantReason:   infrav1.InstanceActiveReason,
			wantStatus:   metav1.ConditionTrue,
			wantProvider: "vultr://" + orphanExistingID,
			wantMember:   true,
		},
		{
			name:         "providerID set and gone: re-check later, no create, ID kept",
			fake:         &fakeInstances{},
			providerID:   orphanExistingID,
			clusterReady: true,
			wantRequeue:  true,
			wantReason:   infrav1.InstanceNotFoundReason,
			wantStatus:   metav1.ConditionFalse,
			wantProvider: "vultr://" + orphanExistingID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fx := newFixture(t, tt.fake, tt.providerID, false, tt.lb)
			if tt.clusterReady {
				conditions.Set(fx.machineScope.Cluster, metav1.Condition{Type: clusterv1.InfrastructureReadyCondition, Status: metav1.ConditionTrue, Reason: "Ready"})
			}
			if tt.noBootstrap {
				fx.machineScope.Machine.Spec.Bootstrap.DataSecretName = nil
			}
			if tt.terminal {
				fx.machineScope.SetTerminalFailure(infrav1.InstanceUnexpectedStatusReason, "status \"locked\" is unexpected")
			}

			res, err := fx.r.reconcileNormal(context.Background(), fx.machineScope, fx.clusterScope)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if (res.RequeueAfter > 0) != tt.wantRequeue {
				t.Errorf("requeue = %v, want %v", res.RequeueAfter, tt.wantRequeue)
			}
			if c := conditions.Get(fx.vultrMachine, infrav1.MachineReadyCondition); c == nil || c.Reason != tt.wantReason || c.Status != tt.wantStatus {
				t.Errorf("Ready = %+v, want %s/%s", c, tt.wantStatus, tt.wantReason)
			}
			if len(tt.fake.created) != tt.wantCreates {
				t.Errorf("creates = %d, want %d", len(tt.fake.created), tt.wantCreates)
			}
			if tt.lb != nil && tt.lb.lb != nil {
				isMember := false
				for _, id := range tt.lb.lb.Instances {
					isMember = isMember || id == orphanExistingID
				}
				if isMember != tt.wantMember {
					t.Errorf("load balancer member = %v, want %v (updates=%d)", isMember, tt.wantMember, tt.lb.updates)
				}
			}
			if tt.terminal || tt.wantReason == infrav1.InstanceWaitingForClusterInfrastructureReason || tt.wantReason == infrav1.InstanceWaitingForBootstrapDataReason {
				if tt.fake.gets+tt.fake.lists != 0 {
					t.Errorf("Vultr was called %d times, want 0", tt.fake.gets+tt.fake.lists)
				}
			}
			// The providerID must be on the API server, not just in memory.
			stored := &infrav1.VultrMachine{}
			if err := fx.r.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: orphanMachineName}, stored); err != nil {
				t.Fatal(err)
			}
			got := ""
			if stored.Spec.ProviderID != nil {
				got = *stored.Spec.ProviderID
			}
			if got != tt.wantProvider {
				t.Errorf("persisted providerID = %q, want %q", got, tt.wantProvider)
			}
		})
	}
}
