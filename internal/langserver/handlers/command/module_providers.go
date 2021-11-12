package command

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2/code"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

const moduleProvidersVersion = 0

type moduleProvidersResponse struct {
	FormatVersion        int                            `json:"v"`
	ProviderRequirements map[string]providerRequirement `json:"provider_requirements"`
	InstalledProviders   map[string]string              `json:"installed_providers"`
}

type providerRequirement struct {
	DisplayName       string `json:"display_name"`
	VersionConstraint string `json:"version_constraint"`
}

func ModuleProvidersHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	response := moduleProvidersResponse{
		FormatVersion:        moduleProvidersVersion,
		ProviderRequirements: make(map[string]providerRequirement),
		InstalledProviders:   make(map[string]string),
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

	mm, err := lsctx.ModuleFinder(ctx)
	if err != nil {
		return response, err
	}

	found, _ := mm.ModuleByPath(modPath)
	if found == nil {
		return response, nil
	}

	for provider, version := range found.Meta.ProviderRequirements {
		response.ProviderRequirements[provider.String()] = providerRequirement{
			DisplayName:       provider.ForDisplay(),
			VersionConstraint: version.String(),
		}
	}

	for provider, version := range found.InstalledProviders {
		response.InstalledProviders[provider.String()] = version.String()
	}

	return response, nil
}
