package command

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/hashicorp/terraform-ls/internal/uri"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

const moduleCallsVersion = 0

type moduleCallsResponse struct {
	FormatVersion int          `json:"v"`
	ModuleCalls   []moduleCall `json:"module_calls"`
}

type moduleCall struct {
	Name             string             `json:"name"`
	SourceAddr       string             `json:"source_addr"`
	Version          string             `json:"version,omitempty"`
	SourceType       datadir.ModuleType `json:"source_type,omitempty"`
	DocsLink         string             `json:"docs_link,omitempty"`
	DependentModules []moduleCall       `json:"dependent_modules"` // will always be an empty list, we keep this for compatibility
}

func (h *CmdHandler) ModuleCallsHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	response := moduleCallsResponse{
		FormatVersion: moduleCallsVersion,
		ModuleCalls:   make([]moduleCall, 0),
	}

	modUri, ok := args.GetString("uri")
	if !ok || modUri == "" {
		return response, fmt.Errorf("%w: expected module uri argument to be set", code.InvalidParams.Err())
	}

	if !uri.IsURIValid(modUri) {
		return response, fmt.Errorf("URI %q is not valid", modUri)
	}

	modPath, err := uri.PathFromURI(modUri)
	if err != nil {
		return response, err
	}

	moduleCalls, err := h.StateStore.Modules.ModuleCalls(modPath)
	if err != nil {
		return response, err
	}

	response.ModuleCalls = h.parseModuleRecords(ctx, moduleCalls)

	return response, nil
}

func (h *CmdHandler) parseModuleRecords(ctx context.Context, moduleCalls tfmod.ModuleCalls) []moduleCall {
	modules := make(map[string]moduleCall)
	for _, manifest := range moduleCalls.Declared {
		if manifest.SourceAddr == nil {
			// We skip all modules without a source address
			continue
		}

		moduleName := manifest.LocalName
		docsLink, err := getModuleDocumentationLink(ctx, manifest.SourceAddr.String(), manifest.Version.String())
		if err != nil {
			h.Logger.Printf("failed to get module docs link: %s", err)
		}

		// build what we know
		moduleInfo := moduleCall{
			Name:             moduleName,
			SourceAddr:       manifest.SourceAddr.String(),
			DocsLink:         docsLink,
			Version:          manifest.Version.String(),
			SourceType:       datadir.GetModuleType(manifest.SourceAddr.String()),
			DependentModules: make([]moduleCall, 0),
		}

		modules[moduleName] = moduleInfo
	}

	// don't need the map anymore, return a list of modules found
	list := make([]moduleCall, 0)
	for _, mo := range modules {
		list = append(list, mo)
	}

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})

	return list
}

func getModuleDocumentationLink(ctx context.Context, sourceAddr string, version string) (string, error) {
	if datadir.GetModuleType(sourceAddr) != datadir.TFREGISTRY {
		return "", nil
	}

	shortName := strings.TrimPrefix(sourceAddr, "registry.terraform.io/")

	rawURL := fmt.Sprintf(`https://registry.terraform.io/modules/%s/%s`, shortName, version)

	u, err := docsURL(ctx, rawURL, "workspace/executeCommand/module.calls")
	if err != nil {
		return "", err
	}

	return u.String(), nil
}
