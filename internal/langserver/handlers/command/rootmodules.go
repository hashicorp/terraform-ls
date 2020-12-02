package command

import (
	"context"
	"fmt"
	"sort"

	"github.com/creachadair/jrpc2/code"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/uri"
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

func RootModulesHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	walker, err := lsctx.RootModuleWalker(ctx)
	if err != nil {
		return nil, err
	}

	fileUri, ok := args.GetString("uri")
	if !ok || fileUri == "" {
		return nil, fmt.Errorf("%w: expected uri argument to be set", code.InvalidParams.Err())
	}

	fh := ilsp.FileHandlerFromDocumentURI(lsp.DocumentURI(fileUri))

	cf, err := lsctx.RootModuleFinder(ctx)
	if err != nil {
		return nil, err
	}
	doneLoading := !walker.IsWalking()
	candidates := cf.RootModuleCandidatesByPath(fh.Dir())

	rootModules := make([]rootModuleInfo, len(candidates))
	for i, candidate := range candidates {
		rootModules[i] = rootModuleInfo{
			URI: uri.FromPath(candidate.Path()),
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
