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

package vultrerrors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vultr/govultr/v3"
)

func TestCapture(t *testing.T) {
	ctx, status := Capture(context.Background())
	if status() != 0 {
		t.Fatal("no response yet means 0")
	}
	Record(ctx, http.StatusNotFound)
	if status() != http.StatusNotFound {
		t.Fatalf("got %d", status())
	}
	other, otherStatus := Capture(context.Background())
	Record(other, http.StatusOK)
	if status() != http.StatusNotFound || otherStatus() != http.StatusOK {
		t.Fatal("captures must not share state")
	}
}

// TestThroughGovultr proves the transport works under the real client on the
// pinned govultr version, including for Delete, which returns no response at
// all, and for a 5xx after retries are exhausted.
func TestThroughGovultr(t *testing.T) {
	cases := []struct {
		name    string
		code    int
		body    string
		retries int
	}{
		{"404", http.StatusNotFound, `{"error":"instance not found","status":404}`, 0},
		{"422", http.StatusUnprocessableEntity, `{"error":"Load Balancer Nodes Still Activating...","status":422}`, 0},
		{"200 with error body", http.StatusOK, `{"error":"(REMOTE) Server failed to include a response","status":400}`, 0},
		{"500 after retries", http.StatusInternalServerError, `{"error":"Internal server error.","status":500}`, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.code)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			c := govultr.NewClient(&http.Client{Transport: Transport{Next: http.DefaultTransport}})
			_ = c.SetBaseURL(srv.URL)
			c.SetRetryLimit(tc.retries)
			c.SetRateLimit(5 * time.Millisecond)

			ctx, status := Capture(context.Background())
			if _, _, err := c.Instance.Get(ctx, "x"); err == nil || status() != tc.code {
				t.Fatalf("get: want error and status %d, got %v, %d", tc.code, err, status())
			}
			ctx, status = Capture(context.Background())
			if err := c.Instance.Delete(ctx, "x"); err == nil || status() != tc.code {
				t.Fatalf("delete: want error and status %d, got %v, %d", tc.code, err, status())
			}
		})
	}
}
