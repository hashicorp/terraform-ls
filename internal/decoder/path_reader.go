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
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

type StateReader interface {
	DeclaredModuleCalls(modPath string) (map[string]tfmod.DeclaredModuleCall, error)
	InstalledModuleCalls(modPath string) (map[string]tfmod.InstalledModuleCall, error)
	LocalModuleMeta(modPath string) (*tfmod.Meta, error)
	RegistryModuleMeta(addr tfaddr.Module, cons version.Constraints) (*registry.ModuleData, error)
	ProviderSchema(modPath string, addr tfaddr.Provider, vc version.Constraints) (*tfschema.ProviderSchema, error)
	TerraformVersion(modPath string) *version.Version

	ModuleRecordByPath(modPath string) (*state.ModuleRecord, error)
	VariableRecordByPath(modPath string) (*state.VariableRecord, error)

	ListModuleRecords() ([]*state.ModuleRecord, error)
	ListVariableRecords() ([]*state.VariableRecord, error)
}

type PathReader struct {
	StateReader StateReader
}

var _ decoder.PathReader = &PathReader{}

func (pr *PathReader) Paths(ctx context.Context) []lang.Path {
	paths := make([]lang.Path, 0)

	moduleRecords, err := pr.StateReader.ListModuleRecords()
	if err == nil {
		for _, record := range moduleRecords {
			paths = append(paths, lang.Path{
				Path:       record.Path(),
				LanguageID: ilsp.Terraform.String(),
			})
		}
	}

	variableRecords, err := pr.StateReader.ListVariableRecords()
	if err == nil {
		for _, record := range variableRecords {
			paths = append(paths, lang.Path{
				Path:       record.Path(),
				LanguageID: ilsp.Tfvars.String(),
			})
		}
	}

	return paths
}

// PathContext returns a PathContext for the given path based on the language ID.
func (pr *PathReader) PathContext(path lang.Path) (*decoder.PathContext, error) {
	switch path.LanguageID {
	case ilsp.Terraform.String():
		mod, err := pr.StateReader.ModuleRecordByPath(path.Path)
		if err != nil {
			return nil, err
		}
		return modulePathContext(mod, pr.StateReader)
	case ilsp.Tfvars.String():
		mod, err := pr.StateReader.VariableRecordByPath(path.Path)
		if err != nil {
			return nil, err
		}
		return varsPathContext(mod, pr.StateReader)
	}

	return nil, fmt.Errorf("unknown language ID: %q", path.LanguageID)
}
