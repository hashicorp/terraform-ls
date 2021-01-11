package handlers

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2/code"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/langserver/handlers/command"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

var handlers = cmd.Handlers{
	cmd.Name("rootmodules"):        command.ModulesHandler,
	cmd.Name("terraform.init"):     command.TerraformInitHandler,
	cmd.Name("terraform.validate"): command.TerraformValidateHandler,
}

func (lh *logHandler) WorkspaceExecuteCommand(ctx context.Context, params lsp.ExecuteCommandParams) (interface{}, error) {
	if params.Command == "editor.action.triggerSuggest" {
		// If this was actually received by the server, it means the client
		// does not support explicit suggest triggering, so we fail silently
		// TODO: Revisit once https://github.com/microsoft/language-server-protocol/issues/1117 is addressed
		return nil, nil
	}

	commandPrefix, _ := lsctx.CommandPrefix(ctx)
	handler, ok := handlers.Get(params.Command, commandPrefix)
	if !ok {
		return nil, fmt.Errorf("%w: command handler not found for %q", code.MethodNotFound.Err(), params.Command)
	}

	pt, ok := params.WorkDoneToken.(lsp.ProgressToken)
	if ok {
		ctx = lsctx.WithProgressToken(ctx, pt)
	}

	return handler(ctx, cmd.ParseCommandArgs(params.Arguments))
}
