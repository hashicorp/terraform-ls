package handlers

import (
	"context"
	"fmt"

	lsp "github.com/sourcegraph/go-lsp"
)

type executeCommandHandler func(context.Context, lsp.ExecuteCommandParams) (interface{}, error)

var executeCommandHandlers = map[string]executeCommandHandler{
	"rootmodules": executeCommandRootModulesHandler,
}

func (lh *logHandler) WorkspaceExecuteCommand(ctx context.Context, params lsp.ExecuteCommandParams) (interface{}, error) {
	handler, ok := executeCommandHandlers[params.Command]
	if !ok {
		return nil, fmt.Errorf("No workspace/executeCommand handler for command: %q", params.Command)
	}
	return handler(ctx, params)
}
