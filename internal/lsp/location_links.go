package lsp

import (
	"path/filepath"

	"github.com/hashicorp/hcl-lang/lang"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func ReferenceToLocationLink(targetModPath string, origin lang.ReferenceOrigin,
	target *lang.ReferenceTarget, linkSupport bool) interface{} {

	if target == nil || target.RangePtr == nil {
		return nil
	}

	targetUri := uri.FromPath(filepath.Join(targetModPath, target.RangePtr.Filename))

	if linkSupport {
		return lsp.LocationLink{
			OriginSelectionRange: HCLRangeToLSP(origin.Range),
			TargetURI:            lsp.DocumentURI(targetUri),
			TargetRange:          HCLRangeToLSP(*target.RangePtr),
			TargetSelectionRange: HCLRangeToLSP(*target.RangePtr),
		}
	}

	return lsp.Location{
		URI:   lsp.DocumentURI(targetUri),
		Range: HCLRangeToLSP(*target.RangePtr),
	}
}
