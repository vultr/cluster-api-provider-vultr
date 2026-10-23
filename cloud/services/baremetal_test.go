package services

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
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta2"
	"github.com/vultr/cluster-api-provider-vultr/cloud/scope"
	"github.com/vultr/cluster-api-provider-vultr/cloud/vultrerrors"
)

// fakeBareMetalService implements the subset of govultr.BareMetalServerService the
// provider uses; the embedded interface panics on anything else.
type fakeBareMetalService struct {
	govultr.BareMetalServerService

	servers   map[string]*govultr.BareMetalServer
	listPages [][]govultr.BareMetalServer
	vpcInfo   map[string][]govultr.VPCInfo
	attachErr error

	created  []*govultr.BareMetalCreate
	attached []string
	deleted  []string
	getErr   error
	getCode  int
	// getEmpty and createEmpty answer 2xx with no server, as govultr does for an error body.
	getEmpty    bool
	createEmpty bool
}

func (f *fakeBareMetalService) Get(ctx context.Context, id string) (*govultr.BareMetalServer, *http.Response, error) {
	if f.getErr != nil {
		vultrerrors.Record(ctx, f.getCode)
		return nil, &http.Response{StatusCode: f.getCode}, f.getErr
	}
	if f.getEmpty {
		vultrerrors.Record(ctx, http.StatusOK)
		return nil, &http.Response{StatusCode: http.StatusOK}, nil
	}
	s, ok := f.servers[id]
	if !ok {
		vultrerrors.Record(ctx, http.StatusNotFound)
		return nil, nil, errors.New("error returned from api : bare metal not found")
	}
	vultrerrors.Record(ctx, http.StatusOK)
	return s, &http.Response{StatusCode: http.StatusOK}, nil
}

func (f *fakeBareMetalService) Create(_ context.Context, req *govultr.BareMetalCreate) (*govultr.BareMetalServer, *http.Response, error) {
	f.created = append(f.created, req)
	if f.createEmpty {
		return nil, &http.Response{StatusCode: http.StatusAccepted}, nil
	}
	s := &govultr.BareMetalServer{ID: "new-server", Status: "pending", Plan: req.Plan, Region: req.Region, Label: req.Label, Tags: req.Tags}
	if f.servers == nil {
		f.servers = map[string]*govultr.BareMetalServer{}
	}
	f.servers[s.ID] = s
	return s, &http.Response{StatusCode: http.StatusAccepted}, nil
}

func (f *fakeBareMetalService) Delete(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	delete(f.servers, id)
	return nil
}

func (f *fakeBareMetalService) List(_ context.Context, opts *govultr.ListOptions) ([]govultr.BareMetalServer, *govultr.Meta, *http.Response, error) {
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

func (f *fakeBareMetalService) ListVPCInfo(_ context.Context, id string) ([]govultr.VPCInfo, *http.Response, error) {
	return f.vpcInfo[id], &http.Response{StatusCode: http.StatusOK}, nil
}

func (f *fakeBareMetalService) AttachVPC(_ context.Context, id, vpcID string) error {
	if f.attachErr != nil {
		return f.attachErr
	}
	f.attached = append(f.attached, id+"/"+vpcID)
	return nil
}

type fakeSSHKeyService struct {
	govultr.SSHKeyService
}

func (f *fakeSSHKeyService) Get(_ context.Context, id string) (*govultr.SSHKey, *http.Response, error) {
	return &govultr.SSHKey{ID: id, Name: "key-" + id}, &http.Response{StatusCode: http.StatusOK}, nil
}

const (
	testBootstrapData = "#cloud-config\nruncmd:\n  - kubeadm join --config /run/kubeadm/kubeadm.yaml\n"
	testNamespace     = "default"
	testMachineName   = "test-bm-0"
	testRegion        = "ewr"
	testPlan          = "test-bare-metal-plan"
	testNameTag       = "name:" + testMachineName
	testActive        = "active"
)

func newTestService(t *testing.T, bm *fakeBareMetalService) (*Service, *scope.BareMetalMachineScope) {
	t.Helper()
	t.Setenv("VULTR_API_KEY", "test")

	sch := runtime.NewScheme()
	if err := corev1.AddToScheme(sch); err != nil {
		t.Fatal(err)
	}
	if err := clusterv1.AddToScheme(sch); err != nil {
		t.Fatal(err)
	}
	if err := infrav1.AddToScheme(sch); err != nil {
		t.Fatal(err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "bootstrap", Namespace: testNamespace},
		Data:       map[string][]byte{"value": []byte(testBootstrapData)},
	}
	cluster := &clusterv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: testNamespace, UID: types.UID("uid-1")}}
	vultrCluster := &infrav1.VultrCluster{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: testNamespace}}
	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{Name: testMachineName, Namespace: testNamespace},
		Spec:       clusterv1.MachineSpec{Bootstrap: clusterv1.Bootstrap{DataSecretName: ptr.To("bootstrap")}},
	}
	bmMachine := &infrav1.VultrBareMetalMachine{
		ObjectMeta: metav1.ObjectMeta{Name: testMachineName, Namespace: testNamespace},
		Spec: infrav1.VultrBareMetalMachineSpec{
			Region:         testRegion,
			PlanID:         testPlan,
			OSID:           2284,
			SSHKey:         []string{"key-1", ""},
			MdiskMode:      infrav1.BareMetalDiskModeRAID1,
			AdditionalTags: []string{"team:platform"},
		},
	}

	c := fake.NewClientBuilder().WithScheme(sch).WithObjects(secret, cluster, vultrCluster, machine, bmMachine).Build()

	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		VultrAPIClients: scope.VultrAPIClients{BareMetal: bm, SSHKeys: &fakeSSHKeyService{}},
		Client:          c,
		Logger:          logr.Discard(),
		Cluster:         cluster,
		VultrCluster:    vultrCluster,
	})
	if err != nil {
		t.Fatal(err)
	}
	machineScope, err := scope.NewBareMetalMachineScope(scope.BareMetalMachineScopeParams{
		Client:                c,
		Logger:                logr.Discard(),
		Machine:               machine,
		Cluster:               cluster,
		VultrCluster:          vultrCluster,
		VultrBareMetalMachine: bmMachine,
	})
	if err != nil {
		t.Fatal(err)
	}

	return NewService(context.Background(), clusterScope), machineScope
}

func TestBuildBareMetalCreateRequest(t *testing.T) {
	base := infrav1.VultrBareMetalMachineSpec{Region: testRegion, PlanID: testPlan, OSID: 2284}
	tags := []string{testVMClusterName, testNameTag}

	req := buildBareMetalCreateRequest(base, testMachineName, []string{"k1"}, "dXNlcg==", tags)
	if req.Region != testRegion || req.Plan != testPlan || req.OsID != 2284 {
		t.Errorf("unexpected placement/image fields: %+v", req)
	}
	if req.Label != testMachineName || req.Hostname != testMachineName || req.UserData != "dXNlcg==" || len(req.SSHKeyIDs) != 1 {
		t.Errorf("unexpected identity fields: %+v", req)
	}
	if req.EnableIPv6 == nil || !*req.EnableIPv6 {
		t.Error("EnableIPv6 must default to true")
	}
	if req.ActivationEmail == nil || *req.ActivationEmail {
		t.Error("ActivationEmail must be false")
	}
	if len(req.Tags) != 2 || req.MdiskMode != "" {
		t.Errorf("unexpected tags/mdisk: %+v", req)
	}

	custom := base
	custom.EnableIPv6 = ptr.To(false)
	custom.MdiskMode = infrav1.BareMetalDiskModeJBOD
	custom.AdditionalTags = []string{"extra"}
	req = buildBareMetalCreateRequest(custom, "n", nil, "", tags)
	if req.OsID != 2284 || *req.EnableIPv6 || req.MdiskMode != "jbod" {
		t.Errorf("custom request wrong: %+v", req)
	}
	if len(req.Tags) != 3 || req.Tags[2] != "extra" {
		t.Errorf("additional tags not appended: %v", req.Tags)
	}
}

func TestCreateBareMetalServer(t *testing.T) {
	bm := &fakeBareMetalService{}
	svc, machineScope := newTestService(t, bm)

	server, err := svc.CreateBareMetalServer(context.Background(), machineScope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if server.ID != "new-server" || len(bm.created) != 1 {
		t.Fatalf("server not created: %+v", server)
	}
	req := bm.created[0]
	if req.Region != testRegion || req.Plan != testPlan || req.OsID != 2284 || req.MdiskMode != "raid1" {
		t.Errorf("spec not translated: %+v", req)
	}
	if len(req.SSHKeyIDs) != 1 || req.SSHKeyIDs[0] != "key-1" {
		t.Errorf("empty ssh key id must be skipped: %v", req.SSHKeyIDs)
	}
	wantTags := map[string]bool{
		testVMClusterName:            true,
		"capvultr:cluster-uid:uid-1": true,
		"capvultr:role:worker":       true,
		testNameTag:                  true,
		"team:platform":              true,
	}
	if len(req.Tags) != len(wantTags) {
		t.Errorf("unexpected tags: %v", req.Tags)
	}
	for _, tag := range req.Tags {
		if !wantTags[tag] {
			t.Errorf("unexpected tag %q", tag)
		}
	}
	if req.UserData == "" || req.UserData == testBootstrapData {
		t.Errorf("user data must be base64 encoded: %q", req.UserData)
	}
}

func TestGetBareMetalServer(t *testing.T) {
	bm := &fakeBareMetalService{servers: map[string]*govultr.BareMetalServer{"a": {ID: "a", Status: testActive}}}
	svc, _ := newTestService(t, bm)

	if s, err := svc.GetBareMetalServer(""); s != nil || err != nil {
		t.Errorf("empty id: got %v, %v", s, err)
	}
	if s, err := svc.GetBareMetalServer("a"); err != nil || s == nil || s.ID != "a" {
		t.Errorf("existing: got %v, %v", s, err)
	}
	if s, err := svc.GetBareMetalServer("missing"); s != nil || err != nil {
		t.Errorf("404 must return nil, nil: got %v, %v", s, err)
	}
	bm.getErr = errors.New(`{"error":"boom","status":500}`)
	bm.getCode = 500
	if _, err := svc.GetBareMetalServer("a"); err == nil {
		t.Error("500 must be returned as an error")
	}

	bm.getErr = nil
	bm.getEmpty = true
	if _, err := svc.GetBareMetalServer("a"); err == nil {
		t.Error("200 without a server must be returned as an error, not as missing")
	}
}

func TestCreateBareMetalServerEmptyResponse(t *testing.T) {
	bm := &fakeBareMetalService{createEmpty: true}
	svc, ms := newTestService(t, bm)
	if s, err := svc.CreateBareMetalServer(context.Background(), ms); err == nil || s != nil {
		t.Fatalf("got %v, %v; want nil and an error", s, err)
	}
}

func TestFindBareMetalServerByTags(t *testing.T) {
	mine := govultr.BareMetalServer{ID: "mine", Tags: []string{testNameTag, "capvultr:cluster-uid:uid-1"}}
	otherCluster := govultr.BareMetalServer{ID: "other", Tags: []string{testNameTag, "capvultr:cluster-uid:uid-2"}}
	unrelated := govultr.BareMetalServer{ID: "x", Tags: []string{"name:something-else"}}

	t.Run("match across pages", func(t *testing.T) {
		bm := &fakeBareMetalService{listPages: [][]govultr.BareMetalServer{{unrelated, otherCluster}, {mine}}}
		svc, _ := newTestService(t, bm)
		s, err := svc.FindBareMetalServerByTags(testMachineName, infrav1.WorkerRoleTagValue)
		if err != nil || s == nil || s.ID != "mine" {
			t.Fatalf("got %v, %v", s, err)
		}
	})
	t.Run("no match", func(t *testing.T) {
		bm := &fakeBareMetalService{listPages: [][]govultr.BareMetalServer{{unrelated, otherCluster}}}
		svc, _ := newTestService(t, bm)
		s, err := svc.FindBareMetalServerByTags(testMachineName, infrav1.WorkerRoleTagValue)
		if err != nil || s != nil {
			t.Fatalf("got %v, %v", s, err)
		}
	})
	t.Run("duplicates are an error", func(t *testing.T) {
		bm := &fakeBareMetalService{listPages: [][]govultr.BareMetalServer{{mine, mine}}}
		svc, _ := newTestService(t, bm)
		if _, err := svc.FindBareMetalServerByTags(testMachineName, infrav1.WorkerRoleTagValue); err == nil {
			t.Fatal("expected an error for duplicate servers")
		}
	})
}

func TestEnsureBareMetalVPC(t *testing.T) {
	t.Run("already attached", func(t *testing.T) {
		bm := &fakeBareMetalService{vpcInfo: map[string][]govultr.VPCInfo{"s": {{ID: "vpc", IPAddress: "10.0.0.5"}}}}
		svc, _ := newTestService(t, bm)
		attached, info, err := svc.EnsureBareMetalVPC("s", "vpc")
		if err != nil || !attached || info == nil || info.IPAddress != "10.0.0.5" || len(bm.attached) != 0 {
			t.Fatalf("got attached=%v info=%v err=%v attachCalls=%v", attached, info, err, bm.attached)
		}
	})
	t.Run("requests attachment once", func(t *testing.T) {
		bm := &fakeBareMetalService{}
		svc, _ := newTestService(t, bm)
		attached, info, err := svc.EnsureBareMetalVPC("s", "vpc")
		if err != nil || attached || info != nil || len(bm.attached) != 1 || bm.attached[0] != "s/vpc" {
			t.Fatalf("got attached=%v info=%v err=%v attachCalls=%v", attached, info, err, bm.attached)
		}
	})
	t.Run("plan does not support vpc is terminal", func(t *testing.T) {
		bm := &fakeBareMetalService{attachErr: errors.New(`{"error":"Unable to attach VPC: Failed to attach VPC network: Plan does not support VPC networking","status":500}`)}
		svc, _ := newTestService(t, bm)
		_, _, err := svc.EnsureBareMetalVPC("s", "vpc")
		if err == nil || !IsTerminalVPCAttachError(err) {
			t.Fatalf("expected terminal error, got %v", err)
		}
		if IsTerminalVPCAttachError(errors.New("connection reset")) {
			t.Error("transport error must not be terminal")
		}
	})
}

func TestGetBareMetalServerAddresses(t *testing.T) {
	server := &govultr.BareMetalServer{ID: "s", MainIP: "45.63.50.145"}
	addrs := GetBareMetalServerAddresses(server, nil)
	if len(addrs) != 1 || addrs[0].Type != corev1.NodeExternalIP || addrs[0].Address != "45.63.50.145" {
		t.Errorf("without vpc: %v", addrs)
	}
	addrs = GetBareMetalServerAddresses(server, &govultr.VPCInfo{IPAddress: "10.1.96.10"})
	if len(addrs) != 2 || addrs[0].Type != corev1.NodeInternalIP || addrs[0].Address != "10.1.96.10" || addrs[1].Type != corev1.NodeExternalIP {
		t.Errorf("with vpc: %v", addrs)
	}
}
