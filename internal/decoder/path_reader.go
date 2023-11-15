// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/state"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	"github.com/hashicorp/terraform-schema/registry"
)

type ModuleReader interface {
	ModuleByPath(modPath string) (*state.Module, error)
	List() ([]*state.Module, error)
	ModuleCalls(modPath string) (tfmod.ModuleCalls, error)
	LocalModuleMeta(modPath string) (*tfmod.Meta, error)
	RegistryModuleMeta(addr tfaddr.Module, cons version.Constraints) (*registry.ModuleData, error)
}

type PathReader struct {
	ModuleReader ModuleReader
	SchemaReader state.SchemaReader
}

var _ decoder.PathReader = &PathReader{}

func (mr *PathReader) Paths(ctx context.Context) []lang.Path {
	paths := make([]lang.Path, 0)

	modList, err := mr.ModuleReader.List()
	if err != nil {
		return paths
	}

	langId, hasLang := LanguageId(ctx)

	for _, mod := range modList {
		if hasLang {
			paths = append(paths, lang.Path{
				Path:       mod.Path,
				LanguageID: langId.String(),
			})
			continue
		}

		paths = append(paths, lang.Path{
			Path:       mod.Path,
			LanguageID: ilsp.Terraform.String(),
		})
		if len(mod.ParsedVarsFiles) > 0 {
			paths = append(paths, lang.Path{
				Path:       mod.Path,
				LanguageID: ilsp.Tfvars.String(),
			})
		}
		if len(mod.ParsedTestFiles) > 0 {
			paths = append(paths, lang.Path{
				Path:       mod.Path,
				LanguageID: ilsp.Tftest.String(),
			})
		}
	}

	return paths
}

func (mr *PathReader) PathContext(path lang.Path) (*decoder.PathContext, error) {
	mod, err := mr.ModuleReader.ModuleByPath(path.Path)
	if err != nil {
		return nil, err
	}

	switch path.LanguageID {
	case ilsp.Terraform.String():
		return modulePathContext(mod, mr.SchemaReader, mr.ModuleReader)
	case ilsp.Tfvars.String():
		return varsPathContext(mod)
	case ilsp.Tftest.String():
		return testPathContext(mod)
	}

	return nil, fmt.Errorf("unknown language ID: %q", path.LanguageID)
}
