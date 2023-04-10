// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestHCLDiagsToLSP_NeverReturnsNil(t *testing.T) {
	diags := HCLDiagsToLSP(nil, "test")
	if diags == nil {
		t.Fatal("diags should not be nil")
	}

	diags = HCLDiagsToLSP(hcl.Diagnostics{}, "test")
	if diags == nil {
		t.Fatal("diags should not be nil")
	}

	diags = HCLDiagsToLSP(hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
		},
	}, "source")
	if diags == nil {
		t.Fatal("diags should not be nil")
	}
}
