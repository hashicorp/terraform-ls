// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package hooks

import (
	"context"
	"fmt"
	"os"
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
		dirName := fmt.Sprintf("%c%s%c", os.PathSeparator, datadir.DataDirName, os.PathSeparator)
		if strings.Contains(mod.Path(), dirName) {
			// Skip installed module copies in cache directories
			continue
		}
		if mod.Path() == path.Path {
			// Exclude the module we're providing completion in
			// to avoid cyclic references
			continue
		}

		relPath, err := filepath.Rel(path.Path, mod.Path())
		if err != nil {
			continue
		}
		if !strings.HasPrefix(relPath, "..") {
			// filepath.Rel will return the cleaned relative path, but Terraform
			// expects local module sources to start with ./
			relPath = "./" + relPath
		}
		relPath = filepath.ToSlash(relPath)
		c := decoder.ExpressionCompletionCandidate(decoder.ExpressionCandidate{
			Value:  cty.StringVal(relPath),
			Detail: "local",
		})
		candidates = append(candidates, c)
	}

	return candidates, nil
}
