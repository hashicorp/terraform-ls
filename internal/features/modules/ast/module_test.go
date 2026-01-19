// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package ast

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty-debug/ctydebug"
)

func TestModuleDiags_autoloadedOnly(t *testing.T) {
	md := ModDiagsFromMap(map[string]hcl.Diagnostics{
		"alpha.tf": {},
		"beta.tf": {
			{
				Severity: hcl.DiagError,
				Summary:  "Test error",
				Detail:   "Test description",
			},
		},
		".hidden.tf": {},
	})
	diags := md.AutoloadedOnly().AsMap()
	expectedDiags := map[string]hcl.Diagnostics{
		"alpha.tf": {},
		"beta.tf": {
			{
				Severity: hcl.DiagError,
				Summary:  "Test error",
				Detail:   "Test description",
			},
		},
	}

	if diff := cmp.Diff(expectedDiags, diags, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("unexpected diagnostics: %s", diff)
	}
}
