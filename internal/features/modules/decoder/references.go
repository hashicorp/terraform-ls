// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/terraform-ls/internal/features/modules/state"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

func referencesForModule(mod *state.ModuleRecord, stateReader CombinedReader) reference.Targets {
	modPath := mod.Path()
	resolvedVersion := tfschema.ResolveVersion(stateReader.TerraformVersion(modPath), mod.Meta.CoreRequirements)

	return tfschema.BuiltinReferencesForVersion(resolvedVersion, modPath)
}
