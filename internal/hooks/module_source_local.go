package hooks

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/zclconf/go-cty/cty"
)

func (h *Hooks) LocalModuleSources(ctx context.Context, value cty.Value) ([]decoder.Candidate, error) {
	candidates := make([]decoder.Candidate, 0)

	modules, err := h.ModStore.List()
	path, ok := decoder.PathFromContext(ctx)
	if err != nil || !ok {
		return candidates, err
	}

	for _, mod := range modules {
		if strings.Contains(mod.Path, datadir.DataDirName) {
			// Skip anything from the data directory
			continue
		}
		if mod.Path == path.Path {
			// Exclude the current module
			continue
		}

		relPath, err := filepath.Rel(path.Path, mod.Path)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(relPath, ".") {
			relPath = fmt.Sprintf("./%s", relPath)
		}
		relPath = strings.ReplaceAll(relPath, "\\", "/")
		c := decoder.ExpressionCompletionCandidate(decoder.ExpressionCandidate{
			Value:  cty.StringVal(relPath),
			Detail: "local",
		})
		candidates = append(candidates, c)
	}

	return candidates, nil
}
