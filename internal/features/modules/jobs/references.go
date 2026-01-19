// Copyright IBM Corp. 2020, 2025
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
)

// DecodeReferenceTargets collects reference targets,
// using previously parsed AST (via [ParseModuleConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
//
// For example it tells us that variable block between certain LOC
// can be referred to as var.foobar. This is useful e.g. during completion,
// go-to-definition or go-to-references.
func DecodeReferenceTargets(ctx context.Context, modStore *state.ModuleStore, rootFeature fdecoder.RootReader, modPath string) error {
	mod, err := modStore.ModuleRecordByPath(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if mod.RefTargetsState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetReferenceTargetsState(modPath, op.OpStateLoading)
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
	targets, rErr := pd.CollectReferenceTargets()

	sErr := modStore.UpdateReferenceTargets(modPath, targets, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}

// DecodeReferenceOrigins collects reference origins,
// using previously parsed AST (via [ParseModuleConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
//
// For example it tells us that there is a reference address var.foobar
// at a particular LOC. This can be later matched with targets
// (as obtained via [DecodeReferenceTargets]) during hover or go-to-definition.
func DecodeReferenceOrigins(ctx context.Context, modStore *state.ModuleStore, rootFeature fdecoder.RootReader, modPath string) error {
	mod, err := modStore.ModuleRecordByPath(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if mod.RefOriginsState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetReferenceOriginsState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&fdecoder.PathReader{
		StateReader: modStore,
		RootReader:  rootFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	moduleDecoder, err := d.Path(lang.Path{
		Path:       modPath,
		LanguageID: ilsp.Terraform.String(),
	})
	if err != nil {
		return err
	}

	origins, rErr := moduleDecoder.CollectReferenceOrigins()

	sErr := modStore.UpdateReferenceOrigins(modPath, origins, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}
