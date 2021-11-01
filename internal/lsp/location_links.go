package lsp

import (
	"path/filepath"

	"github.com/hashicorp/hcl-lang/decoder"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func RefTargetToLocationLink(target *decoder.ReferenceTarget, linkSupport bool) interface{} {
	targetUri := uri.FromPath(filepath.Join(target.Path.Path, target.Range.Filename))

	if linkSupport {
		locLink := lsp.LocationLink{
			OriginSelectionRange: HCLRangeToLSP(target.OriginRange),
			TargetURI:            lsp.DocumentURI(targetUri),
			TargetRange:          HCLRangeToLSP(target.Range),
		}

		if target.DefRangePtr != nil {
			locLink.TargetSelectionRange = HCLRangeToLSP(*target.DefRangePtr)
		}

		return locLink
	}

	return lsp.Location{
		URI:   lsp.DocumentURI(targetUri),
		Range: HCLRangeToLSP(target.Range),
	}
}
