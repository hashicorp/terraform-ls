package command

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/creachadair/jrpc2/code"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

const moduleCallersVersion = 0

type moduleCallersResponse struct {
	FormatVersion int            `json:"v"`
	Callers       []moduleCaller `json:"callers"`
}

type moduleCaller struct {
	URI          string `json:"uri"`
	RelativePath string `json:"rel_path"`
}

func ModuleCallersHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	modUri, ok := args.GetString("uri")
	if !ok || modUri == "" {
		return nil, fmt.Errorf("%w: expected uri argument to be set", code.InvalidParams.Err())
	}

	modPath, err := uri.PathFromURI(modUri)
	if err != nil {
		return nil, err
	}

	mf, err := lsctx.ModuleFinder(ctx)
	if err != nil {
		return nil, err
	}

	modCallers, err := mf.CallersOfModule(modPath)
	if err != nil {
		return nil, err
	}

	callers := make([]moduleCaller, 0)
	for _, caller := range modCallers {
		relPath, err := filepath.Rel(modPath, caller.Path)
		if err != nil {
			return nil, err
		}
		callers = append(callers, moduleCaller{
			URI:          uri.FromPath(caller.Path),
			RelativePath: relPath,
		})
	}
	sort.SliceStable(callers, func(i, j int) bool {
		return callers[i].URI < callers[j].URI
	})
	return moduleCallersResponse{
		FormatVersion: moduleCallersVersion,
		Callers:       callers,
	}, nil
}
