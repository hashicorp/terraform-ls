// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"context"
	"fmt"
)

type clientNameCtxKey struct{}

func ContextWithClientName(ctx context.Context, namePtr *string) context.Context {
	return context.WithValue(ctx, clientNameCtxKey{}, namePtr)
}

func ClientName(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(clientNameCtxKey{}).(*string)
	if !ok {
		return "", false
	}
	return *name, true
}

func SetClientName(ctx context.Context, name string) error {
	namePtr, ok := ctx.Value(clientNameCtxKey{}).(*string)
	if !ok {
		return fmt.Errorf("missing context: client name")
	}

	*namePtr = name
	return nil
}
