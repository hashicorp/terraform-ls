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
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

const modulesCommandResponseVersion = 0

type modulesCommandResponse struct {
	ResponseVersion int          `json:"responseVersion"`
	DoneLoading     bool         `json:"doneLoading"`
	Modules         []moduleInfo `json:"rootModules"`
}

type moduleInfo struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

func ModulesHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	walker, err := lsctx.ModuleWalker(ctx)
	if err != nil {
		return nil, err
	}

	fileUri, ok := args.GetString("uri")
	if !ok || fileUri == "" {
		return nil, fmt.Errorf("%w: expected uri argument to be set", code.InvalidParams.Err())
	}

	fh := ilsp.FileHandlerFromDocumentURI(lsp.DocumentURI(fileUri))

	modMgr, err := lsctx.ModuleManager(ctx)
	if err != nil {
		return nil, err
	}

	doneLoading := !walker.IsWalking()

	var sources []module.SchemaSource
	sources, err = modMgr.SchemaSourcesForModule(fh.Dir())
	if err != nil {
		if module.IsModuleNotFound(err) {
			_, err := modMgr.AddModule(fh.Dir())
			if err != nil {
				return nil, err
			}
			sources, err = modMgr.SchemaSourcesForModule(fh.Dir())
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	rootDir, _ := lsctx.RootDirectory(ctx)

	modules := make([]moduleInfo, len(sources))
	for i, source := range sources {
		modules[i] = moduleInfo{
			URI:  uri.FromPath(source.Path()),
			Name: source.HumanReadablePath(rootDir),
		}
	}
	sort.SliceStable(modules, func(i, j int) bool {
		return modules[i].URI < modules[j].URI
	})
	return modulesCommandResponse{
		ResponseVersion: modulesCommandResponseVersion,
		DoneLoading:     doneLoading,
		Modules:         modules,
	}, nil
}
