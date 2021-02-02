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
		// the diagnostic must be tied to a file to exist in the map
		if d.Range == nil || d.Range.Filename == "" {
			continue
		}

		diags := diagsMap[d.Range.Filename]

		var severity hcl.DiagnosticSeverity
		if d.Severity == "error" {
			severity = hcl.DiagError
		} else if d.Severity == "warning" {
			severity = hcl.DiagWarning
		}

		diags = append(diags, &hcl.Diagnostic{
			Severity: severity,
			Summary:  d.Summary,
			Detail:   d.Detail,
			Subject: &hcl.Range{
				Filename: d.Range.Filename,
				Start:    hcl.Pos(d.Range.Start),
				End:      hcl.Pos(d.Range.End),
			},
		})
		diagsMap[d.Range.Filename] = diags
	}

	return diagsMap
}
