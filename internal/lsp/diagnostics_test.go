package lsp

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestHCLDiagsToLSP_NeverReturnsNil(t *testing.T) {
	diags := HCLDiagsToLSP(nil)
	if diags == nil {
		t.Fatal("diags should not be nil")
	}

	diags = HCLDiagsToLSP(hcl.Diagnostics{})
	if diags == nil {
		t.Fatal("diags should not be nil")
	}

	diags = HCLDiagsToLSP(hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
		},
	})
	if diags == nil {
		t.Fatal("diags should not be nil")
	}
}
