// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/document"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

func (svc *service) TextDocumentLink(ctx context.Context, params lsp.DocumentLinkParams) ([]lsp.DocumentLink, error) {
	cc, err := ilsp.ClientCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)
	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return nil, err
	}

	if doc.LanguageID != ilsp.Terraform.String() {
		return nil, nil
	}

	jobIds, err := svc.stateStore.JobStore.ListIncompleteJobsForDir(dh.Dir)
	if err != nil {
		return nil, err
	}
	svc.stateStore.JobStore.WaitForJobs(ctx, jobIds...)

	d, err := svc.decoderForDocument(ctx, doc)
	if err != nil {
		return nil, err
	}

	links, err := d.LinksInFile(doc.Filename)
	if err != nil {
		return nil, err
	}

	docLinks, err := svc.generateDocLinks(dh)
	if err == nil {
		links = append(links, docLinks...)
	}

	return ilsp.Links(links, cc.TextDocument.DocumentLink), nil
}

// generateDocLinks parses the Terraform file to find resource and data blocks and generates documentation links for their type labels.
func (svc *service) generateDocLinks(dh document.Handle) ([]lang.Link, error) {
	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return nil, err
	}

	f, diagnostics := hclsyntax.ParseConfig(doc.Text, doc.Filename, hcl.InitialPos)
	if diagnostics.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diagnostics.Error())
	}

	var links []lang.Link

	for _, block := range f.Body.(*hclsyntax.Body).Blocks {
		var docPath, blockTypeDescription string

		if block.Type == "resource" && len(block.Labels) >= 1 {
			docPath, blockTypeDescription = "resources", "resource"
		} else if block.Type == "data" && len(block.Labels) >= 1 {
			docPath, blockTypeDescription = "data-sources", "data source"
		} else {
			continue // Skip other block types.
		}

		resourceType := block.Labels[0]

		// Extract provider name and resource name from the resource type.
		// e.g. "azurerm_resource_group" -> provider: "azurerm", resource: "resource_group"
		parts := strings.SplitN(resourceType, "_", 2)
		if len(parts) != 2 {
			continue // Skip malformed resource types.
		}

		providerAddr, ver := svc.installedProviderForType(dh.Dir.Path(), parts[0])
		docURL, err := buildResourceDocURL(providerAddr, ver, docPath, parts[1])
		if err != nil {
			continue
		}

		// Create link for the resource type (first label).
		if len(block.LabelRanges) > 0 {
			links = append(links, lang.Link{
				URI:     docURL,
				Tooltip: fmt.Sprintf("View documentation for %s %s", blockTypeDescription, resourceType),
				Range:   block.LabelRanges[0],
			})
		}
	}

	return links, nil
}

// resourceDocURLAtPos returns the documentation URL and resource type if the given position
// falls within the first label of a resource or data block, otherwise returns empty strings.
func (svc *service) resourceDocURLAtPos(dh document.Handle, pos hcl.Pos) (docURL, resourceType string) {
	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return "", ""
	}

	f, diagnostics := hclsyntax.ParseConfig(doc.Text, doc.Filename, hcl.InitialPos)
	if diagnostics.HasErrors() {
		return "", ""
	}

	for _, block := range f.Body.(*hclsyntax.Body).Blocks {
		var docPath string

		if block.Type == "resource" && len(block.Labels) >= 1 {
			docPath = "resources"
		} else if block.Type == "data" && len(block.Labels) >= 1 {
			docPath = "data-sources"
		} else {
			continue
		}

		if len(block.LabelRanges) == 0 {
			continue
		}

		lr := block.LabelRanges[0]
		if pos.Byte < lr.Start.Byte || pos.Byte >= lr.End.Byte {
			continue
		}

		resourceType = block.Labels[0]
		parts := strings.SplitN(resourceType, "_", 2)
		if len(parts) != 2 {
			continue
		}

		providerAddr, ver := svc.installedProviderForType(dh.Dir.Path(), parts[0])
		docURL, err = buildResourceDocURL(providerAddr, ver, docPath, parts[1])
		if err != nil {
			return "", ""
		}

		return docURL, resourceType
	}

	return "", ""
}

// installedProviderForType returns the installed provider address and version for the given
// provider type (e.g. "aws"), or a partial address with only Type set when not found.
func (svc *service) installedProviderForType(modPath, providerType string) (tfaddr.Provider, *version.Version) {
	if svc.features == nil || svc.features.RootModules == nil {
		return tfaddr.Provider{Type: providerType}, nil
	}

	installedProviders, err := svc.features.RootModules.InstalledProviders(modPath)
	if err != nil {
		return tfaddr.Provider{Type: providerType}, nil
	}

	for provider, ver := range installedProviders {
		if provider.Type == providerType {
			return provider, ver
		}
	}

	return tfaddr.Provider{Type: providerType}, nil
}

// buildResourceDocURL constructs a versioned Terraform Registry documentation URL with UTM parameters.
// Falls back to the legacy terraform.io URL format when provider namespace is unavailable.
func buildResourceDocURL(providerAddr tfaddr.Provider, ver *version.Version, docPath, resourceName string) (string, error) {
	var rawURL string

	switch {
	case providerAddr.Namespace != "" && ver != nil:
		rawURL = fmt.Sprintf("https://registry.terraform.io/providers/%s/%s/%s/docs/%s/%s",
			providerAddr.Namespace, providerAddr.Type, ver.String(), docPath, resourceName)
	case providerAddr.Namespace != "":
		rawURL = fmt.Sprintf("https://registry.terraform.io/providers/%s/%s/latest/docs/%s/%s",
			providerAddr.Namespace, providerAddr.Type, docPath, resourceName)
	default:
		// Fallback: provider not installed or lock file absent — use legacy URL.
		urlPath := "r"
		if docPath == "data-sources" {
			urlPath = "d"
		}
		rawURL = fmt.Sprintf("https://www.terraform.io/docs/providers/%s/%s/%s.html",
			providerAddr.Type, urlPath, resourceName)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("utm_source", "terraform-ls")
	q.Set("utm_content", "documentLink")
	u.RawQuery = q.Encode()

	return u.String(), nil
}
