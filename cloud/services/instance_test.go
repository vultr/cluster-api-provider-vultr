/*
Copyright 2020 The Kubernetes Authors.

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

package services

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/vultr/govultr/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta2"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
)

const (
	testVMMachineName = "test-vm-0"
	testVMPlan        = "vc2-2c-4gb"
	testVMClusterTag  = "sigs-k8s-io:capvultr:test:uid-1:node"
	testVMNameTag     = "name:" + testVMMachineName
	testVMCluster     = "test"
	testVMPending     = "pending"
	testVMExistingID  = "i-1"
	testVMTaggedID    = "tagged"

	// emptyBodyErr is what Vultr returns in place of an instance during outages.
	emptyBodyErr = `{"error":"(REMOTE) Server failed to include a response","status":400}`
)

// fakeInstanceService implements the subset of govultr.InstanceService the
// provider uses; the embedded interface panics on anything else.
type fakeInstanceService struct {
	govultr.InstanceService

	instances map[string]*govultr.Instance
	listPages [][]govultr.Instance

	// getEmpty makes Get answer 200 with no instance, as govultr does for an error body.
	getCode  int
	getErr   error
	getEmpty bool

	createErr   error
	createEmpty bool
	created     []*govultr.InstanceCreateReq
	deleted     []string
	deleteErr   error
	listCalls   int
}

func (f *fakeInstanceService) Get(_ context.Context, id string) (*govultr.Instance, *http.Response, error) {
	if f.getErr != nil {
		return nil, &http.Response{StatusCode: f.getCode}, f.getErr
	}
	if f.getEmpty {
		return nil, &http.Response{StatusCode: http.StatusOK}, nil
	}
	i, ok := f.instances[id]
	if !ok {
		return nil, &http.Response{StatusCode: http.StatusNotFound}, errors.New(`{"error":"Invalid instance.","status":404}`)
	}
	return i, &http.Response{StatusCode: http.StatusOK}, nil
}

func (f *fakeInstanceService) Create(_ context.Context, req *govultr.InstanceCreateReq) (*govultr.Instance, *http.Response, error) {
	f.created = append(f.created, req)
	if f.createErr != nil {
		return nil, &http.Response{StatusCode: http.StatusInternalServerError}, f.createErr
	}
	if f.createEmpty {
		return nil, &http.Response{StatusCode: http.StatusAccepted}, nil
	}
	i := &govultr.Instance{ID: "new-instance", Status: testVMPending, Plan: req.Plan, Region: req.Region, Label: req.Label, Hostname: req.Hostname, Tags: req.Tags}
	if f.instances == nil {
		f.instances = map[string]*govultr.Instance{}
	}
	f.instances[i.ID] = i
	return i, &http.Response{StatusCode: http.StatusAccepted}, nil
}

func (f *fakeInstanceService) Delete(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.instances, id)
	return nil
}

func (f *fakeInstanceService) List(_ context.Context, opts *govultr.ListOptions) ([]govultr.Instance, *govultr.Meta, *http.Response, error) {
	f.listCalls++
	page := 0
	if opts != nil && opts.Cursor != "" {
		page = int(opts.Cursor[0] - '0')
	}
	if page >= len(f.listPages) {
		return nil, &govultr.Meta{Links: &govultr.Links{}}, &http.Response{StatusCode: http.StatusOK}, nil
	}
	next := ""
	if page+1 < len(f.listPages) {
		next = string(rune('0' + page + 1))
	}
	return f.listPages[page], &govultr.Meta{Links: &govultr.Links{Next: next}}, &http.Response{StatusCode: http.StatusOK}, nil
}

// newInstanceTestService wires a Service and MachineScope for a worker VultrMachine.
func newInstanceTestService(t *testing.T, f *fakeInstanceService) (*Service, *scope.MachineScope) {
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
		Data:       map[string][]byte{"value": []byte(testBootstrapData)},
	}
	cluster := &clusterv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: testVMCluster, Namespace: testNamespace, UID: types.UID("uid-1")}}
	vultrCluster := &infrav1.VultrCluster{ObjectMeta: metav1.ObjectMeta{Name: testVMCluster, Namespace: testNamespace}}
	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{Name: testVMMachineName, Namespace: testNamespace},
		Spec:       clusterv1.MachineSpec{Bootstrap: clusterv1.Bootstrap{DataSecretName: ptr.To("bootstrap")}},
	}
	vultrMachine := &infrav1.VultrMachine{
		ObjectMeta: metav1.ObjectMeta{Name: testVMMachineName, Namespace: testNamespace},
		Spec:       infrav1.VultrMachineSpec{Region: testRegion, PlanID: testVMPlan, Snapshot: "snap-1"},
	}

	c := fake.NewClientBuilder().WithScheme(sch).WithObjects(secret, cluster, vultrCluster, machine, vultrMachine).Build()

	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		VultrAPIClients: scope.VultrAPIClients{Instances: f, SSHKeys: &fakeSSHKeyService{}},
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
	return NewService(context.Background(), clusterScope), machineScope
}

func TestGetInstance(t *testing.T) {
	existing := &govultr.Instance{ID: testVMExistingID, Label: testVMMachineName, Status: testActive}

	tests := []struct {
		name    string
		fake    *fakeInstanceService
		id      string
		want    *govultr.Instance
		wantErr bool
	}{
		{name: "empty id is nil, nil", fake: &fakeInstanceService{}, id: "", want: nil},
		{name: "found", fake: &fakeInstanceService{instances: map[string]*govultr.Instance{testVMExistingID: existing}}, id: testVMExistingID, want: existing},
		{name: "404 is nil, nil", fake: &fakeInstanceService{}, id: "gone", want: nil},
		{name: "400 is an error, not missing", fake: &fakeInstanceService{getCode: http.StatusBadRequest, getErr: errors.New(emptyBodyErr)}, id: testVMExistingID, wantErr: true},
		{name: "500 is an error", fake: &fakeInstanceService{getCode: http.StatusInternalServerError, getErr: errors.New(`{"error":"Internal server error.","status":500}`)}, id: testVMExistingID, wantErr: true},
		{name: "200 without an instance is an error, not missing", fake: &fakeInstanceService{getEmpty: true}, id: testVMExistingID, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newInstanceTestService(t, tt.fake)
			got, err := svc.GetInstance(tt.id)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("instance = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindInstanceForMachine(t *testing.T) {
	tagged := govultr.Instance{ID: testVMTaggedID, Label: testVMMachineName, Hostname: testVMMachineName, Tags: []string{testVMClusterTag, testVMNameTag}}
	untagged := govultr.Instance{ID: "untagged", Label: testVMMachineName, Hostname: testVMMachineName}
	otherCluster := govultr.Instance{ID: "other-cluster", Label: testVMMachineName, Hostname: testVMMachineName, Tags: []string{"sigs-k8s-io:capvultr:other:uid-2:node", testVMNameTag}}
	otherLabel := govultr.Instance{ID: "other-label", Label: testVMMachineName + "-x", Hostname: testVMMachineName + "-x", Tags: []string{testVMClusterTag, "name:" + testVMMachineName + "-x"}}
	handMade := govultr.Instance{ID: "hand-made", Label: testVMMachineName, Hostname: "something-else"}

	tests := []struct {
		name    string
		pages   [][]govultr.Instance
		wantID  string
		wantErr string
	}{
		{name: "none", pages: nil, wantID: ""},
		{name: "tagged match", pages: [][]govultr.Instance{{otherLabel, tagged}}, wantID: testVMTaggedID},
		{name: "untagged instance whose tag step failed is matched on label and hostname", pages: [][]govultr.Instance{{untagged}}, wantID: "untagged"},
		{name: "same label in another cluster is ignored", pages: [][]govultr.Instance{{otherCluster}}, wantID: ""},
		{name: "untagged instance with a different hostname is ignored", pages: [][]govultr.Instance{{handMade}}, wantID: ""},
		{name: "match on a later page", pages: [][]govultr.Instance{{otherLabel}, {tagged}}, wantID: testVMTaggedID},
		{name: "two candidates is an error naming both", pages: [][]govultr.Instance{{tagged}, {untagged}}, wantErr: testVMTaggedID + ", untagged"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeInstanceService{listPages: tt.pages}
			svc, ms := newInstanceTestService(t, f)
			got, err := svc.FindInstanceForMachine(ms.Name(), ms.Role())
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			gotID := ""
			if got != nil {
				gotID = got.ID
			}
			if gotID != tt.wantID {
				t.Fatalf("instance = %q, want %q", gotID, tt.wantID)
			}
			if f.listCalls != max(len(tt.pages), 1) {
				t.Fatalf("list calls = %d, want %d", f.listCalls, max(len(tt.pages), 1))
			}
		})
	}
}

func TestCreateInstance(t *testing.T) {
	t.Run("creates once with label, hostname and tags", func(t *testing.T) {
		f := &fakeInstanceService{}
		svc, ms := newInstanceTestService(t, f)
		got, err := svc.CreateInstance(ms)
		if err != nil {
			t.Fatal(err)
		}
		if len(f.created) != 1 {
			t.Fatalf("create calls = %d, want 1", len(f.created))
		}
		req := f.created[0]
		if req.Label != testVMMachineName || req.Hostname != testVMMachineName || req.Plan != testVMPlan || req.Region != testRegion {
			t.Errorf("unexpected request: %+v", req)
		}
		if !hasTags(req.Tags, testVMNameTag, testVMClusterTag) {
			t.Errorf("tags = %v, want %q and %q", req.Tags, testVMNameTag, testVMClusterTag)
		}
		if got == nil || got.ID != "new-instance" {
			t.Errorf("instance = %+v", got)
		}
	})

	t.Run("accepted create without an instance is an error, not a panic", func(t *testing.T) {
		f := &fakeInstanceService{createEmpty: true}
		svc, ms := newInstanceTestService(t, f)
		got, err := svc.CreateInstance(ms)
		if err == nil || got != nil {
			t.Fatalf("got %v, %v; want nil and an error", got, err)
		}
	})

	t.Run("create error is returned", func(t *testing.T) {
		f := &fakeInstanceService{createErr: errors.New(`{"error":"Internal server error.","status":500}`)}
		svc, ms := newInstanceTestService(t, f)
		if _, err := svc.CreateInstance(ms); err == nil {
			t.Fatal("want an error")
		}
	})

	t.Run("missing bootstrap data is an error, not a panic", func(t *testing.T) {
		f := &fakeInstanceService{}
		svc, ms := newInstanceTestService(t, f)
		ms.Machine.Spec.Bootstrap.DataSecretName = ptr.To("missing")
		if _, err := svc.CreateInstance(ms); err == nil {
			t.Fatal("want an error")
		}
		if len(f.created) != 0 {
			t.Fatal("must not call create without bootstrap data")
		}
	})
}
