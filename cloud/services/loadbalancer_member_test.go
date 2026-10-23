package services

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/vultr/govultr/v3"

	"github.com/vultr/cluster-api-provider-vultr/cloud/vultrerrors"
)

type fakeLBService struct {
	govultr.LoadBalancerService

	lb         *govultr.LoadBalancer
	getCode    int
	updates    []*govultr.LoadBalancerReq
	updateErr  error
	updateCode int
}

func (f *fakeLBService) Get(ctx context.Context, _ string) (*govultr.LoadBalancer, *http.Response, error) {
	if f.lb == nil {
		vultrerrors.Record(ctx, f.getCode)
		return nil, nil, errors.New("error returned from api : Load Balancer Subscription ID Not Found")
	}
	vultrerrors.Record(ctx, http.StatusOK)
	return f.lb, &http.Response{StatusCode: http.StatusOK}, nil
}

func (f *fakeLBService) Update(ctx context.Context, _ string, req *govultr.LoadBalancerReq) error {
	f.updates = append(f.updates, req)
	if f.updateErr != nil {
		vultrerrors.Record(ctx, f.updateCode)
		return f.updateErr
	}
	vultrerrors.Record(ctx, http.StatusNoContent)
	f.lb.Instances = req.Instances
	return nil
}

func newLBTestService(t *testing.T, lb *fakeLBService) *Service {
	t.Helper()
	svc, _ := newTestService(t, &fakeBareMetalService{})
	svc.scope.LoadBalancers = lb
	return svc
}

func TestEnsureLoadBalancerMember(t *testing.T) {
	t.Run("not active yet", func(t *testing.T) {
		lb := &fakeLBService{lb: &govultr.LoadBalancer{Status: "pending"}}
		svc := newLBTestService(t, lb)
		member, err := svc.EnsureLoadBalancerMember("lb", "srv")
		if err != nil || member || len(lb.updates) != 0 {
			t.Fatalf("member=%v err=%v updates=%d", member, err, len(lb.updates))
		}
	})
	t.Run("appends once and keeps existing members", func(t *testing.T) {
		lb := &fakeLBService{lb: &govultr.LoadBalancer{Status: testActive, Instances: []string{"other"}}}
		svc := newLBTestService(t, lb)
		for i := 0; i < 2; i++ {
			member, err := svc.EnsureLoadBalancerMember("lb", "srv")
			if err != nil || !member {
				t.Fatalf("iteration %d: member=%v err=%v", i, member, err)
			}
		}
		if len(lb.updates) != 1 || len(lb.updates[0].Instances) != 2 || lb.updates[0].Instances[1] != "srv" {
			t.Fatalf("unexpected updates: %+v", lb.updates)
		}
	})
	t.Run("empty lb id", func(t *testing.T) {
		svc := newLBTestService(t, &fakeLBService{})
		if _, err := svc.EnsureLoadBalancerMember("", "srv"); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestEnsureLoadBalancerMemberStillActivating(t *testing.T) {
	lb := &fakeLBService{lb: &govultr.LoadBalancer{Status: "active"}, updateErr: errors.New("error returned from api : Load Balancer Nodes Still Activating..."), updateCode: 422}
	svc := newLBTestService(t, lb)
	member, err := svc.EnsureLoadBalancerMember("lb", "i")
	if err != nil || member {
		t.Fatalf("got member=%v err=%v; want false, nil so the caller requeues", member, err)
	}
	lb.updateErr, lb.updateCode = errors.New("error returned from api : Internal server error."), 500
	if _, err := svc.EnsureLoadBalancerMember("lb", "i"); err == nil {
		t.Fatal("500 must be returned as an error")
	}
}

func TestRemoveLoadBalancerMember(t *testing.T) {
	t.Run("removes only the member", func(t *testing.T) {
		lb := &fakeLBService{lb: &govultr.LoadBalancer{Status: testActive, Instances: []string{"a", "srv", "b"}}}
		svc := newLBTestService(t, lb)
		if err := svc.RemoveLoadBalancerMember("lb", "srv"); err != nil {
			t.Fatal(err)
		}
		if len(lb.updates) != 1 || len(lb.updates[0].Instances) != 2 {
			t.Fatalf("unexpected updates: %+v", lb.updates)
		}
	})
	t.Run("not a member is a no-op", func(t *testing.T) {
		lb := &fakeLBService{lb: &govultr.LoadBalancer{Status: testActive, Instances: []string{"a"}}}
		svc := newLBTestService(t, lb)
		if err := svc.RemoveLoadBalancerMember("lb", "srv"); err != nil || len(lb.updates) != 0 {
			t.Fatalf("err=%v updates=%d", err, len(lb.updates))
		}
	})
	t.Run("missing load balancer is not an error", func(t *testing.T) {
		svc := newLBTestService(t, &fakeLBService{getCode: http.StatusNotFound})
		if err := svc.RemoveLoadBalancerMember("lb", "srv"); err != nil {
			t.Fatal(err)
		}
	})
}
