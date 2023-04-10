// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/terraform-ls/internal/state"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

func coreFunctions(mod *state.Module) map[string]schema.FunctionSignature {
	if mod.TerraformVersion != nil {
		s, err := tfschema.FunctionsForVersion(mod.TerraformVersion)
		if err == nil {
			return s
		}
		if mod.TerraformVersion.LessThan(tfschema.OldestAvailableVersion) {
			return mustFunctionsForVersion(tfschema.OldestAvailableVersion)
		}

		return mustFunctionsForVersion(tfschema.LatestAvailableVersion)
	}

	s, err := tfschema.FunctionsForConstraint(mod.Meta.CoreRequirements)
	if err == nil {
		return s
	}
	if mod.Meta.CoreRequirements.Check(tfschema.OldestAvailableVersion) {
		return mustFunctionsForVersion(tfschema.OldestAvailableVersion)
	}

	return mustFunctionsForVersion(tfschema.LatestAvailableVersion)
}

func mustFunctionsForVersion(v *version.Version) map[string]schema.FunctionSignature {
	s, err := tfschema.FunctionsForVersion(v)
	if err != nil {
		// this should never happen
		panic(err)
	}
	return s
}
