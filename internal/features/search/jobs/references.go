// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	idecoder "github.com/hashicorp/terraform-ls/internal/decoder"
	"github.com/hashicorp/terraform-ls/internal/document"
	sdecoder "github.com/hashicorp/terraform-ls/internal/features/search/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/search/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// DecodeReferenceTargets collects reference targets,
// using previously parsed AST (via [ParseSearchConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
//
// For example it tells us that variable block between certain LOC
// can be referred to as var.foobar. This is useful e.g. during completion,
// go-to-definition or go-to-references.
func DecodeReferenceTargets(ctx context.Context, searchStore *state.SearchStore, moduleReader sdecoder.ModuleReader, rootReader sdecoder.RootReader, searchPath string) error {
	mod, err := searchStore.GetSearchRecordByPath(searchPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if mod.RefTargetsState != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(searchPath)}
	}

	err = searchStore.SetReferenceTargetsState(searchPath, operation.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&sdecoder.PathReader{
		StateReader:  searchStore,
		ModuleReader: moduleReader,
		RootReader:   rootReader,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	searchDecoder, err := d.Path(lang.Path{
		Path:       searchPath,
		LanguageID: ilsp.Search.String(),
	})
	if err != nil {
		return err
	}
	searchTargets, rErr := searchDecoder.CollectReferenceTargets()

	targets := make(reference.Targets, 0)
	targets = append(targets, searchTargets...)

	sErr := searchStore.UpdateReferenceTargets(searchPath, targets, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}

// DecodeReferenceOrigins collects reference origins,
// using previously parsed AST (via [ParseSearchConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
//
// For example it tells us that there is a reference address var.foobar
// at a particular LOC. This can be later matched with targets
// (as obtained via [DecodeReferenceTargets]) during hover or go-to-definition.
func DecodeReferenceOrigins(ctx context.Context, searchStore *state.SearchStore, moduleReader sdecoder.ModuleReader, rootReader sdecoder.RootReader, searchPath string) error {
	mod, err := searchStore.GetSearchRecordByPath(searchPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if mod.RefOriginsState != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(searchPath)}
	}

	err = searchStore.SetReferenceOriginsState(searchPath, operation.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&sdecoder.PathReader{
		StateReader:  searchStore,
		ModuleReader: moduleReader,
		RootReader:   rootReader,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	searchDecoder, err := d.Path(lang.Path{
		Path:       searchPath,
		LanguageID: ilsp.Search.String(),
	})
	if err != nil {
		return err
	}
	searchOrigins, rErr := searchDecoder.CollectReferenceOrigins()

	origins := make(reference.Origins, 0)
	origins = append(origins, searchOrigins...)

	sErr := searchStore.UpdateReferenceOrigins(searchPath, origins, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}
