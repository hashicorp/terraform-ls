// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package exec

import (
	"context"
	"time"
)

type ExecutorOpts struct {
	ExecPath    string
	ExecLogPath string
	Timeout     time.Duration
}

var ctxExecOpts = ctxKey("executor opts")

func ExecutorOptsFromContext(ctx context.Context) (*ExecutorOpts, bool) {
	opts, ok := ctx.Value(ctxExecOpts).(*ExecutorOpts)
	return opts, ok
}

func WithExecutorOpts(ctx context.Context, opts *ExecutorOpts) context.Context {
	return context.WithValue(ctx, ctxExecOpts, opts)
}
