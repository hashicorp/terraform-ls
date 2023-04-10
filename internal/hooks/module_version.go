// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hooks

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl/v2"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	"github.com/zclconf/go-cty/cty"
)

func getModuleSourceAddr(moduleCalls map[string]tfmod.DeclaredModuleCall, pos hcl.Pos, filename string) (tfmod.ModuleSourceAddr, bool) {
	for _, mc := range moduleCalls {
		if mc.RangePtr == nil {
			// This can only happen if the file is JSON
			// In this case we're not providing completion anyway
			continue
		}
		if mc.RangePtr.ContainsPos(pos) && mc.RangePtr.Filename == filename {
			return mc.SourceAddr, true
		}
	}

	return nil, false
}

func (h *Hooks) RegistryModuleVersions(ctx context.Context, value cty.Value) ([]decoder.Candidate, error) {
	candidates := make([]decoder.Candidate, 0)

	path, ok := decoder.PathFromContext(ctx)
	if !ok {
		return candidates, errors.New("missing context: path")
	}
	pos, ok := decoder.PosFromContext(ctx)
	if !ok {
		return candidates, errors.New("missing context: pos")
	}
	filename, ok := decoder.FilenameFromContext(ctx)
	if !ok {
		return candidates, errors.New("missing context: filename")
	}
	maxCandidates, ok := decoder.MaxCandidatesFromContext(ctx)
	if !ok {
		return candidates, errors.New("missing context: maxCandidates")
	}

	module, err := h.ModStore.ModuleByPath(path.Path)
	if err != nil {
		return candidates, err
	}

	sourceAddr, ok := getModuleSourceAddr(module.Meta.ModuleCalls, pos, filename)
	if !ok {
		return candidates, nil
	}
	registryAddr, ok := sourceAddr.(tfaddr.Module)
	if !ok {
		// Trying to complete version on local or external module
		return candidates, nil
	}

	versions, err := h.RegistryClient.GetModuleVersions(ctx, registryAddr)
	if err != nil {
		return candidates, err
	}

	for i, v := range versions {
		if uint(i) >= maxCandidates {
			return candidates, nil
		}

		c := decoder.ExpressionCompletionCandidate(decoder.ExpressionCandidate{
			Value: cty.StringVal(v.String()),
		})
		// We rely on the fact that hcl-lang limits number of candidates
		// to 100, so padding with <=3 zeros provides naive but good enough
		// way to reliably "lexicographically" sort the versions as there's
		// no better way to do it in LSP.
		c.SortText = fmt.Sprintf("%3d", i)

		candidates = append(candidates, c)
	}

	return candidates, nil
}
