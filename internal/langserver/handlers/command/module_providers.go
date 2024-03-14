// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/uri"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

const moduleProvidersVersion = 0

type moduleProvidersResponse struct {
	FormatVersion        int                            `json:"v"`
	ProviderRequirements map[string]providerRequirement `json:"provider_requirements"`
	InstalledProviders   map[string]string              `json:"installed_providers"`
}

type providerRequirement struct {
	DisplayName       string `json:"display_name"`
	VersionConstraint string `json:"version_constraint,omitempty"`
	DocsLink          string `json:"docs_link,omitempty"`
}

func (h *CmdHandler) ModuleProvidersHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	response := moduleProvidersResponse{
		FormatVersion:        moduleProvidersVersion,
		ProviderRequirements: make(map[string]providerRequirement),
		InstalledProviders:   make(map[string]string),
	}

	modUri, ok := args.GetString("uri")
	if !ok || modUri == "" {
		return response, fmt.Errorf("%w: expected module uri argument to be set", jrpc2.InvalidParams.Err())
	}

	if !uri.IsURIValid(modUri) {
		return response, fmt.Errorf("URI %q is not valid", modUri)
	}

	modPath, err := uri.PathFromURI(modUri)
	if err != nil {
		return response, err
	}

	mod, _ := h.StateStore.Modules.ModuleByPath(modPath)
	if mod == nil {
		return response, nil
	}

	for provider, version := range mod.Meta.ProviderRequirements {
		docsLink, err := getProviderDocumentationLink(ctx, provider)
		if err != nil {
			return response, err
		}
		response.ProviderRequirements[provider.String()] = providerRequirement{
			DisplayName:       provider.ForDisplay(),
			VersionConstraint: version.String(),
			DocsLink:          docsLink,
		}
	}

	// TODO!
	// for provider, version := range mod.InstalledProviders {
	// 	response.InstalledProviders[provider.String()] = version.String()
	// }

	return response, nil
}

func getProviderDocumentationLink(ctx context.Context, provider tfaddr.Provider) (string, error) {
	if provider.IsLegacy() || provider.IsBuiltIn() || provider.Hostname != "registry.terraform.io" {
		return "", nil
	}

	rawURL := fmt.Sprintf(`https://registry.terraform.io/providers/%s/latest`, provider.ForDisplay())

	u, err := docsURL(ctx, rawURL, "workspace/executeCommand/module.providers")
	if err != nil {
		return "", err
	}

	return u.String(), nil
}
