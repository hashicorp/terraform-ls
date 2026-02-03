// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"path/filepath"

	"github.com/hashicorp/hcl-lang/decoder"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func RefTargetsToDefinitionLocationLinks(targets decoder.ReferenceTargets, defCaps *lsp.DefinitionClientCapabilities) interface{} {
	if defCaps == nil {
		return RefTargetsToLocationLinks(targets, false)
	}
	return RefTargetsToLocationLinks(targets, defCaps.LinkSupport)
}

func RefTargetsToDeclarationLocationLinks(targets decoder.ReferenceTargets, declCaps *lsp.DeclarationClientCapabilities) interface{} {
	if declCaps == nil {
		return RefTargetsToLocationLinks(targets, false)
	}
	return RefTargetsToLocationLinks(targets, declCaps.LinkSupport)
}

func RefTargetsToLocationLinks(targets decoder.ReferenceTargets, linkSupport bool) interface{} {
	if linkSupport {
		links := make([]lsp.LocationLink, 0)
		for _, target := range targets {
			links = append(links, refTargetToLocationLink(target))
		}
		return links
	}

	locations := make([]lsp.Location, 0)
	for _, target := range targets {
		locations = append(locations, refTargetToLocation(target))
	}
	return locations
}

func refTargetToLocationLink(target *decoder.ReferenceTarget) lsp.LocationLink {
	targetUri := uri.FromPath(filepath.Join(target.Path.Path, target.Range.Filename))
	originRange := HCLRangeToLSP(target.OriginRange)

	locLink := lsp.LocationLink{
		OriginSelectionRange: &originRange,
		TargetURI:            lsp.DocumentURI(targetUri),
		TargetRange:          HCLRangeToLSP(target.Range),
		TargetSelectionRange: HCLRangeToLSP(target.Range),
	}

	if target.DefRangePtr != nil {
		locLink.TargetSelectionRange = HCLRangeToLSP(*target.DefRangePtr)
	}

	return locLink
}

func refTargetToLocation(target *decoder.ReferenceTarget) lsp.Location {
	targetUri := uri.FromPath(filepath.Join(target.Path.Path, target.Range.Filename))

	return lsp.Location{
		URI:   lsp.DocumentURI(targetUri),
		Range: HCLRangeToLSP(target.Range),
	}
}
