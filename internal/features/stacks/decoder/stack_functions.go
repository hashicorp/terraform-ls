// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

func functionsForStack(record *state.StackRecord, version *version.Version, stateReader CombinedReader) (map[string]schema.FunctionSignature, error) {
	fm := tfschema.NewFunctionsMerger(mustFunctionsForVersion(version))
	fm.SetTerraformVersion(version)
	fm.SetStateReader(stateReader)

	// We have to create the provider requirements and references based on the types the functions merger expects
	providerRequirements := make(tfmod.ProviderRequirements, len(record.Meta.ProviderRequirements))
	providerReferences := make(map[tfmod.ProviderRef]tfaddr.Provider, len(record.Meta.ProviderRequirements))
	for localName, req := range record.Meta.ProviderRequirements {
		providerRequirements[req.Source] = req.VersionConstraints
		providerReferences[tfmod.ProviderRef{LocalName: localName}] = req.Source
	}

	mMeta := &tfmod.Meta{
		Path:                 record.Path(),
		ProviderRequirements: providerRequirements,
		ProviderReferences:   providerReferences,
	}

	return fm.FunctionsForModule(mMeta)
}
