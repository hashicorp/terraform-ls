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
	DependentModules []moduleCall       `json:"dependent_modules"`
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

	found, _ := h.StateStore.Modules.ModuleByPath(modPath)
	if found == nil {
		return response, nil
	}

	if found.ModManifest == nil {
		return response, nil
	}

	modCalls, err := parseModuleRecords(ctx, found.ModManifest.Records)
	if err != nil {
		return response, err
	}

	response.ModuleCalls = modCalls

	return response, nil
}

func parseModuleRecords(ctx context.Context, records []datadir.ModuleRecord) ([]moduleCall, error) {
	// sort all records by key so that dependent modules are found
	// after primary modules
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].Key < records[j].Key
	})

	modules := make(map[string]moduleCall)
	for _, manifest := range records {
		if manifest.IsRoot() {
			// this is the current directory, which is technically a module
			// skipping as it's not relevant in the activity bar (yet?)
			continue
		}

		moduleName := manifest.Key
		subModuleName := ""

		// determine if this module is nested in another module
		// in the current workspace by finding a period in the moduleName
		// is it better to look at SourceAddr and compare?
		if strings.Contains(manifest.Key, ".") {
			v := strings.Split(manifest.Key, ".")
			moduleName = v[0]
			subModuleName = v[1]
		}

		docsLink, err := getModuleDocumentationLink(ctx, manifest)
		if err != nil {
			return nil, err
		}

		// build what we know
		moduleInfo := moduleCall{
			Name:             moduleName,
			SourceAddr:       manifest.SourceAddr,
			DocsLink:         docsLink,
			Version:          manifest.VersionStr,
			SourceType:       manifest.GetModuleType(),
			DependentModules: make([]moduleCall, 0),
		}

		m, present := modules[moduleName]
		if present {
			// this module is located inside another so append
			moduleInfo.Name = subModuleName
			m.DependentModules = append(m.DependentModules, moduleInfo)
			modules[moduleName] = m
		} else {
			// this is the first we've seen module
			modules[moduleName] = moduleInfo
		}
	}

	// don't need the map anymore, return a list of modules found
	list := make([]moduleCall, 0)
	for _, mo := range modules {
		list = append(list, mo)
	}

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})

	return list, nil
}

func getModuleDocumentationLink(ctx context.Context, record datadir.ModuleRecord) (string, error) {
	if record.GetModuleType() != datadir.TFREGISTRY {
		return "", nil
	}

	shortName := strings.TrimPrefix(record.SourceAddr, "registry.terraform.io/")

	rawURL := fmt.Sprintf(`https://registry.terraform.io/modules/%s/%s`, shortName, record.VersionStr)

	u, err := docsURL(ctx, rawURL, "workspace/executeCommand/module.calls")
	if err != nil {
		return "", err
	}

	return u.String(), nil
}
