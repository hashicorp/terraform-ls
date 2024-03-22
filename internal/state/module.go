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
	tfmod "github.com/hashicorp/terraform-schema/module"

	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// ModuleRecord contains all information about module files
// we have for a certain path
type ModuleRecord struct {
	path string

	// ProviderSchemaState tracks if we tried loading all provider schemas
	// that this module is using via Terraform CLI
	ProviderSchemaState op.OpState
	ProviderSchemaErr   error

	// PreloadEmbeddedSchemaState tracks if we tried loading all provider
	// schemas from our embedded schema data
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

func (m *ModuleRecord) Copy() *ModuleRecord {
	if m == nil {
		return nil
	}
	newMod := &ModuleRecord{
		path: m.path,

		ProviderSchemaErr:   m.ProviderSchemaErr,
		ProviderSchemaState: m.ProviderSchemaState,

		PreloadEmbeddedSchemaState: m.PreloadEmbeddedSchemaState,

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

func (m *ModuleRecord) Path() string {
	return m.path
}

func newModule(modPath string) *ModuleRecord {
	return &ModuleRecord{
		path:                       modPath,
		ProviderSchemaState:        op.OpStateUnknown,
		PreloadEmbeddedSchemaState: op.OpStateUnknown,
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

// NewModuleTest is a test helper to create a new Module object
func NewModuleTest(path string) *ModuleRecord {
	return &ModuleRecord{
		path: path,
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

	oldMod := oldObj.(*ModuleRecord)
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

func (s *ModuleStore) ModuleByPath(path string) (*ModuleRecord, error) {
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
		if IsRecordNotFound(err) {
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

func (s *ModuleStore) DeclaredModuleCalls(modPath string) (map[string]tfmod.DeclaredModuleCall, error) {
	mod, err := s.ModuleByPath(modPath)
	if err != nil {
		return map[string]tfmod.DeclaredModuleCall{}, err
	}

	declared := make(map[string]tfmod.DeclaredModuleCall)
	for _, mc := range mod.Meta.ModuleCalls {
		declared[mc.LocalName] = tfmod.DeclaredModuleCall{
			LocalName:  mc.LocalName,
			SourceAddr: mc.SourceAddr,
			Version:    mc.Version,
			InputNames: mc.InputNames,
			RangePtr:   mc.RangePtr,
		}
	}

	return declared, err
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

	// TODO! move into RootStore
	// if mod.ModManifest != nil {
	// 	for _, record := range mod.ModManifest.Records {
	// 		_, ok := record.SourceAddr.(tfmod.LocalSourceAddr)
	// 		if ok {
	// 			continue
	// 		}

	// 		if record.IsRoot() {
	// 			continue
	// 		}

	// 		fullPath := filepath.Join(modPath, record.Dir)
	// 		pr, err := s.providerRequirementsForModule(fullPath, level)
	// 		if err != nil {
	// 			continue
	// 		}
	// 		for pAddr, pCons := range pr {
	// 			if cons, ok := requirements[pAddr]; ok {
	// 				for _, c := range pCons {
	// 					if !constraintContains(cons, c) {
	// 						requirements[pAddr] = append(requirements[pAddr], c)
	// 					}
	// 				}
	// 			}
	// 			requirements[pAddr] = pCons
	// 		}
	// 	}
	// }

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
		Path:      mod.path,
		Filenames: mod.Meta.Filenames,

		CoreRequirements:     mod.Meta.CoreRequirements,
		Backend:              mod.Meta.Backend,
		Cloud:                mod.Meta.Cloud,
		ProviderReferences:   mod.Meta.ProviderReferences,
		ProviderRequirements: mod.Meta.ProviderRequirements,
		Variables:            mod.Meta.Variables,
		Outputs:              mod.Meta.Outputs,
		ModuleCalls:          mod.Meta.ModuleCalls,
	}, nil
}

func moduleByPath(txn *memdb.Txn, path string) (*ModuleRecord, error) {
	obj, err := txn.First(moduleTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &RecordNotFoundError{
			Source: path,
		}
	}
	return obj.(*ModuleRecord), nil
}

func moduleCopyByPath(txn *memdb.Txn, path string) (*ModuleRecord, error) {
	mod, err := moduleByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return mod.Copy(), nil
}

func (s *ModuleStore) List() ([]*ModuleRecord, error) {
	txn := s.db.Txn(false)

	it, err := txn.Get(s.tableName, "id")
	if err != nil {
		return nil, err
	}

	modules := make([]*ModuleRecord, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		mod := item.(*ModuleRecord)
		modules = append(modules, mod)
	}

	return modules, nil
}

func (s *ModuleStore) Exists(path string) bool {
	txn := s.db.Txn(false)

	obj, err := txn.First(s.tableName, "id", path)
	if err != nil {
		return false
	}

	return obj != nil
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
