package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/sourcegraph/go-lsp"
)

type rootmodulesCommandResponse struct {
	DoneLoading bool             `json:"doneLoading"`
	RootModules []rootModuleInfo `json:"rootModules"`
}

type rootModuleInfo struct {
	Path string `json:"path"`
}

func executeCommandRootModulesHandler(ctx context.Context, params lsp.ExecuteCommandParams) (interface{}, error) {
	walker, err := lsctx.RootModuleWalker(ctx)
	if err != nil {
		return nil, err
	}

	uri := lsp.DocumentURI(params.Arguments[0].(string))
	fh := ilsp.FileHandlerFromDocumentURI(uri)

	cf, err := lsctx.RootModuleCandidateFinder(ctx)
	if err != nil {
		return nil, err
	}
	doneLoading := !walker.IsWalking()
	candidates := cf.RootModuleCandidatesByPath(fh.Dir())

	rootModules := make([]rootModuleInfo, len(candidates))
	for i, candidate := range candidates {
		rootModules[i] = rootModuleInfo{
			Path: candidate.Path(),
		}
	}
	return rootmodulesCommandResponse{
		DoneLoading: doneLoading,
		RootModules: rootModules,
	}, nil
}
