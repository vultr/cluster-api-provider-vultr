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

// Package vultrerrors recovers the HTTP status of a govultr call. govultr
// drops it from the errors it returns, so a transport under govultr records
// the status of every response for the call that owns the request's context.
package vultrerrors

import (
	"context"
	"net/http"
)

type ctxKey struct{}

type capture struct{ status int }

// Capture returns a context for one govultr call and a function that reports
// the HTTP status of that call, or 0 if no response was received.
func Capture(ctx context.Context) (context.Context, func() int) {
	c := &capture{}
	return context.WithValue(ctx, ctxKey{}, c), func() int { return c.status }
}

// Record stores the HTTP status for the call that owns ctx.
func Record(ctx context.Context, status int) {
	if c, ok := ctx.Value(ctxKey{}).(*capture); ok {
		c.status = status
	}
}

// Transport records the status of every response that passes through it.
type Transport struct {
	Next http.RoundTripper
}

// RoundTrip sends the request through Next and records the response status.
func (t Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := t.Next.RoundTrip(req)
	if res != nil {
		Record(req.Context(), res.StatusCode)
	}
	return res, err
}
