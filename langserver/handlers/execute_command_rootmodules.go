package handlers

import (
	"context"
	"fmt"
	"sort"

	"github.com/creachadair/jrpc2/code"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/sourcegraph/go-lsp"
)

const rootmodulesCommandResponseVersion = 0

type rootmodulesCommandResponse struct {
	ResponseVersion int              `json:"responseVersion"`
	DoneLoading     bool             `json:"doneLoading"`
	RootModules     []rootModuleInfo `json:"rootModules"`
}

type rootModuleInfo struct {
	URI string `json:"uri"`
}

func executeCommandRootModulesHandler(ctx context.Context, args commandArgs) (interface{}, error) {
	walker, err := lsctx.RootModuleWalker(ctx)
	if err != nil {
		return nil, err
	}

	uri, ok := args.GetString("uri")
	if !ok || uri == "" {
		return nil, fmt.Errorf("%w: expected uri argument to be set", code.InvalidParams.Err())
	}

	fh := ilsp.FileHandlerFromDocumentURI(lsp.DocumentURI(uri))

	cf, err := lsctx.RootModuleCandidateFinder(ctx)
	if err != nil {
		return nil, err
	}
	doneLoading := !walker.IsWalking()
	candidates := cf.RootModuleCandidatesByPath(fh.Dir())

	rootModules := make([]rootModuleInfo, len(candidates))
	for i, candidate := range candidates {
		rootModules[i] = rootModuleInfo{
			URI: filesystem.URIFromPath(candidate.Path()),
		}
	}
	sort.SliceStable(rootModules, func(i, j int) bool {
		return rootModules[i].URI < rootModules[j].URI
	})
	return rootmodulesCommandResponse{
		ResponseVersion: rootmodulesCommandResponseVersion,
		DoneLoading:     doneLoading,
		RootModules:     rootModules,
	}, nil
}
