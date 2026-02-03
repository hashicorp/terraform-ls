// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	idecoder "github.com/hashicorp/terraform-ls/internal/decoder"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/ast"
	stackDecoder "github.com/hashicorp/terraform-ls/internal/features/stacks/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/decoder/validations"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func SchemaStackValidation(ctx context.Context, stackStore *state.StackStore, moduleFeature stackDecoder.ModuleReader, rootFeature stackDecoder.RootReader, stackPath string) error {
	rpcContext := lsctx.DocumentContext(ctx)

	record, err := stackStore.StackRecordByPath(stackPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if record.DiagnosticsState[globalAst.SchemaValidationSource] != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(stackPath)}
	}

	err = stackStore.SetDiagnosticsState(stackPath, globalAst.SchemaValidationSource, operation.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&stackDecoder.PathReader{
		StateReader:  stackStore,
		ModuleReader: moduleFeature,
		RootReader:   rootFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	var rErr error
	if rpcContext.Method == "textDocument/didChange" {
		// We validate only the file that has changed
		// This means only creating a decoder for the file type that has changed
		decoder, err := d.Path(lang.Path{
			Path:       stackPath,
			LanguageID: rpcContext.LanguageID,
		})
		if err != nil {
			return err
		}

		filename := path.Base(rpcContext.URI)

		var fileDiags hcl.Diagnostics
		fileDiags, rErr = decoder.ValidateFile(ctx, filename)

		diags, ok := record.Diagnostics[globalAst.SchemaValidationSource]
		if !ok {
			diags = make(ast.Diagnostics)
		}
		diags[ast.FilenameFromName(filename)] = fileDiags

		sErr := stackStore.UpdateDiagnostics(stackPath, globalAst.SchemaValidationSource, diags)
		if sErr != nil {
			return sErr
		}
	} else {
		// We validate the whole stack, and so need to create decoders for
		// all the file types in the stack
		stackDecoder, err := d.Path(lang.Path{
			Path:       stackPath,
			LanguageID: ilsp.Stacks.String(),
		})
		if err != nil {
			return err
		}
		deployDecoder, err := d.Path(lang.Path{
			Path:       stackPath,
			LanguageID: ilsp.Deploy.String(),
		})
		if err != nil {
			return err
		}

		diags := make(lang.DiagnosticsMap)

		stacksDiags, err := stackDecoder.Validate(ctx)
		if err != nil {
			// TODO: Should we really return here or continue with the other decoders?
			// Is this really a complete fail case? Shouldn't a failure in a stack file
			// not prevent the deploy file from being validated?
			return err
		}
		diags = diags.Extend(stacksDiags)

		deployDiags, err := deployDecoder.Validate(ctx)
		if err != nil {
			return err
		}
		diags = diags.Extend(deployDiags)

		sErr := stackStore.UpdateDiagnostics(stackPath, globalAst.SchemaValidationSource, ast.DiagnosticsFromMap(diags))
		if sErr != nil {
			return sErr
		}
	}

	return rErr
}

// ReferenceValidation does validation based on (mis)matched
// reference origins and targets, to flag up "orphaned" references.
//
// It relies on [DecodeReferenceTargets] and [DecodeReferenceOrigins]
// to supply both origins and targets to compare.
func ReferenceValidation(ctx context.Context, stackStore *state.StackStore, moduleFeature stackDecoder.ModuleReader, rootFeature stackDecoder.RootReader, stackPath string) error {
	record, err := stackStore.StackRecordByPath(stackPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if record.DiagnosticsState[globalAst.ReferenceValidationSource] != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(stackPath)}
	}

	err = stackStore.SetDiagnosticsState(stackPath, globalAst.ReferenceValidationSource, operation.OpStateLoading)
	if err != nil {
		return err
	}

	pathReader := &stackDecoder.PathReader{
		StateReader:  stackStore,
		ModuleReader: moduleFeature,
		RootReader:   rootFeature,
	}

	stackDecoder, err := pathReader.PathContext(lang.Path{
		Path:       stackPath,
		LanguageID: ilsp.Stacks.String(),
	})
	if err != nil {
		return err
	}

	deployDecoder, err := pathReader.PathContext(lang.Path{
		Path:       stackPath,
		LanguageID: ilsp.Deploy.String(),
	})
	if err != nil {
		return err
	}

	diags := validations.UnreferencedOrigins(ctx, stackDecoder)

	deployDiags := validations.UnreferencedOrigins(ctx, deployDecoder)
	diags = diags.Extend(deployDiags)

	return stackStore.UpdateDiagnostics(stackPath, globalAst.ReferenceValidationSource, ast.DiagnosticsFromMap(diags))
}
