// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validations

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
)

func TestUnReferencedOrigin(t *testing.T) {
	ctx := context.Background()
	// build pathdecoder
	// set pathctx
	// ctx = withPathContext(ctx, d.pathCtx)

	tests := []struct {
		name string
		ctx  context.Context
		want lang.DiagnosticsMap
	}{
		{
			name: "unreferenced variable",
			ctx:  ctx,
			want: lang.DiagnosticsMap{
				"test.tf": hcl.Diagnostics{
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "No reference found",
						Subject:  &hcl.Range{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UnReferencedOrigin(tt.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnReferencedOrigin() = %v, want %v", got, tt.want)
			}
		})
	}
}
