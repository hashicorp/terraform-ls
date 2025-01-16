// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/modules/ast"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	"github.com/hashicorp/terraform-schema/registry"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

type ModuleStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger

	// MaxModuleNesting represents how many nesting levels we'd attempt
	// to parse provider requirements before returning error.
	MaxModuleNesting int

	providerSchemasStore *globalState.ProviderSchemaStore
	registryModuleStore  *globalState.RegistryModuleStore
	changeStore          *globalState.ChangeStore
}

func (s *ModuleStore) SetLogger(logger *log.Logger) {
	s.logger = logger
}

func moduleByPath(txn *memdb.Txn, path string) (*ModuleRecord, error) {
	obj, err := txn.First(moduleTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &globalState.RecordNotFoundError{
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
		return &globalState.AlreadyExistsError{
			Idx: modPath,
		}
	}

	mod := newModule(modPath)
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	err = s.queueModuleChange(nil, mod)
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
	err = s.queueModuleChange(oldMod, nil)
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

func (s *ModuleStore) ModuleRecordByPath(path string) (*ModuleRecord, error) {
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
		if globalState.IsRecordNotFound(err) {
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
	mod, err := s.ModuleRecordByPath(modPath)
	if err != nil {
		return map[string]tfmod.DeclaredModuleCall{}, err
	}

	declared := make(map[string]tfmod.DeclaredModuleCall)
	for _, mc := range mod.Meta.ModuleCalls {
		declared[mc.LocalName] = tfmod.DeclaredModuleCall{
			LocalName:     mc.LocalName,
			RawSourceAddr: mc.RawSourceAddr,
			SourceAddr:    mc.SourceAddr,
			Version:       mc.Version,
			InputNames:    mc.InputNames,
			RangePtr:      mc.RangePtr,
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
	mod, err := s.ModuleRecordByPath(modPath)
	if err != nil {
		// It's possible that the configuration contains a module with an
		// invalid local source, so we just ignore it if it can't be found.
		// This allows us to still return provider requirements for other modules
		return tfmod.ProviderRequirements{}, nil
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
	mod, err := s.ModuleRecordByPath(modPath)
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

	err = s.queueModuleChange(oldMod, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateModuleDiagnostics(path string, source globalAst.DiagnosticSource, diags ast.ModDiags) error {
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

	err = s.queueModuleChange(oldMod, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) SetModuleDiagnosticsState(path string, source globalAst.DiagnosticSource, state op.OpState) error {
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

func (s *ModuleStore) SetWriteOnlyAttributesState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.WriteOnlyAttributesState = state
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateWriteOnlyAttributes(path string, woAttrs WriteOnlyAttributes, woAttrsErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetWriteOnlyAttributesState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.WriteOnlyAttributes = woAttrs
	mod.WriteOnlyAttributesErr = woAttrsErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) RegistryModuleMeta(addr tfaddr.Module, cons version.Constraints) (*registry.ModuleData, error) {
	return s.registryModuleStore.RegistryModuleMeta(addr, cons)
}

func (s *ModuleStore) ProviderSchema(modPath string, addr tfaddr.Provider, vc version.Constraints) (*tfschema.ProviderSchema, error) {
	return s.providerSchemasStore.ProviderSchema(modPath, addr, vc)
}

func (s *ModuleStore) queueModuleChange(oldMod, newMod *ModuleRecord) error {
	changes := globalState.Changes{}

	switch {
	// new module added
	case oldMod == nil && newMod != nil:
		if len(newMod.Meta.CoreRequirements) > 0 {
			changes.CoreRequirements = true
		}
		if newMod.Meta.Cloud != nil {
			changes.Cloud = true
		}
		if newMod.Meta.Backend != nil {
			changes.Backend = true
		}
		if len(newMod.Meta.ProviderRequirements) > 0 {
			changes.ProviderRequirements = true
		}
	// module removed
	case oldMod != nil && newMod == nil:
		changes.IsRemoval = true

		if len(oldMod.Meta.CoreRequirements) > 0 {
			changes.CoreRequirements = true
		}
		if oldMod.Meta.Cloud != nil {
			changes.Cloud = true
		}
		if oldMod.Meta.Backend != nil {
			changes.Backend = true
		}
		if len(oldMod.Meta.ProviderRequirements) > 0 {
			changes.ProviderRequirements = true
		}
	// module changed
	default:
		if !oldMod.Meta.CoreRequirements.Equals(newMod.Meta.CoreRequirements) {
			changes.CoreRequirements = true
		}
		if !oldMod.Meta.Backend.Equals(newMod.Meta.Backend) {
			changes.Backend = true
		}
		if !oldMod.Meta.Cloud.Equals(newMod.Meta.Cloud) {
			changes.Cloud = true
		}
		if !oldMod.Meta.ProviderRequirements.Equals(newMod.Meta.ProviderRequirements) {
			changes.ProviderRequirements = true
		}
	}

	oldDiags, newDiags := 0, 0
	if oldMod != nil {
		oldDiags = oldMod.ModuleDiagnostics.Count()
	}
	if newMod != nil {
		newDiags = newMod.ModuleDiagnostics.Count()
	}
	// Comparing diagnostics accurately could be expensive
	// so we just treat any non-empty diags as a change
	if oldDiags > 0 || newDiags > 0 {
		changes.Diagnostics = true
	}

	oldOrigins, oldTargets := 0, 0
	if oldMod != nil {
		oldOrigins = len(oldMod.RefOrigins)
		oldTargets = len(oldMod.RefTargets)
	}
	newOrigins, newTargets := 0, 0
	if newMod != nil {
		newOrigins = len(newMod.RefOrigins)
		newTargets = len(newMod.RefTargets)
	}
	if oldOrigins != newOrigins {
		changes.ReferenceOrigins = true
	}
	if oldTargets != newTargets {
		changes.ReferenceTargets = true
	}

	var modHandle document.DirHandle
	if oldMod != nil {
		modHandle = document.DirHandleFromPath(oldMod.Path())
	} else {
		modHandle = document.DirHandleFromPath(newMod.Path())
	}

	return s.changeStore.QueueChange(modHandle, changes)
}

func (f *ModuleStore) MetadataReady(dir document.DirHandle) (<-chan struct{}, bool, error) {
	rTxn := f.db.Txn(false)

	wCh, recordObj, err := rTxn.FirstWatch(f.tableName, "module_state", dir.Path(), op.OpStateLoaded)
	if err != nil {
		return nil, false, err
	}
	if recordObj != nil {
		return wCh, true, nil
	}

	return wCh, false, nil
}
