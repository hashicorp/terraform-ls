// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validations

import (
	"context"
	"fmt"
	"log"

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
			continue
		}

		foo := matchableOrigin.Address()[0]
		if foo.String() != "var" {
			continue
		}

		log.Printf("MatchableOrigin: %s", matchableOrigin.Address())

		_, ok = pathCtx.ReferenceTargets.Match(matchableOrigin)
		if !ok {
			// target not found
			diagsMap[origin.OriginRange().Filename] = hcl.Diagnostics{
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("No declaration found for %q", matchableOrigin.Address()),
					Subject:  origin.OriginRange().Ptr(),
				},
			}
			continue
		}

	}
	log.Printf("Length: %d Diags produced: %+v", len(pathCtx.ReferenceOrigins) , diagsMap)
	return diagsMap
}
