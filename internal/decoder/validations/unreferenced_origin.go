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

		address := matchableOrigin.Address()

		if len(address) > 2 {
			// We temporarily ignore references with more than 2 segments
			// as these indicate references to complex types
			// which we do not fully support yet.
			// TODO: revisit as part of https://github.com/hashicorp/terraform-ls/issues/653
			continue
		}

		// we only initially validate variables & local values
		// resources and data sources can have unknown schema
		// and will be researched at a later point
		firstStep := address[0].String()
		if firstStep != "var" && firstStep != "local" {
			continue
		}

		_, ok = pathCtx.ReferenceTargets.Match(matchableOrigin)
		if !ok {
			// target not found
			fileName := origin.OriginRange().Filename
			d := &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("No declaration found for %q", address),
				Subject:  origin.OriginRange().Ptr(),
			}
			diagsMap[fileName] = diagsMap[fileName].Append(d)

			continue
		}

	}

	return diagsMap
}
