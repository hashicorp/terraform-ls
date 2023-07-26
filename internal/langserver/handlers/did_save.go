// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/state"
)

func (svc *service) TextDocumentDidSave(ctx context.Context, params lsp.DidSaveTextDocumentParams) error {
	expFeatures, err := lsctx.ExperimentalFeatures(ctx)
	if err != nil {
		return err
	}
	if !expFeatures.ValidateOnSave {
		return nil
	}

	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)

	// cmdHandler := &command.CmdHandler{
	// 	StateStore: svc.stateStore,
	// }
	// _, err = cmdHandler.EarlyValidationHandler(ctx, cmd.CommandArgs{
	// 	"uri": dh.Dir.URI,
	// })

	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return err
	}

	d, err := svc.decoderForDocument(ctx, doc)
	if err != nil {
		return err
	}

	notifier, err := lsctx.DiagnosticsNotifier(ctx)
	if err != nil {
		return err
	}

	mod, err := svc.modStore.ModuleByPath(dh.Dir.Path())
	if err != nil {
		if state.IsModuleNotFound(err) {
			err = svc.modStore.Add(dh.Dir.Path())
			if err != nil {
				return err
			}
			mod, err = svc.modStore.ModuleByPath(dh.Dir.Path())
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	validateDiags := make(map[string]hcl.Diagnostics, 0)
	foo, err := d.ValidateFilePerSchema(ctx, doc.Filename)
	svc.logger.Printf("DIAGS %#v", foo)
	if err != nil {
		return err
	}
	validateDiags[doc.Filename] = foo

	diags := diagnostics.NewDiagnostics()
	diags.EmptyRootDiagnostic()
	diags.Append("early validation", validateDiags)
	diags.Append("HCL", mod.ModuleDiagnostics.AutoloadedOnly().AsMap())
	diags.Append("HCL", mod.VarsDiagnostics.AutoloadedOnly().AsMap())

	notifier.PublishHCLDiags(ctx, mod.Path, diags)

	return err
}
