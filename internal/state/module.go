package state

import (
	"path/filepath"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"

	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

type ModuleMetadata struct {
	CoreRequirements     version.Constraints
	Backend              *tfmod.Backend
	ProviderReferences   map[tfmod.ProviderRef]tfaddr.Provider
	ProviderRequirements tfmod.ProviderRequirements
	Variables            map[string]tfmod.Variable
	Outputs              map[string]tfmod.Output
	Filenames            []string
	ModuleCalls          map[string]tfmod.ModuleCall
}

func (mm ModuleMetadata) Copy() ModuleMetadata {
	newMm := ModuleMetadata{
		// version.Constraints is practically immutable once parsed
		CoreRequirements: mm.CoreRequirements,
		Filenames:        mm.Filenames,
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
		newMm.ModuleCalls = make(map[string]tfmod.ModuleCall, len(mm.ModuleCalls))
		for name, moduleCall := range mm.ModuleCalls {
			newMm.ModuleCalls[name] = moduleCall
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

	InstalledProviders InstalledProviders

	ProviderSchemaErr   error
	ProviderSchemaState op.OpState

	RefTargets      reference.Targets
	RefTargetsErr   error
	RefTargetsState op.OpState

	RefOrigins      reference.Origins
	RefOriginsErr   error
	RefOriginsState op.OpState

	VarsRefOrigins      reference.Origins
	VarsRefOriginsErr   error
	VarsRefOriginsState op.OpState

	ParsedModuleFiles  ast.ModFiles
	ParsedVarsFiles    ast.VarsFiles
	ModuleParsingErr   error
	VarsParsingErr     error
	ModuleParsingState op.OpState
	VarsParsingState   op.OpState

	Meta      ModuleMetadata
	MetaErr   error
	MetaState op.OpState

	ModuleDiagnostics ast.ModDiags
	VarsDiagnostics   ast.VarsDiags
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

		RefTargets:      m.RefTargets.Copy(),
		RefTargetsErr:   m.RefTargetsErr,
		RefTargetsState: m.RefTargetsState,

		RefOrigins:      m.RefOrigins.Copy(),
		RefOriginsErr:   m.RefOriginsErr,
		RefOriginsState: m.RefOriginsState,

		VarsRefOrigins:      m.VarsRefOrigins.Copy(),
		VarsRefOriginsErr:   m.VarsRefOriginsErr,
		VarsRefOriginsState: m.VarsRefOriginsState,

		ModuleParsingErr:   m.ModuleParsingErr,
		VarsParsingErr:     m.VarsParsingErr,
		ModuleParsingState: m.ModuleParsingState,
		VarsParsingState:   m.VarsParsingState,

		Meta:      m.Meta.Copy(),
		MetaErr:   m.MetaErr,
		MetaState: m.MetaState,
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

	if m.ParsedVarsFiles != nil {
		newMod.ParsedVarsFiles = make(ast.VarsFiles, len(m.ParsedVarsFiles))
		for name, f := range m.ParsedVarsFiles {
			// hcl.File is practically immutable once it comes out of parser
			newMod.ParsedVarsFiles[name] = f
		}
	}

	if m.ModuleDiagnostics != nil {
		newMod.ModuleDiagnostics = make(ast.ModDiags, len(m.ModuleDiagnostics))
		for name, diags := range m.ModuleDiagnostics {
			newMod.ModuleDiagnostics[name] = make(hcl.Diagnostics, len(diags))
			for i, diag := range diags {
				// hcl.Diagnostic is practically immutable once it comes out of parser
				newMod.ModuleDiagnostics[name][i] = diag
			}
		}
	}

	if m.VarsDiagnostics != nil {
		newMod.VarsDiagnostics = make(ast.VarsDiags, len(m.VarsDiagnostics))
		for name, diags := range m.VarsDiagnostics {
			newMod.VarsDiagnostics[name] = make(hcl.Diagnostics, len(diags))
			for i, diag := range diags {
				// hcl.Diagnostic is practically immutable once it comes out of parser
				newMod.VarsDiagnostics[name][i] = diag
			}
		}
	}

	return newMod
}

func newModule(modPath string) *Module {
	return &Module{
		Path:                  modPath,
		ModManifestState:      op.OpStateUnknown,
		TerraformVersionState: op.OpStateUnknown,
		ProviderSchemaState:   op.OpStateUnknown,
		RefTargetsState:       op.OpStateUnknown,
		ModuleParsingState:    op.OpStateUnknown,
		MetaState:             op.OpStateUnknown,
	}
}

func (s *ModuleStore) Add(modPath string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

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

	txn.Defer(func() {
		go s.ChangeHooks.notifyModuleChange(nil, mod)
	})

	txn.Commit()
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

	txn.Defer(func() {
		oldMod := oldObj.(*Module)
		go s.ChangeHooks.notifyModuleChange(oldMod, nil)
	})

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

func (s *ModuleStore) ModuleCalls(modPath string) ([]tfmod.ModuleCall, error) {
	result := make([]tfmod.ModuleCall, 0)
	modList, err := s.List()
	for _, mod := range modList {
		// Try to generate moduleCalls from manifest first
		// This will only work if the module has been installed
		// With a local installation it's possible to resolve the `Path` field
		if mod.ModManifest != nil {
			for _, record := range mod.ModManifest.Records {
				if record.IsRoot() {
					continue
				}
				result = append(result, tfmod.ModuleCall{
					LocalName:  record.Key,
					SourceAddr: record.SourceAddr,
					Version:    record.VersionStr,
					Path:       filepath.Join(modPath, record.Dir),
				})
			}
		}
	}
	// If there are no installed modules, we can source a list of module calls
	// from earlydecoder, but miss the `Path`
	if len(result) == 0 {
		for _, mod := range modList {
			for _, moduleCall := range mod.Meta.ModuleCalls {
				result = append(result, tfmod.ModuleCall{
					LocalName:  moduleCall.LocalName,
					SourceAddr: moduleCall.SourceAddr,
					Version:    moduleCall.Version,
				})
			}
		}
	}
	return result, err
}

func (s *ModuleStore) ModuleMeta(modPath string) (*tfmod.Meta, error) {
	mod, err := s.ModuleByPath(modPath)
	if err != nil {
		return nil, err
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

func moduleByPath(txn *memdb.Txn, path string) (*Module, error) {
	obj, err := txn.First(moduleTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &ModuleNotFoundError{
			Path: path,
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

func (s *ModuleStore) UpdateInstalledProviders(path string, pvs map[tfaddr.Provider]*version.Version) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	oldMod, err := moduleByPath(txn, path)
	if err != nil {
		return err
	}

	mod := oldMod.Copy()

	// Providers may come from different sources (schema or version command)
	// and we don't get their versions in both cases, so we make sure the existing
	// versions are retained to get the most of both sources.
	newProviders := make(map[tfaddr.Provider]*version.Version, 0)
	for addr, pv := range pvs {
		if pv == nil {
			if v, ok := oldMod.InstalledProviders[addr]; ok && v != nil {
				pv = v
			}
		}
		newProviders[addr] = pv
	}
	mod.InstalledProviders = newProviders

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Defer(func() {
		go s.ChangeHooks.notifyModuleChange(oldMod, mod)
	})

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

	txn.Defer(func() {
		s.logger.Printf("Queuing refresh for %s", path)
		go s.ChangeHooks.notifyModuleChange(nil, mod)
	})

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

	txn.Defer(func() {
		go s.ChangeHooks.notifyModuleChange(nil, mod)
	})

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

	txn.Defer(func() {
		go s.ChangeHooks.notifyModuleChange(oldMod, mod)
	})

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateTerraformVersion(modPath string, tfVer *version.Version, pv map[tfaddr.Provider]*version.Version, vErr error) error {
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

	txn.Defer(func() {
		go s.ChangeHooks.notifyModuleChange(oldMod, mod)
	})

	err = updateProviderVersions(txn, modPath, pv)
	if err != nil {
		return err
	}

	txn.Defer(func() {
		go s.ChangeHooks.notifyModuleChange(nil, mod)
	})

	txn.Commit()
	return nil
}

func (s *ModuleStore) SetModuleParsingState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ModuleParsingState = state
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) SetVarsParsingState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.VarsParsingState = state
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateParsedModuleFiles(path string, pFiles ast.ModFiles, pErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetModuleParsingState(path, op.OpStateLoaded)
	})
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

func (s *ModuleStore) UpdateParsedVarsFiles(path string, vFiles ast.VarsFiles, vErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetVarsParsingState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ParsedVarsFiles = vFiles

	mod.VarsParsingErr = vErr

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

	txn.Defer(func() {
		go s.ChangeHooks.notifyModuleChange(oldMod, mod)
	})

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateModuleDiagnostics(path string, diags ast.ModDiags) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	oldMod, err := moduleByPath(txn, path)
	if err != nil {
		return err
	}

	mod := oldMod.Copy()
	mod.ModuleDiagnostics = diags

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Defer(func() {
		go s.ChangeHooks.notifyModuleChange(oldMod, mod)
	})

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateVarsDiagnostics(path string, diags ast.VarsDiags) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	oldMod, err := moduleByPath(txn, path)
	if err != nil {
		return err
	}

	mod := oldMod.Copy()
	mod.VarsDiagnostics = diags

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Defer(func() {
		go s.ChangeHooks.notifyModuleChange(oldMod, mod)
	})

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

func (s *ModuleStore) SetVarsReferenceOriginsState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.VarsRefOriginsState = state
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateVarsReferenceOrigins(path string, origins reference.Origins, roErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetVarsReferenceOriginsState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.VarsRefOrigins = origins
	mod.VarsRefOriginsErr = roErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}
