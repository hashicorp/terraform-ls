// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/langserver/handlers/command"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
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

	cmdHandler := &command.CmdHandler{
		StateStore: svc.stateStore,
	}
	_, err = cmdHandler.TerraformValidateHandler(ctx, cmd.CommandArgs{
		"uri": dh.Dir.URI,
	})

	return err
}
