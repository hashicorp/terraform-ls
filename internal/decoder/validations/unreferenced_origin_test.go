// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validations

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
)

func TestUnReferencedOrigin(t *testing.T) {
	ctx := context.Background()

	pathCtx := &decoder.PathContext{
		ReferenceOrigins: reference.Origins{
			reference.LocalOrigin{
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{},
					End:      hcl.Pos{},
				},
				Addr: lang.Address{
					lang.RootStep{Name: "var"},
					lang.AttrStep{Name: "foo"},
				},
			},
		},
	}

	ctx = decoder.WithPathContext(ctx, pathCtx)

	tests := []struct {
		name string
		ctx  context.Context
		want lang.DiagnosticsMap
	}{
		{
			name: "undeclared variable",
			ctx:  ctx,
			want: lang.DiagnosticsMap{
				"test.tf": hcl.Diagnostics{
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "No declaration found for \"var.foo\"",
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
