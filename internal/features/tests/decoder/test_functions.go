// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/terraform-ls/internal/features/tests/state"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

func functionsForTest(record *state.TestRecord, version *version.Version, stateReader CombinedReader) (map[string]schema.FunctionSignature, error) {
	fm := tfschema.NewFunctionsMerger(mustFunctionsForVersion(version))
	fm.SetTerraformVersion(version)
	fm.SetStateReader(stateReader)

	// providers used := stateReader.ModuleReader.LocalModuleMeta(record.Path()) ...
	// get module meta for "root" module under test (possibly in parent dir)
	// get module meta for all modules used in run blocks in record (Or the keyed on for the current test file!)

	// maybe persist the "module under test"-path in state? makes it just a single lookup then
	// needs to happen in a job that depends on module discovery done (so we know whether there really isn't code in the same dir e.g.)

	// FIXME: PathContext assumes all hcl files in a dir are merged, but that's not the case for TF test
	// something probably needs a change in hcl lang so it cleanly supports hcl files that are NOT merged

	// TODO: re-enable once we have a way to get provider requirements
	// We have to create the provider requirements and references based on the types the functions merger expects
	providerRequirements := make(tfmod.ProviderRequirements)          //, len(record.Meta.ProviderRequirements))
	providerReferences := make(map[tfmod.ProviderRef]tfaddr.Provider) //, len(record.Meta.ProviderRequirements))
	// for localName, req := range record.Meta.ProviderRequirements {
	// 	providerRequirements[req.Source] = req.VersionConstraints
	// 	providerReferences[tfmod.ProviderRef{LocalName: localName}] = req.Source
	// }

	// TODO: steps
	// 1. find the related terraform module as that's defining which providers are available for functions
	// (needs to be in terraform sources, i think; todo: check if implicit provider refs also work)

	// given a test record (which points to a test directory),
	//

	mMeta := &tfmod.Meta{
		Path:                 record.Path(),
		ProviderRequirements: providerRequirements,
		ProviderReferences:   providerReferences,
	}

	return fm.FunctionsForModule(mMeta)
}
