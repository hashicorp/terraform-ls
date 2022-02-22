package handlers

import (
	"context"
	"fmt"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (svc *service) TextDocumentDidChange(ctx context.Context, params lsp.DidChangeTextDocumentParams) error {
	p := lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: lsp.TextDocumentIdentifier{
				URI: params.TextDocument.URI,
			},
			Version: params.TextDocument.Version,
		},
		ContentChanges: params.ContentChanges,
	}

	dh := ilsp.HandleFromDocumentURI(p.TextDocument.URI)
	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return err
	}

	newVersion := int(p.TextDocument.Version)

	// Versions don't have to be consecutive, but they must be increasing
	if newVersion <= doc.Version {
		svc.stateStore.DocumentStore.CloseDocument(dh)
		return fmt.Errorf("Old version (%d) received, current version is %d. "+
			"Unable to update %s. This is likely a bug, please report it.",
			newVersion, doc.Version, p.TextDocument.URI)
	}

	changes := ilsp.DocumentChanges(params.ContentChanges)
	newText, err := document.ApplyChanges(doc.Text, changes)
	if err != nil {
		return err
	}
	err = svc.stateStore.DocumentStore.UpdateDocument(dh, newText, newVersion)
	if err != nil {
		return err
	}

	modMgr, err := lsctx.ModuleManager(ctx)
	if err != nil {
		return err
	}

	mod, err := modMgr.ModuleByPath(dh.Dir.Path())
	if err != nil {
		return err
	}

	err = modMgr.EnqueueModuleOp(mod.Path, op.OpTypeParseModuleConfiguration, nil)
	if err != nil {
		return err
	}
	err = modMgr.EnqueueModuleOp(mod.Path, op.OpTypeParseVariables, nil)
	if err != nil {
		return err
	}
	err = modMgr.EnqueueModuleOp(mod.Path, op.OpTypeLoadModuleMetadata, nil)
	if err != nil {
		return err
	}
	err = modMgr.EnqueueModuleOp(mod.Path, op.OpTypeDecodeReferenceTargets, nil)
	if err != nil {
		return err
	}
	err = modMgr.EnqueueModuleOp(mod.Path, op.OpTypeDecodeReferenceOrigins, nil)
	if err != nil {
		return err
	}
	err = modMgr.EnqueueModuleOp(mod.Path, op.OpTypeDecodeVarsReferences, nil)
	if err != nil {
		return err
	}

	return nil
}
