// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/state"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

func functionsForPolicyTest(policytest *state.PolicyTestRecord, stateReader CombinedReader) (map[string]schema.FunctionSignature, error) {
	version := stateReader.TerraformVersion(policytest.Path())
	if version == nil {
		version = tfschema.LatestAvailableVersion
	}
	return mustFunctionsForVersion(version), nil

}

func mustFunctionsForVersion(v *version.Version) map[string]schema.FunctionSignature {
	fs, err := tfschema.FunctionsForVersion(v)
	if err != nil {
		// this should never happen
		panic(err)
	}
	coreFunctions := make(map[string]schema.FunctionSignature, len(fs))
	for name, signature := range fs {
		coreFunctions["core::"+name] = signature
	}
	return coreFunctions
}
