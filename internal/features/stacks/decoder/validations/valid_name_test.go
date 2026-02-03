// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package validations

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestHasValidNameLabel_no_label(t *testing.T) {
	var diags hcl.Diagnostics
	block := hclsyntax.Block{
		Labels: []string{}, // should not panic
	}

	hasValidNameLabel(&block, diags)
}
