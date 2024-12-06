// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	idecoder "github.com/hashicorp/terraform-ls/internal/decoder"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/tests/ast"
	fdecoder "github.com/hashicorp/terraform-ls/internal/features/tests/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/tests/state"
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
func DecodeReferenceTargets(ctx context.Context, testStore *state.TestStore, testPath string, moduleFeature fdecoder.ModuleReader, rootFeature fdecoder.RootReader) error {
	record, err := testStore.TestRecordByPath(testPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if record.RefTargetsState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(testPath)}
	}

	err = testStore.SetReferenceTargetsState(testPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&fdecoder.PathReader{
		StateReader:  testStore,
		ModuleReader: moduleFeature,
		RootReader:   rootFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	// loop through all parsed files and collect targets separately for all test files
	testTargets := make(map[string]reference.Targets, 0)
	var rErr error
	for name := range record.ParsedFiles {
		if _, ok := name.(ast.TestFilename); ok {
			testDecoder, err := d.Path(lang.Path{
				Path:       testPath,
				LanguageID: ilsp.Test.String(),
				File:       name.String(),
			})
			if err != nil {
				return err
			}
			targets, err := testDecoder.CollectReferenceTargets()
			testTargets[name.String()] = targets
			rErr = err
		}
	}

	mockDecoder, err := d.Path(lang.Path{
		Path:       testPath,
		LanguageID: ilsp.Mock.String(),
	})
	if err != nil {
		return err
	}
	mockTargets, rErr := mockDecoder.CollectReferenceTargets()

	sErr := testStore.UpdateReferenceTargets(testPath, testTargets, mockTargets, rErr)
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
func DecodeReferenceOrigins(ctx context.Context, testStore *state.TestStore, testPath string, moduleFeature fdecoder.ModuleReader, rootFeature fdecoder.RootReader) error {
	record, err := testStore.TestRecordByPath(testPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if record.RefOriginsState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(testPath)}
	}

	err = testStore.SetReferenceOriginsState(testPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&fdecoder.PathReader{
		StateReader:  testStore,
		ModuleReader: moduleFeature,
		RootReader:   rootFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	// loop through all parsed files and collect targets separately for all test files
	testOrigins := make(map[string]reference.Origins, 0)
	var rErr error
	for name := range record.ParsedFiles {
		if _, ok := name.(ast.TestFilename); ok {
			testDecoder, err := d.Path(lang.Path{
				Path:       testPath,
				LanguageID: ilsp.Test.String(),
				File:       name.String(),
			})
			if err != nil {
				return err
			}
			origins, err := testDecoder.CollectReferenceOrigins()
			testOrigins[name.String()] = origins
			rErr = err
		}
	}

	mockDecoder, err := d.Path(lang.Path{
		Path:       testPath,
		LanguageID: ilsp.Mock.String(),
	})
	if err != nil {
		return err
	}
	mockOrigins, rErr := mockDecoder.CollectReferenceOrigins()

	sErr := testStore.UpdateReferenceOrigins(testPath, testOrigins, mockOrigins, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}
