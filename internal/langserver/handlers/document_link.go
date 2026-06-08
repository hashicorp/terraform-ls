// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/document"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
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

	// Add resource documentation links
	resourceLinks, err := svc.generateResourceLinks(dh)
	if err != nil {
		// Don't fail the entire request if resource link generation fails
		// Just continue with existing provider links
	} else {
		links = append(links, resourceLinks...)
	}

	return ilsp.Links(links, cc.TextDocument.DocumentLink), nil
}

// generateResourceLinks parses the Terraform file to find resource blocks
// and generates documentation links for them
func (svc *service) generateResourceLinks(dh document.Handle) ([]lang.Link, error) {
	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return nil, err
	}

	f, diags := hclsyntax.ParseConfig(doc.Text, doc.Filename, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}

	var links []lang.Link

	for _, block := range f.Body.(*hclsyntax.Body).Blocks {
		var urlPath string
		var blockTypeDescription string

		if block.Type == "resource" && len(block.Labels) >= 1 {
			urlPath = "r"
			blockTypeDescription = "resource"
		} else if block.Type == "data" && len(block.Labels) >= 1 {
			urlPath = "d"
			blockTypeDescription = "data source"
		} else {
			continue // Skip other block types
		}

		resourceType := block.Labels[0]

		// Extract provider name and resource name from resource type
		// e.g., "azurerm_resource_group" -> provider: "azurerm", resource: "resource_group"
		parts := strings.SplitN(resourceType, "_", 2)
		if len(parts) != 2 {
			continue // Skip malformed resource types
		}

		providerName := parts[0]
		resourceName := parts[1]

		// Generate documentation URL
		docURL := fmt.Sprintf("https://www.terraform.io/docs/providers/%s/%s/%s.html", providerName, urlPath, resourceName)

		// Add UTM parameters similar to existing provider links
		u, err := url.Parse(docURL)
		if err != nil {
			continue
		}

		q := u.Query()
		q.Set("utm_source", "terraform-ls")
		q.Set("utm_content", "documentLink")
		u.RawQuery = q.Encode()

		// Create link for the resource type (first label)
		if len(block.LabelRanges) > 0 {
			links = append(links, lang.Link{
				URI:     u.String(),
				Tooltip: fmt.Sprintf("View documentation for %s %s", blockTypeDescription, resourceType),
				Range:   block.LabelRanges[0],
			})
		}
	}

	return links, nil
}
