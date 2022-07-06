package command

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/uri"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/module"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

const moduleCallsVersion = 0

type moduleCallsResponse struct {
	FormatVersion int          `json:"v"`
	ModuleCalls   []moduleCall `json:"module_calls"`
}

type moduleCall struct {
	Name             string       `json:"name"`
	SourceAddr       string       `json:"source_addr"`
	Version          string       `json:"version,omitempty"`
	SourceType       ModuleType   `json:"source_type,omitempty"`
	DocsLink         string       `json:"docs_link,omitempty"`
	DependentModules []moduleCall `json:"dependent_modules"` // will always be an empty list, we keep this for compatibility
}

type ModuleType string

const (
	UNKNOWN    ModuleType = "unknown"
	TFREGISTRY ModuleType = "tfregistry"
	LOCAL      ModuleType = "local"
	GITHUB     ModuleType = "github"
	GIT        ModuleType = "git"
)

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
		sourceType := getModuleType(manifest.SourceAddr)

		docsLink := ""
		if sourceType == TFREGISTRY {
			var err error
			docsLink, err = getModuleDocumentationLink(ctx, manifest.SourceAddr.String(), manifest.Version.String())
			if err != nil {
				h.Logger.Printf("failed to get module docs link: %s", err)
			}
		}

		// build what we know
		moduleInfo := moduleCall{
			Name:             moduleName,
			SourceAddr:       manifest.SourceAddr.String(),
			DocsLink:         docsLink,
			Version:          manifest.Version.String(),
			SourceType:       sourceType,
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
	shortName := strings.TrimPrefix(sourceAddr, "registry.terraform.io/")

	rawURL := fmt.Sprintf(`https://registry.terraform.io/modules/%s/%s`, shortName, version)

	u, err := docsURL(ctx, rawURL, "workspace/executeCommand/module.calls")
	if err != nil {
		return "", err
	}

	return u.String(), nil
}

// GetModuleType checks source addresses to determine what kind of source the Terraform module comes
// from. It currently supports detecting Terraform Registry modules, GitHub modules, Git modules, and
// local file paths
func getModuleType(sourceAddr tfmod.ModuleSourceAddr) ModuleType {
	// Example: terraform-aws-modules/ec2-instance/aws
	// Example: registry.terraform.io/terraform-aws-modules/vpc/aws
	_, ok := sourceAddr.(tfaddr.Module)
	if ok {
		return TFREGISTRY
	}

	_, ok = sourceAddr.(module.LocalSourceAddr)
	if ok {
		return LOCAL
	}

	// Example: github.com/terraform-aws-modules/terraform-aws-security-group
	if strings.HasPrefix(sourceAddr.String(), "github.com/") {
		return GITHUB
	}

	// Example: git::https://example.com/vpc.git
	if strings.HasPrefix(sourceAddr.String(), "git::") {
		return GIT
	}

	return UNKNOWN
}
