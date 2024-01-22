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

type VarsReader interface {
	VarsByPath(modPath string) (*state.Vars, error)
	List() ([]*state.Vars, error)
}

type PathReader struct {
	ModuleReader ModuleReader
	VarsReader   VarsReader
	SchemaReader state.SchemaReader
}

var _ decoder.PathReader = &PathReader{}

func (mr *PathReader) LanguageIDs() []string {
	return []string{
		ilsp.Terraform.String(),
		ilsp.Tfvars.String(),
	}
}

func (mr *PathReader) Paths(ctx context.Context, languageID string) []lang.Path {
	paths := make([]lang.Path, 0)

	// TODO! This logic is flawed. We want to end up with different language id paths
	// for the SAME directory. This is what allows cross file references to work.

	switch languageID {
	case ilsp.Terraform.String():
		modList, err := mr.ModuleReader.List()
		if err != nil {
			return paths
		}

		for _, mod := range modList {
			paths = append(paths, lang.Path{
				Path:       mod.Path(),
				LanguageID: ilsp.Terraform.String(),
			})
		}

	case ilsp.Tfvars.String():
		varList, err := mr.VarsReader.List()
		if err != nil {
			return paths
		}

		for _, mod := range varList {
			paths = append(paths, lang.Path{
				Path:       mod.Path(),
				LanguageID: ilsp.Tfvars.String(),
			})
		}
	}

	return paths
}

func (mr *PathReader) PathContext(path lang.Path) (*decoder.PathContext, error) {
	switch path.LanguageID {
	case ilsp.Terraform.String():
		mod, err := mr.ModuleReader.ModuleByPath(path.Path)
		if err != nil {
			return nil, err
		}
		return modulePathContext(mod, mr.SchemaReader, mr.ModuleReader)

	case ilsp.Tfvars.String():
		mod, err := mr.VarsReader.VarsByPath(path.Path)
		if err != nil {
			return nil, err
		}
		return varsPathContext(mod, mr.ModuleReader)
	}

	return nil, fmt.Errorf("unknown language ID: %q", path.LanguageID)
}
