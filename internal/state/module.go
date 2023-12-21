// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/backend"
	tfmod "github.com/hashicorp/terraform-schema/module"
	"github.com/hashicorp/terraform-schema/registry"

	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

type ModuleMetadata struct {
	CoreRequirements     version.Constraints
	Backend              *tfmod.Backend
	Cloud                *backend.Cloud
	ProviderReferences   map[tfmod.ProviderRef]tfaddr.Provider
	ProviderRequirements tfmod.ProviderRequirements
	Variables            map[string]tfmod.Variable
	Outputs              map[string]tfmod.Output
	Filenames            []string
	ModuleCalls          map[string]tfmod.DeclaredModuleCall
}

func (mm ModuleMetadata) Copy() ModuleMetadata {
	newMm := ModuleMetadata{
		// version.Constraints is practically immutable once parsed
		CoreRequirements: mm.CoreRequirements,
		Filenames:        mm.Filenames,
	}

	if mm.Cloud != nil {
		newMm.Cloud = mm.Cloud
	}

	if mm.Backend != nil {
		newMm.Backend = &tfmod.Backend{
			Type: mm.Backend.Type,
			Data: mm.Backend.Data.Copy(),
		}
	}

	if mm.ProviderReferences != nil {
		newMm.ProviderReferences = make(map[tfmod.ProviderRef]tfaddr.Provider, len(mm.ProviderReferences))
		for ref, provider := range mm.ProviderReferences {
			newMm.ProviderReferences[ref] = provider
		}
	}

	if mm.ProviderRequirements != nil {
		newMm.ProviderRequirements = make(tfmod.ProviderRequirements, len(mm.ProviderRequirements))
		for provider, vc := range mm.ProviderRequirements {
			// version.Constraints is never mutated in this context
			newMm.ProviderRequirements[provider] = vc
		}
	}

	if mm.Variables != nil {
		newMm.Variables = make(map[string]tfmod.Variable, len(mm.Variables))
		for name, variable := range mm.Variables {
			newMm.Variables[name] = variable
		}
	}

	if mm.Outputs != nil {
		newMm.Outputs = make(map[string]tfmod.Output, len(mm.Outputs))
		for name, output := range mm.Outputs {
			newMm.Outputs[name] = output
		}
	}

	if mm.ModuleCalls != nil {
		newMm.ModuleCalls = make(map[string]tfmod.DeclaredModuleCall, len(mm.ModuleCalls))
		for name, moduleCall := range mm.ModuleCalls {
			newMm.ModuleCalls[name] = moduleCall.Copy()
		}
	}

	return newMm
}

type Module struct {
	Path string

	ModManifest      *datadir.ModuleManifest
	ModManifestErr   error
	ModManifestState op.OpState

	TerraformVersion      *version.Version
	TerraformVersionErr   error
	TerraformVersionState op.OpState

	InstalledProviders      InstalledProviders
	InstalledProvidersErr   error
	InstalledProvidersState op.OpState

	ProviderSchemaErr   error
	ProviderSchemaState op.OpState

	PreloadEmbeddedSchemaState op.OpState

	RefTargets      reference.Targets
	RefTargetsErr   error
	RefTargetsState op.OpState

	RefOrigins      reference.Origins
	RefOriginsErr   error
	RefOriginsState op.OpState

	ParsedModuleFiles ast.ModFiles
	ModuleParsingErr  error

	Meta      ModuleMetadata
	MetaErr   error
	MetaState op.OpState

	ModuleDiagnostics      ast.SourceModDiags
	ModuleDiagnosticsState ast.DiagnosticSourceState
}


func (m *Module) Copy() *Module {
	if m == nil {
		return nil
	}
	newMod := &Module{
		Path: m.Path,

		ModManifest:      m.ModManifest.Copy(),
		ModManifestErr:   m.ModManifestErr,
		ModManifestState: m.ModManifestState,

		// version.Version is practically immutable once parsed
		TerraformVersion:      m.TerraformVersion,
		TerraformVersionErr:   m.TerraformVersionErr,
		TerraformVersionState: m.TerraformVersionState,

		ProviderSchemaErr:   m.ProviderSchemaErr,
		ProviderSchemaState: m.ProviderSchemaState,

		PreloadEmbeddedSchemaState: m.PreloadEmbeddedSchemaState,

		InstalledProvidersErr:   m.InstalledProvidersErr,
		InstalledProvidersState: m.InstalledProvidersState,

		RefTargets:      m.RefTargets.Copy(),
		RefTargetsErr:   m.RefTargetsErr,
		RefTargetsState: m.RefTargetsState,

		RefOrigins:      m.RefOrigins.Copy(),
		RefOriginsErr:   m.RefOriginsErr,
		RefOriginsState: m.RefOriginsState,

		ModuleParsingErr: m.ModuleParsingErr,

		Meta:      m.Meta.Copy(),
		MetaErr:   m.MetaErr,
		MetaState: m.MetaState,

		ModuleDiagnosticsState: m.ModuleDiagnosticsState.Copy(),
	}

	if m.InstalledProviders != nil {
		newMod.InstalledProviders = make(InstalledProviders, 0)
		for addr, pv := range m.InstalledProviders {
			// version.Version is practically immutable once parsed
			newMod.InstalledProviders[addr] = pv
		}
	}

	if m.ParsedModuleFiles != nil {
		newMod.ParsedModuleFiles = make(ast.ModFiles, len(m.ParsedModuleFiles))
		for name, f := range m.ParsedModuleFiles {
			// hcl.File is practically immutable once it comes out of parser
			newMod.ParsedModuleFiles[name] = f
		}
	}

	if m.ModuleDiagnostics != nil {
		newMod.ModuleDiagnostics = make(ast.SourceModDiags, len(m.ModuleDiagnostics))

		for source, modDiags := range m.ModuleDiagnostics {
			newMod.ModuleDiagnostics[source] = make(ast.ModDiags, len(modDiags))

			for name, diags := range modDiags {
				newMod.ModuleDiagnostics[source][name] = make(hcl.Diagnostics, len(diags))
				copy(newMod.ModuleDiagnostics[source][name], diags)
			}
		}
	}

	return newMod
}

func newModule(modPath string) *Module {
	return &Module{
		Path:                       modPath,
		ModManifestState:           op.OpStateUnknown,
		TerraformVersionState:      op.OpStateUnknown,
		ProviderSchemaState:        op.OpStateUnknown,
		PreloadEmbeddedSchemaState: op.OpStateUnknown,
		InstalledProvidersState:    op.OpStateUnknown,
		RefTargetsState:            op.OpStateUnknown,
		MetaState:                  op.OpStateUnknown,
		ModuleDiagnosticsState: ast.DiagnosticSourceState{
			ast.HCLParsingSource:          op.OpStateUnknown,
			ast.SchemaValidationSource:    op.OpStateUnknown,
			ast.ReferenceValidationSource: op.OpStateUnknown,
			ast.TerraformValidateSource:   op.OpStateUnknown,
		},
	}
}

func (s *ModuleStore) Add(modPath string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	err := s.add(txn, modPath)
	if err != nil {
		return err
	}
	txn.Commit()

	return nil
}

func (s *ModuleStore) add(txn *memdb.Txn, modPath string) error {
	// TODO: Introduce Exists method to Txn?
	obj, err := txn.First(s.tableName, "id", modPath)
	if err != nil {
		return err
	}
	if obj != nil {
		return &AlreadyExistsError{
			Idx: modPath,
		}
	}

	mod := newModule(modPath)
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	err = s.queueModuleChange(txn, nil, mod)
	if err != nil {
		return err
	}

	return nil
}

func (s *ModuleStore) Remove(modPath string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	oldObj, err := txn.First(s.tableName, "id", modPath)
	if err != nil {
		return err
	}

	if oldObj == nil {
		// already removed
		return nil
	}

	oldMod := oldObj.(*Module)
	err = s.queueModuleChange(txn, oldMod, nil)
	if err != nil {
		return err
	}

	_, err = txn.DeleteAll(s.tableName, "id", modPath)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) CallersOfModule(modPath string) ([]*Module, error) {
	txn := s.db.Txn(false)
	it, err := txn.Get(s.tableName, "id")
	if err != nil {
		return nil, err
	}

	callers := make([]*Module, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		mod := item.(*Module)

		if mod.ModManifest == nil {
			continue
		}
		if mod.ModManifest.ContainsLocalModule(modPath) {
			callers = append(callers, mod)
		}
	}

	return callers, nil
}

func (s *ModuleStore) ModuleByPath(path string) (*Module, error) {
	txn := s.db.Txn(false)

	mod, err := moduleByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return mod, nil
}

func (s *ModuleStore) AddIfNotExists(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	_, err := moduleByPath(txn, path)
	if err != nil {
		if IsModuleNotFound(err) {
			err := s.add(txn, path)
			if err != nil {
				return err
			}
			txn.Commit()
			return nil
		}

		return err
	}

	return nil
}

func (s *ModuleStore) ModuleCalls(modPath string) (tfmod.ModuleCalls, error) {
	mod, err := s.ModuleByPath(modPath)
	if err != nil {
		return tfmod.ModuleCalls{}, err
	}

	modCalls := tfmod.ModuleCalls{
		Installed: make(map[string]tfmod.InstalledModuleCall),
		Declared:  make(map[string]tfmod.DeclaredModuleCall),
	}

	if mod.ModManifest != nil {
		for _, record := range mod.ModManifest.Records {
			if record.IsRoot() {
				continue
			}
			modCalls.Installed[record.Key] = tfmod.InstalledModuleCall{
				LocalName:  record.Key,
				SourceAddr: record.SourceAddr,
				Version:    record.Version,
				Path:       filepath.Join(modPath, record.Dir),
			}
		}
	}

	for _, mc := range mod.Meta.ModuleCalls {
		modCalls.Declared[mc.LocalName] = tfmod.DeclaredModuleCall{
			LocalName:  mc.LocalName,
			SourceAddr: mc.SourceAddr,
			Version:    mc.Version,
			InputNames: mc.InputNames,
			RangePtr:   mc.RangePtr,
		}
	}

	return modCalls, err
}

func (s *ModuleStore) ProviderRequirementsForModule(modPath string) (tfmod.ProviderRequirements, error) {
	return s.providerRequirementsForModule(modPath, 0)
}

func (s *ModuleStore) providerRequirementsForModule(modPath string, level int) (tfmod.ProviderRequirements, error) {
	// This is just a naive way of checking for cycles, so we don't end up
	// crashing due to stack overflow.
	//
	// Cycles are however unlikely - at least for installed modules, since
	// Terraform would return error when attempting to install modules
	// with cycles.
	if level > s.MaxModuleNesting {
		return nil, fmt.Errorf("%s: too deep module nesting (%d)", modPath, s.MaxModuleNesting)
	}
	mod, err := s.ModuleByPath(modPath)
	if err != nil {
		return nil, err
	}

	level++

	requirements := make(tfmod.ProviderRequirements, 0)
	for k, v := range mod.Meta.ProviderRequirements {
		requirements[k] = v
	}

	for _, mc := range mod.Meta.ModuleCalls {
		localAddr, ok := mc.SourceAddr.(tfmod.LocalSourceAddr)
		if !ok {
			continue
		}

		fullPath := filepath.Join(modPath, localAddr.String())

		pr, err := s.providerRequirementsForModule(fullPath, level)
		if err != nil {
			return requirements, err
		}
		for pAddr, pCons := range pr {
			if cons, ok := requirements[pAddr]; ok {
				for _, c := range pCons {
					if !constraintContains(cons, c) {
						requirements[pAddr] = append(requirements[pAddr], c)
					}
				}
			}
			requirements[pAddr] = pCons
		}
	}

	if mod.ModManifest != nil {
		for _, record := range mod.ModManifest.Records {
			_, ok := record.SourceAddr.(tfmod.LocalSourceAddr)
			if ok {
				continue
			}

			if record.IsRoot() {
				continue
			}

			fullPath := filepath.Join(modPath, record.Dir)
			pr, err := s.providerRequirementsForModule(fullPath, level)
			if err != nil {
				continue
			}
			for pAddr, pCons := range pr {
				if cons, ok := requirements[pAddr]; ok {
					for _, c := range pCons {
						if !constraintContains(cons, c) {
							requirements[pAddr] = append(requirements[pAddr], c)
						}
					}
				}
				requirements[pAddr] = pCons
			}
		}
	}

	return requirements, nil
}

func constraintContains(vCons version.Constraints, cons *version.Constraint) bool {
	for _, c := range vCons {
		if c == cons {
			return true
		}
	}
	return false
}

func (s *ModuleStore) LocalModuleMeta(modPath string) (*tfmod.Meta, error) {
	mod, err := s.ModuleByPath(modPath)
	if err != nil {
		return nil, err
	}
	if mod.MetaState != op.OpStateLoaded {
		return nil, fmt.Errorf("%s: module data not available", modPath)
	}
	return &tfmod.Meta{
		Path:                 mod.Path,
		ProviderReferences:   mod.Meta.ProviderReferences,
		ProviderRequirements: mod.Meta.ProviderRequirements,
		CoreRequirements:     mod.Meta.CoreRequirements,
		Variables:            mod.Meta.Variables,
		Outputs:              mod.Meta.Outputs,
		Filenames:            mod.Meta.Filenames,
		ModuleCalls:          mod.Meta.ModuleCalls,
	}, nil
}

func (s *ModuleStore) RegistryModuleMeta(addr tfaddr.Module, cons version.Constraints) (*registry.ModuleData, error) {
	txn := s.db.Txn(false)

	it, err := txn.Get(registryModuleTableName, "source_addr", addr)
	if err != nil {
		return nil, err
	}

	for item := it.Next(); item != nil; item = it.Next() {
		mod := item.(*RegistryModuleData)

		if mod.Error {
			continue
		}

		if cons.Check(mod.Version) {
			return &registry.ModuleData{
				Version: mod.Version,
				Inputs:  mod.Inputs,
				Outputs: mod.Outputs,
			}, nil
		}
	}

	return nil, &ModuleNotFoundError{
		Source: addr.String(),
	}
}

func moduleByPath(txn *memdb.Txn, path string) (*Module, error) {
	obj, err := txn.First(moduleTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &ModuleNotFoundError{
			Source: path,
		}
	}
	return obj.(*Module), nil
}

func moduleCopyByPath(txn *memdb.Txn, path string) (*Module, error) {
	mod, err := moduleByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return mod.Copy(), nil
}

func (s *ModuleStore) UpdateInstalledProviders(path string, pvs map[tfaddr.Provider]*version.Version, pvErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetInstalledProvidersState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldMod, err := moduleByPath(txn, path)
	if err != nil {
		return err
	}

	mod := oldMod.Copy()
	mod.InstalledProviders = pvs
	mod.InstalledProvidersErr = pvErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	err = s.queueModuleChange(txn, oldMod, mod)
	if err != nil {
		return err
	}

	err = updateProviderVersions(txn, path, pvs)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) SetInstalledProvidersState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.InstalledProvidersState = state

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) List() ([]*Module, error) {
	txn := s.db.Txn(false)

	it, err := txn.Get(s.tableName, "id")
	if err != nil {
		return nil, err
	}

	modules := make([]*Module, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		mod := item.(*Module)
		modules = append(modules, mod)
	}

	return modules, nil
}

func (s *ModuleStore) SetModManifestState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ModManifestState = state

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateModManifest(path string, manifest *datadir.ModuleManifest, mErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetModManifestState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ModManifest = manifest
	mod.ModManifestErr = mErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	err = s.queueModuleChange(txn, nil, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) SetTerraformVersionState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.TerraformVersionState = state
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	err = s.queueModuleChange(txn, nil, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) SetProviderSchemaState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ProviderSchemaState = state
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) SetPreloadEmbeddedSchemaState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.PreloadEmbeddedSchemaState = state
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) FinishProviderSchemaLoading(path string, psErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetProviderSchemaState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldMod, err := moduleByPath(txn, path)
	if err != nil {
		return err
	}

	mod := oldMod.Copy()
	mod.ProviderSchemaErr = psErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	err = s.queueModuleChange(txn, oldMod, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateTerraformAndProviderVersions(modPath string, tfVer *version.Version, pv map[tfaddr.Provider]*version.Version, vErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetTerraformVersionState(modPath, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldMod, err := moduleByPath(txn, modPath)
	if err != nil {
		return err
	}

	mod := oldMod.Copy()
	mod.TerraformVersion = tfVer
	mod.TerraformVersionErr = vErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	err = s.queueModuleChange(txn, oldMod, mod)
	if err != nil {
		return err
	}

	err = updateProviderVersions(txn, modPath, pv)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateParsedModuleFiles(path string, pFiles ast.ModFiles, pErr error) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ParsedModuleFiles = pFiles

	mod.ModuleParsingErr = pErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) SetMetaState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.MetaState = state
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateMetadata(path string, meta *tfmod.Meta, mErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetMetaState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldMod, err := moduleByPath(txn, path)
	if err != nil {
		return err
	}

	mod := oldMod.Copy()
	mod.Meta = ModuleMetadata{
		CoreRequirements:     meta.CoreRequirements,
		Cloud:                meta.Cloud,
		Backend:              meta.Backend,
		ProviderReferences:   meta.ProviderReferences,
		ProviderRequirements: meta.ProviderRequirements,
		Variables:            meta.Variables,
		Outputs:              meta.Outputs,
		Filenames:            meta.Filenames,
		ModuleCalls:          meta.ModuleCalls,
	}
	mod.MetaErr = mErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	err = s.queueModuleChange(txn, oldMod, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateModuleDiagnostics(path string, source ast.DiagnosticSource, diags ast.ModDiags) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetModuleDiagnosticsState(path, source, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldMod, err := moduleByPath(txn, path)
	if err != nil {
		return err
	}

	mod := oldMod.Copy()
	if mod.ModuleDiagnostics == nil {
		mod.ModuleDiagnostics = make(ast.SourceModDiags)
	}
	mod.ModuleDiagnostics[source] = diags

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	err = s.queueModuleChange(txn, oldMod, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) SetModuleDiagnosticsState(path string, source ast.DiagnosticSource, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}
	mod.ModuleDiagnosticsState[source] = state

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) SetReferenceTargetsState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.RefTargetsState = state
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateReferenceTargets(path string, refs reference.Targets, rErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetReferenceTargetsState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.RefTargets = refs
	mod.RefTargetsErr = rErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) SetReferenceOriginsState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.RefOriginsState = state
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateReferenceOrigins(path string, origins reference.Origins, roErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetReferenceOriginsState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.RefOrigins = origins
	mod.RefOriginsErr = roErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}
