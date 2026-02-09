// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/policy/ast"
	"github.com/hashicorp/terraform-ls/internal/features/policy/state"
)

func policyPathContext(policy *state.PolicyRecord, stateReader CombinedReader) (*decoder.PathContext, error) {
	schema, err := schemaForPolicy(policy, stateReader)
	if err != nil {
		return nil, err
	}
	functions, err := functionsForPolicy(policy, stateReader)
	if err != nil {
		return nil, err
	}

	pathCtx := &decoder.PathContext{
		Schema:           schema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File, 0),
		Functions:        functions,
		Validators:       policyValidators,
	}

	for _, origin := range policy.RefOrigins {
		if ast.IsPolicyFilename(origin.OriginRange().Filename) {
			pathCtx.ReferenceOrigins = append(pathCtx.ReferenceOrigins, origin)
		}
	}
	for _, target := range policy.RefTargets {
		if target.RangePtr != nil && ast.IsPolicyFilename(target.RangePtr.Filename) {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		} else if target.RangePtr == nil {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		}
	}

	for name, f := range policy.ParsedPolicyFiles {
		pathCtx.Files[name.String()] = f
	}

	return pathCtx, nil
}
