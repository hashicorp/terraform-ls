// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	idecoder "github.com/hashicorp/terraform-ls/internal/decoder"
	"github.com/hashicorp/terraform-ls/internal/document"
	sdecoder "github.com/hashicorp/terraform-ls/internal/features/stacks/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// DecodeReferenceTargets collects reference targets,
// using previously parsed AST (via [ParseStackConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
//
// For example it tells us that variable block between certain LOC
// can be referred to as var.foobar. This is useful e.g. during completion,
// go-to-definition or go-to-references.
func DecodeReferenceTargets(ctx context.Context, stackStore *state.StackStore, moduleReader sdecoder.ModuleReader, rootReader sdecoder.RootReader, stackPath string) error {
	mod, err := stackStore.StackRecordByPath(stackPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if mod.RefTargetsState != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(stackPath)}
	}

	err = stackStore.SetReferenceTargetsState(stackPath, operation.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&sdecoder.PathReader{
		StateReader:  stackStore,
		ModuleReader: moduleReader,
		RootReader:   rootReader,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	stackDecoder, err := d.Path(lang.Path{
		Path:       stackPath,
		LanguageID: ilsp.Stacks.String(),
	})
	if err != nil {
		return err
	}
	stackTargets, rErr := stackDecoder.CollectReferenceTargets()

	deployDecoder, err := d.Path(lang.Path{
		Path:       stackPath,
		LanguageID: ilsp.Deploy.String(),
	})
	if err != nil {
		return err
	}
	deployTargets, rErr := deployDecoder.CollectReferenceTargets()

	record, err := stackStore.StackRecordByPath(stackPath)
	if err != nil {
		return err
	}
	builtinTargets := builtinReferences(record)

	targets := make(reference.Targets, 0)
	targets = append(targets, stackTargets...)
	targets = append(targets, deployTargets...)
	targets = append(targets, builtinTargets...)

	sErr := stackStore.UpdateReferenceTargets(stackPath, targets, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}

// DecodeReferenceOrigins collects reference origins,
// using previously parsed AST (via [ParseStackConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
//
// For example it tells us that there is a reference address var.foobar
// at a particular LOC. This can be later matched with targets
// (as obtained via [DecodeReferenceTargets]) during hover or go-to-definition.
func DecodeReferenceOrigins(ctx context.Context, stackStore *state.StackStore, moduleReader sdecoder.ModuleReader, rootReader sdecoder.RootReader, stackPath string) error {
	mod, err := stackStore.StackRecordByPath(stackPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if mod.RefOriginsState != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(stackPath)}
	}

	err = stackStore.SetReferenceOriginsState(stackPath, operation.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&sdecoder.PathReader{
		StateReader:  stackStore,
		ModuleReader: moduleReader,
		RootReader:   rootReader,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	stackDecoder, err := d.Path(lang.Path{
		Path:       stackPath,
		LanguageID: ilsp.Stacks.String(),
	})
	if err != nil {
		return err
	}
	stackOrigins, rErr := stackDecoder.CollectReferenceOrigins()

	deployDecoder, err := d.Path(lang.Path{
		Path:       stackPath,
		LanguageID: ilsp.Deploy.String(),
	})
	if err != nil {
		return err
	}
	deployOrigins, rErr := deployDecoder.CollectReferenceOrigins()

	origins := make(reference.Origins, 0)
	origins = append(origins, stackOrigins...)
	origins = append(origins, deployOrigins...)

	sErr := stackStore.UpdateReferenceOrigins(stackPath, origins, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}
