// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	idecoder "github.com/hashicorp/terraform-ls/internal/decoder"
	"github.com/hashicorp/terraform-ls/internal/document"
	fdecoder "github.com/hashicorp/terraform-ls/internal/features/modules/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/modules/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

// DecodeWriteOnlyAttributes collects usages of write only attributes,
// using previously parsed AST (via [ParseModuleConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
func DecodeWriteOnlyAttributes(ctx context.Context, modStore *state.ModuleStore, rootFeature fdecoder.RootReader, modPath string) error {
	mod, err := modStore.ModuleRecordByPath(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if mod.WriteOnlyAttributesState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetWriteOnlyAttributesState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&fdecoder.PathReader{
		StateReader: modStore,
		RootReader:  rootFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	pd, err := d.Path(lang.Path{
		Path:       modPath,
		LanguageID: ilsp.Terraform.String(),
	})
	if err != nil {
		return err
	}

	// input list of write only attributes
	woAttrs, rErr := pd.CollectWriteOnlyAttributes()

	if rErr != nil {
		return rErr
	}

	findProviderAddr := func(resourceName string) *tfaddr.Provider {
		for localRef, addr := range mod.Meta.ProviderReferences {
			if tfschema.TypeBelongsToProvider(resourceName, localRef) {
				return &addr
			}
		}
		return nil
	}

	// output counts of write only attributes aggregated by provider, resource and attribute
	woAttrsMap := make(state.WriteOnlyAttributes)

	// count usages and resolve provider
	for _, attr := range woAttrs {
		providerAddr := findProviderAddr(attr.Resource)
		if providerAddr == nil {
			continue
		}

		if _, ok := woAttrsMap[*providerAddr]; !ok {
			woAttrsMap[*providerAddr] = make(map[state.ResourceName]map[state.AttributeName]int)
		}

		if _, ok := woAttrsMap[*providerAddr][attr.Resource]; !ok {
			woAttrsMap[*providerAddr][attr.Resource] = make(map[state.AttributeName]int)
		}

		woAttrsMap[*providerAddr][attr.Resource][attr.Name]++
	}

	sErr := modStore.UpdateWriteOnlyAttributes(modPath, woAttrsMap, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}
