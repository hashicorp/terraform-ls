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

func UnreferencedOrigins(ctx context.Context) lang.DiagnosticsMap {
	diagsMap := make(lang.DiagnosticsMap)

	pathCtx, err := decoder.PathCtx(ctx)
	if err != nil {
		return diagsMap
	}

	for _, origin := range pathCtx.ReferenceOrigins {
		matchableOrigin, ok := origin.(reference.MatchableOrigin)
		if !ok {
			continue
		}

		foo := matchableOrigin.Address()[0]
		if foo.String() != "var" {
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
