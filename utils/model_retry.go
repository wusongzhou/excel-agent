/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package utils

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// retryDelays is the backoff schedule between attempts. A single agent task
// issues a long series of model requests (plan, execute, replan, report), so
// rate limits (HTTP 429) are hit on low-tier plans; waiting them out is the
// only remedy short of upgrading the plan.
var retryDelays = []time.Duration{5 * time.Second, 10 * time.Second, 20 * time.Second}

// newRetryingChatModel wraps a chat model with bounded retries on transient
// failures (rate limits and gateway errors).
func newRetryingChatModel(inner model.ToolCallingChatModel) model.ToolCallingChatModel {
	return &retryingChatModel{inner: inner, delays: retryDelays}
}

type retryingChatModel struct {
	inner  model.ToolCallingChatModel
	delays []time.Duration
}

func (r *retryingChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	inner, err := r.inner.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &retryingChatModel{inner: inner, delays: r.delays}, nil
}

func (r *retryingChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	var out *schema.Message
	err := r.withRetry(ctx, func() error {
		var e error
		out, e = r.inner.Generate(ctx, input, opts...)
		return e
	})
	return out, err
}

func (r *retryingChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	var out *schema.StreamReader[*schema.Message]
	err := r.withRetry(ctx, func() error {
		var e error
		out, e = r.inner.Stream(ctx, input, opts...)
		return e
	})
	return out, err
}

func (r *retryingChatModel) withRetry(ctx context.Context, call func() error) error {
	for attempt := 0; ; attempt++ {
		err := call()
		if err == nil || attempt >= len(r.delays) || !isTransientError(err) {
			return err
		}

		delay := r.delays[attempt]
		log.Printf("[model] transient error (%v), retrying in %s (attempt %d/%d)", err, delay, attempt+1, len(r.delays))
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// isTransientError reports whether the error looks like a rate limit or a
// gateway hiccup worth retrying. Detection is text-based because providers
// surface these as plain errors in varying shapes.
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	if strings.Contains(s, "status code: 429") ||
		strings.Contains(s, "status code: 500") ||
		strings.Contains(s, "status code: 502") ||
		strings.Contains(s, "status code: 503") ||
		strings.Contains(s, "status code: 504") {
		return true
	}
	s = strings.ToLower(s)
	return strings.Contains(s, "429") ||
		strings.Contains(s, "too many requests") ||
		strings.Contains(s, "rate limit")
}
