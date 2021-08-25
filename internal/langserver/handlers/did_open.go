package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (lh *logHandler) TextDocumentDidOpen(ctx context.Context, params lsp.DidOpenTextDocumentParams) error {
	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return err
	}

	f := ilsp.FileFromDocumentItem(params.TextDocument)
	err = fs.CreateAndOpenDocument(f, f.LanguageID(), f.Text())
	if err != nil {
		return err
	}

	modMgr, err := lsctx.ModuleManager(ctx)
	if err != nil {
		return err
	}

	var mod module.Module

	mod, err = modMgr.ModuleByPath(f.Dir())
	if err != nil {
		if module.IsModuleNotFound(err) {
			mod, err = modMgr.AddModule(f.Dir())
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	lh.logger.Printf("opened module: %s", mod.Path)

	// We reparse because the file being opened may not match
	// (originally parsed) content on the disk
	// TODO: Do this only if we can verify the file differs?
	modMgr.EnqueueModuleOpWait(mod.Path, op.OpTypeParseModuleConfiguration)
	modMgr.EnqueueModuleOpWait(mod.Path, op.OpTypeParseVariables)
	modMgr.EnqueueModuleOpWait(mod.Path, op.OpTypeLoadModuleMetadata)
	modMgr.EnqueueModuleOpWait(mod.Path, op.OpTypeDecodeReferenceTargets)
	modMgr.EnqueueModuleOpWait(mod.Path, op.OpTypeDecodeReferenceOrigins)

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

	notifier, err := lsctx.DiagnosticsNotifier(ctx)
	if err != nil {
		return err
	}

	diags := diagnostics.NewDiagnostics()
	diags.EmptyRootDiagnostic()
	diags.Append("HCL", mod.ModuleDiagnostics.AsMap())
	diags.Append("HCL", mod.VarsDiagnostics.AutoloadedOnly().AsMap())
	if vf, ok := ast.NewVarsFilename(f.Filename()); ok && !vf.IsAutoloaded() {
		diags.Append("HCL", mod.VarsDiagnostics.ForFile(vf).AsMap())
	}

	notifier.PublishHCLDiags(ctx, mod.Path, diags)

	return nil
}
