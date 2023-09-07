// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validations

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
)

func TestUnreferencedOrigins(t *testing.T) {
	tests := []struct {
		name    string
		origins reference.Origins
		want    lang.DiagnosticsMap
	}{
		{
			name: "undeclared variable",
			origins: reference.Origins{
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
			want: lang.DiagnosticsMap{
				"test.tf": hcl.Diagnostics{
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "No declaration found for \"var.foo\"",
						Subject: &hcl.Range{
							Filename: "test.tf",
							Start:    hcl.Pos{},
							End:      hcl.Pos{},
						},
					},
				},
			},
		},
		{
			name: "undeclared local value",
			origins: reference.Origins{
				reference.LocalOrigin{
					Range: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{},
						End:      hcl.Pos{},
					},
					Addr: lang.Address{
						lang.RootStep{Name: "local"},
						lang.AttrStep{Name: "foo"},
					},
				},
			},
			want: lang.DiagnosticsMap{
				"test.tf": hcl.Diagnostics{
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "No declaration found for \"local.foo\"",
						Subject: &hcl.Range{
							Filename: "test.tf",
							Start:    hcl.Pos{},
							End:      hcl.Pos{},
						},
					},
				},
			},
		},
		{
			name: "unsupported variable of complex type",
			origins: reference.Origins{
				reference.LocalOrigin{
					Range: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{},
						End:      hcl.Pos{},
					},
					Addr: lang.Address{
						lang.RootStep{Name: "var"},
						lang.AttrStep{Name: "obj"},
						lang.AttrStep{Name: "field"},
					},
				},
			},
			want: lang.DiagnosticsMap{},
		},
		{
			name: "unsupported path origins (module input)",
			origins: reference.Origins{
				reference.PathOrigin{
					Range: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{},
						End:      hcl.Pos{},
					},
					TargetAddr: lang.Address{
						lang.RootStep{Name: "var"},
						lang.AttrStep{Name: "foo"},
					},
					TargetPath: lang.Path{
						Path:       "./submodule",
						LanguageID: "terraform",
					},
					Constraints: reference.OriginConstraints{},
				},
			},
			want: lang.DiagnosticsMap{},
		},
		{
			name: "many undeclared variables",
			origins: reference.Origins{
				reference.LocalOrigin{
					Range: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 1, Column: 10, Byte: 10},
					},
					Addr: lang.Address{
						lang.RootStep{Name: "var"},
						lang.AttrStep{Name: "foo"},
					},
				},
				reference.LocalOrigin{
					Range: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 2, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 2, Column: 10, Byte: 10},
					},
					Addr: lang.Address{
						lang.RootStep{Name: "var"},
						lang.AttrStep{Name: "wakka"},
					},
				},
			},
			want: lang.DiagnosticsMap{
				"test.tf": hcl.Diagnostics{
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "No declaration found for \"var.foo\"",
						Subject: &hcl.Range{
							Filename: "test.tf",
							Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
							End:      hcl.Pos{Line: 1, Column: 10, Byte: 10},
						},
					},
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "No declaration found for \"var.wakka\"",
						Subject: &hcl.Range{
							Filename: "test.tf",
							Start:    hcl.Pos{Line: 2, Column: 1, Byte: 0},
							End:      hcl.Pos{Line: 2, Column: 10, Byte: 10},
						},
					},
				},
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%2d-%s", i, tt.name), func(t *testing.T) {
			ctx := context.Background()

			pathCtx := &decoder.PathContext{
				ReferenceOrigins: tt.origins,
			}

			diags := UnreferencedOrigins(ctx, pathCtx)
			if diff := cmp.Diff(tt.want["test.tf"], diags["test.tf"]); diff != "" {
				t.Fatalf("unexpected diagnostics: %s", diff)
			}
		})
	}
}
