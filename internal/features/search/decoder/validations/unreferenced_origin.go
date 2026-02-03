// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package validations

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
)

func UnreferencedOrigins(ctx context.Context, pathCtx *decoder.PathContext) lang.DiagnosticsMap {
	diagsMap := make(lang.DiagnosticsMap)

	for _, origin := range pathCtx.ReferenceOrigins {
		localOrigin, ok := origin.(reference.LocalOrigin)
		if !ok {
			// We avoid reporting on other origin types.
			//
			// DirectOrigin is represented as module's source
			// and we already validate existence of the local module
			// and avoiding linking to a non-existent module in terraform-schema
			// https://github.com/hashicorp/terraform-schema/blob/b39f3de0/schema/module_schema.go#L212-L232
			//
			// PathOrigin is represented as module inputs
			// and we can validate module inputs more meaningfully
			// as attributes in body (module block), e.g. raise that
			// an input is required or unknown, rather than "reference"
			// lacking a corresponding target.
			continue

		}

		address := localOrigin.Address()

		if len(address) > 2 {
			// We temporarily ignore references with more than 2 segments
			// as these indicate references to complex types
			// which we do not fully support yet.
			// TODO: revisit as part of https://github.com/hashicorp/terraform-ls/issues/653
			continue
		}

		// we only initially validate variables & local values
		// list sources can have unknown schema
		// and will be researched at a later point
		// TODO: revisit as part of https://github.com/hashicorp/terraform-ls/issues/1364
		supported := []string{"var", "local"}
		firstStep := address[0].String()
		if !slices.Contains(supported, firstStep) {
			continue
		}

		_, ok = pathCtx.ReferenceTargets.Match(localOrigin)
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
