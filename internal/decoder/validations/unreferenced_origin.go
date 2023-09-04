// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validations

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
)

func UnreferencedOrigins(ctx context.Context, pathCtx *decoder.PathContext) lang.DiagnosticsMap {
	diagsMap := make(lang.DiagnosticsMap)

	for _, origin := range pathCtx.ReferenceOrigins {
		matchableOrigin, ok := origin.(reference.MatchableOrigin)
		if !ok {
			// we don't report on other origins to avoid complexity for now
			// other origins would need to be matched against other
			// modules/directories and we cannot be sure the targets are
			// available within the workspace or were parsed/decoded/collected
			// at the time this event occurs
			continue
		}

		_, ok = origin.(reference.LocalOrigin)
		if !ok {
			// we avoid reporting on origins outside of the current module
			// for now, to reduce complexity and reduce performance impact
			continue
		}

		// we only initially validate variables
		// resources and data sources can have unknown schema
		// and will be researched at a later point
		firstStep := matchableOrigin.Address()[0]
		if firstStep.String() != "var" {
			continue
		}

		_, ok = pathCtx.ReferenceTargets.Match(matchableOrigin)
		if !ok {
			// target not found
			fileName := origin.OriginRange().Filename
			d := &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("No declaration found for %q", matchableOrigin.Address()),
				Subject:  origin.OriginRange().Ptr(),
			}
			diagsMap[fileName] = diagsMap[fileName].Append(d)

			continue
		}

	}

	return diagsMap
}
