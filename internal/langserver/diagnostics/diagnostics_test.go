// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package diagnostics

import (
	"context"
	"io/ioutil"
	"log"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
)

var discardLogger = log.New(ioutil.Discard, "", 0)

func TestDiags_Closes(t *testing.T) {
	n := NewNotifier(noopNotifier{}, discardLogger)

	diags := NewDiagnostics()
	diags.Append(ast.HCLParsingSource, map[string]hcl.Diagnostics{
		ast.HCLParsingSource.String(): {
			{
				Severity: hcl.DiagError,
			},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n.PublishHCLDiags(ctx, t.TempDir(), diags)

	if _, open := <-n.diags; open {
		t.Fatal("channel should be closed")
	}
}

func TestPublish_DoesNotSendAfterClose(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Fatal(err)
		}
	}()

	n := NewNotifier(noopNotifier{}, discardLogger)

	diags := NewDiagnostics()
	diags.Append(ast.TerraformValidateSource, map[string]hcl.Diagnostics{
		ast.TerraformValidateSource.String(): {
			{
				Severity: hcl.DiagError,
			},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n.PublishHCLDiags(ctx, t.TempDir(), diags)
}

func TestDiagnostics_Append(t *testing.T) {
	diags := NewDiagnostics()
	diags.Append(ast.HCLParsingSource, map[string]hcl.Diagnostics{
		"first.tf": {
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Something went wrong",
				Detail:   "Testing detail",
				Subject: &hcl.Range{
					Filename: "first.tf",
					Start:    hcl.InitialPos,
					End: hcl.Pos{
						Line:   3,
						Column: 2,
						Byte:   10,
					},
				},
			},
		},
	})
	diags.Append(ast.SchemaValidationSource, map[string]hcl.Diagnostics{
		"first.tf": {
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Something else went wrong",
				Detail:   "Testing detail",
			},
		},
		"second.tf": {
			&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Beware",
			},
		},
	})

	expectedDiags := Diagnostics{
		"first.tf": map[ast.DiagnosticSource]hcl.Diagnostics{
			ast.HCLParsingSource: {
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Something went wrong",
					Detail:   "Testing detail",
					Subject: &hcl.Range{
						Filename: "first.tf",
						Start:    hcl.InitialPos,
						End: hcl.Pos{
							Line:   3,
							Column: 2,
							Byte:   10,
						},
					},
				},
			},
			ast.SchemaValidationSource: {
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Something else went wrong",
					Detail:   "Testing detail",
				},
			},
		},
		"second.tf": map[ast.DiagnosticSource]hcl.Diagnostics{
			ast.SchemaValidationSource: {
				&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Beware",
				},
			},
		},
	}
	if diff := cmp.Diff(expectedDiags, diags); diff != "" {
		t.Fatalf("diagnostics mismatch: %s", diff)
	}
}

type noopNotifier struct{}

func (noopNotifier) Notify(ctx context.Context, method string, params interface{}) error {
	return nil
}
