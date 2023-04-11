// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package diagnostics

import (
	"github.com/hashicorp/hcl/v2"
	tfjson "github.com/hashicorp/terraform-json"
)

// tfjson.Diagnostic is a conversion of an internal diag to terraform core,
// tfdiags, which is effectively based on hcl.Diagnostic.
// This process is really just converting it back to hcl.Diagnotic
// since it is the defacto diagnostic type for our codebase currently
// https://github.com/hashicorp/terraform/blob/ae025248cc0712bf53c675dc2fe77af4276dd5cc/command/validate.go#L138
func HCLDiagsFromJSON(jsonDiags []tfjson.Diagnostic) map[string]hcl.Diagnostics {
	diagsMap := make(map[string]hcl.Diagnostics)

	for _, d := range jsonDiags {
		file := ""
		if d.Range != nil {
			file = d.Range.Filename
		}

		diags := diagsMap[file]

		var severity hcl.DiagnosticSeverity
		if d.Severity == "error" {
			severity = hcl.DiagError
		} else if d.Severity == "warning" {
			severity = hcl.DiagWarning
		}

		diag := &hcl.Diagnostic{
			Severity: severity,
			Summary:  d.Summary,
			Detail:   d.Detail,
		}

		if d.Range != nil {
			diag.Subject = &hcl.Range{
				Filename: d.Range.Filename,
				Start:    hcl.Pos(d.Range.Start),
				End:      hcl.Pos(d.Range.End),
			}
		}

		diags = append(diags, diag)

		diagsMap[file] = diags
	}

	return diagsMap
}
