package lsp

import (
	"path/filepath"

	"github.com/hashicorp/hcl-lang/lang"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func RefOriginsToLocations(originModPath string, origins lang.ReferenceOrigins) []lsp.Location {
	locations := make([]lsp.Location, len(origins))

	for i, origin := range origins {
		originUri := uri.FromPath(filepath.Join(originModPath, origin.Range.Filename))
		locations[i] = lsp.Location{
			URI:   lsp.DocumentURI(originUri),
			Range: HCLRangeToLSP(origin.Range),
		}
	}

	return locations
}
