// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package diagnostics

import (
	"context"
	"io/ioutil"
	"log"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
)

var discardLogger = log.New(ioutil.Discard, "", 0)

func TestDiags_Closes(t *testing.T) {
	n := NewNotifier(noopNotifier{}, discardLogger)

	diags := NewDiagnostics()
	diags.Append("test", map[string]hcl.Diagnostics{
		"test": {
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
	diags.Append("test", map[string]hcl.Diagnostics{
		"test": {
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
	diags.Append("foo", map[string]hcl.Diagnostics{
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
	diags.Append("bar", map[string]hcl.Diagnostics{
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
		"first.tf": map[DiagnosticSource]hcl.Diagnostics{
			DiagnosticSource("foo"): {
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
			DiagnosticSource("bar"): {
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Something else went wrong",
					Detail:   "Testing detail",
				},
			},
		},
		"second.tf": map[DiagnosticSource]hcl.Diagnostics{
			DiagnosticSource("bar"): {
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
