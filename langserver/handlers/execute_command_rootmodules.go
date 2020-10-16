package handlers

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2/code"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/sourcegraph/go-lsp"
)

const rootmodulesCommandResponseVersion = 0
const rootmodulesCommandFileArgNotFound code.Code = -32004

type rootmodulesCommandResponse struct {
	Version     int              `json:"version"`
	DoneLoading bool             `json:"doneLoading"`
	RootModules []rootModuleInfo `json:"rootModules"`
}

type rootModuleInfo struct {
	Path string `json:"path"`
}

func executeCommandRootModulesHandler(ctx context.Context, args commandArgs) (interface{}, error) {
	walker, err := lsctx.RootModuleWalker(ctx)
	if err != nil {
		return nil, err
	}

	file, ok := args.GetString("file")
	if !ok || file == "" {
		return nil, fmt.Errorf("%w: expected file argument to be set", rootmodulesCommandFileArgNotFound.Err())
	}

	uri := lsp.DocumentURI(file)
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
		Version:     rootmodulesCommandResponseVersion,
		DoneLoading: doneLoading,
		RootModules: rootModules,
	}, nil
}
