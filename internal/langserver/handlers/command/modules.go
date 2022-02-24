package command

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/creachadair/jrpc2/code"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/state"
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

func (h *CmdHandler) ModulesHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	walker, err := lsctx.ModuleWalker(ctx)
	if err != nil {
		return nil, err
	}

	docUri, ok := args.GetString("uri")
	if !ok || docUri == "" {
		return nil, fmt.Errorf("%w: expected module uri argument to be set", code.InvalidParams.Err())
	}

	dh := document.HandleFromURI(docUri)

	doneLoading := !walker.IsWalking()

	var sources []SchemaSource
	sources, err = h.schemaSourcesForModule(dh.Dir.Path())
	if err != nil {
		if state.IsModuleNotFound(err) {
			err := h.StateStore.Modules.Add(dh.Dir.Path())
			if err != nil {
				return nil, err
			}
			sources, err = h.schemaSourcesForModule(dh.Dir.Path())
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
			URI:  uri.FromPath(source.Path),
			Name: humanReadablePath(rootDir, source.Path),
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

// schemaSourcesForModule is DEPRECATED and should NOT be used anymore
// it is just maintained for backwards compatibility in the "rootmodules"
// custom LSP command which itself will be DEPRECATED as external parties
// should not need to know where does a matched schema come from in practice
func (h *CmdHandler) schemaSourcesForModule(modPath string) ([]SchemaSource, error) {
	ok, err := h.moduleHasAnyLocallySourcedSchema(modPath)
	if err != nil {
		return nil, err
	}
	if ok {
		return []SchemaSource{
			{Path: modPath},
		}, nil
	}

	callers, err := h.StateStore.Modules.CallersOfModule(modPath)
	if err != nil {
		return nil, err
	}

	sources := make([]SchemaSource, 0)
	for _, modCaller := range callers {
		ok, err := h.moduleHasAnyLocallySourcedSchema(modCaller.Path)
		if err != nil {
			return nil, err
		}
		if ok {
			sources = append(sources, SchemaSource{
				Path: modCaller.Path,
			})
		}

	}

	return sources, nil
}

func (h *CmdHandler) moduleHasAnyLocallySourcedSchema(modPath string) (bool, error) {
	si, err := h.StateStore.ProviderSchemas.ListSchemas()
	if err != nil {
		return false, err
	}

	for ps := si.Next(); ps != nil; ps = si.Next() {
		if lss, ok := ps.Source.(state.LocalSchemaSource); ok {
			if lss.ModulePath == modPath {
				return true, nil
			}
		}
	}

	return false, nil
}

type SchemaSource struct {
	Path              string
	HumanReadablePath string
}

func humanReadablePath(rootDir, modPath string) string {
	if rootDir == "" {
		return modPath
	}

	// absolute paths can be too long for UI/messages,
	// so we just display relative to root dir
	relDir, err := filepath.Rel(rootDir, modPath)
	if err != nil {
		return modPath
	}

	if relDir == "." {
		// Name of the root dir is more helpful than "."
		return filepath.Base(rootDir)
	}

	return relDir
}
