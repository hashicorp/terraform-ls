// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/ast"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/state"
)

func policytestPathContext(policytest *state.PolicyTestRecord, stateReader CombinedReader) (*decoder.PathContext, error) {
	schema, err := schemaForPolicyTest(policytest, stateReader)
	if err != nil {
		return nil, err
	}
	functions, err := functionsForPolicyTest(policytest, stateReader)
	if err != nil {
		return nil, err
	}

	pathCtx := &decoder.PathContext{
		Schema:           schema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File, 0),
		Functions:        functions,
		Validators:       policytestValidators,
	}

	for _, origin := range policytest.RefOrigins {
		if ast.IsPolicyTestFilename(origin.OriginRange().Filename) {
			pathCtx.ReferenceOrigins = append(pathCtx.ReferenceOrigins, origin)
		}
	}
	for _, target := range policytest.RefTargets {
		if target.RangePtr != nil && ast.IsPolicyTestFilename(target.RangePtr.Filename) {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		} else if target.RangePtr == nil {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		}
	}

	for name, f := range policytest.ParsedPolicyTestFiles {
		pathCtx.Files[name.String()] = f
	}

	return pathCtx, nil
}
