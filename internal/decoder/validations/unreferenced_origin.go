// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validations

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
)

func UnReferencedOrigin(ctx context.Context) lang.DiagnosticsMap {
	pathCtx, err := decoder.PathCtx(ctx)
	if err != nil {
		// TODO
	}

	diagsMap := make(lang.DiagnosticsMap)

	for _, origin := range pathCtx.ReferenceOrigins {
		matchableOrigin, ok := origin.(reference.MatchableOrigin)
		if !ok {
			// TODO: add a diag here
			continue
		}

		_, ok = pathCtx.ReferenceTargets.Match(matchableOrigin)
		if !ok {
			// target not found
			diagsMap[origin.OriginRange().Filename] = hcl.Diagnostics{
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "No reference found", // TODO: Is there more we can state here?
					Subject:  origin.OriginRange().Ptr(),
				},
			}
			continue
		}

	}

	return diagsMap
}
