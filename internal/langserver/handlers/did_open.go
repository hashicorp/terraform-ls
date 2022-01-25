package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (svc *service) TextDocumentDidOpen(ctx context.Context, params lsp.DidOpenTextDocumentParams) error {
	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)

	err := svc.stateStore.DocumentStore.OpenDocument(dh, params.TextDocument.LanguageID,
		int(params.TextDocument.Version), []byte(params.TextDocument.Text))
	if err != nil {
		return err
	}

	modMgr, err := lsctx.ModuleManager(ctx)
	if err != nil {
		return err
	}

	var mod module.Module

	mod, err = modMgr.ModuleByPath(dh.Dir.Path())
	if err != nil {
		if module.IsModuleNotFound(err) {
			mod, err = modMgr.AddModule(dh.Dir.Path())
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	svc.logger.Printf("opened module: %s", mod.Path)

	// We reparse because the file being opened may not match
	// (originally parsed) content on the disk
	// TODO: Do this only if we can verify the file differs?
	modMgr.EnqueueModuleOp(mod.Path, op.OpTypeParseModuleConfiguration, nil)
	modMgr.EnqueueModuleOp(mod.Path, op.OpTypeParseVariables, nil)
	modMgr.EnqueueModuleOp(mod.Path, op.OpTypeLoadModuleMetadata, nil)
	modMgr.EnqueueModuleOp(mod.Path, op.OpTypeDecodeReferenceTargets, nil)
	modMgr.EnqueueModuleOp(mod.Path, op.OpTypeDecodeReferenceOrigins, nil)
	modMgr.EnqueueModuleOp(mod.Path, op.OpTypeDecodeVarsReferences, nil)

	if mod.TerraformVersionState == op.OpStateUnknown {
		modMgr.EnqueueModuleOp(mod.Path, op.OpTypeGetTerraformVersion, nil)
	}

	watcher, err := lsctx.Watcher(ctx)
	if err != nil {
		return err
	}

	if !watcher.IsModuleWatched(mod.Path) {
		err := watcher.AddModule(mod.Path)
		if err != nil {
			return err
		}
	}

	return nil
}
