package state

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"

	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

type ModuleMetadata struct {
	CoreRequirements     version.Constraints
	ProviderReferences   map[tfmod.ProviderRef]tfaddr.Provider
	ProviderRequirements map[tfaddr.Provider]version.Constraints
	Variables            map[string]tfmod.Variable
}

func (mm ModuleMetadata) Copy() ModuleMetadata {
	newMm := ModuleMetadata{
		// version.Constraints is practically immutable once parsed
		CoreRequirements: mm.CoreRequirements,
	}

	if mm.ProviderReferences != nil {
		newMm.ProviderReferences = make(map[tfmod.ProviderRef]tfaddr.Provider, len(mm.ProviderReferences))
		for ref, provider := range mm.ProviderReferences {
			newMm.ProviderReferences[ref] = provider
		}
	}

	if mm.ProviderRequirements != nil {
		newMm.ProviderRequirements = make(map[tfaddr.Provider]version.Constraints, len(mm.ProviderRequirements))
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

	ProviderSchemaErr   error
	ProviderSchemaState op.OpState

	RefTargets      lang.ReferenceTargets
	RefTargetsErr   error
	RefTargetsState op.OpState

	ParsedModuleFiles  map[string]*hcl.File
	ParsedVarsFiles    map[string]*hcl.File
	ModuleParsingErr   error
	VarsParsingErr     error
	ModuleParsingState op.OpState
	VarsParsingState   op.OpState

	Meta      ModuleMetadata
	MetaErr   error
	MetaState op.OpState

	ModuleDiagnostics map[string]hcl.Diagnostics
	VarsDiagnostics   map[string]hcl.Diagnostics
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

		ModuleParsingErr:   m.ModuleParsingErr,
		VarsParsingErr:     m.VarsParsingErr,
		ModuleParsingState: m.ModuleParsingState,
		VarsParsingState:   m.VarsParsingState,

		Meta:      m.Meta.Copy(),
		MetaErr:   m.MetaErr,
		MetaState: m.MetaState,
	}

	if m.ParsedModuleFiles != nil {
		newMod.ParsedModuleFiles = make(map[string]*hcl.File, len(m.ParsedModuleFiles))
		for name, f := range m.ParsedModuleFiles {
			// hcl.File is practically immutable once it comes out of parser
			newMod.ParsedModuleFiles[name] = f
		}
	}

	if m.ParsedVarsFiles != nil {
		newMod.ParsedVarsFiles = make(map[string]*hcl.File, len(m.ParsedVarsFiles))
		for name, f := range m.ParsedVarsFiles {
			// hcl.File is practically immutable once it comes out of parser
			newMod.ParsedVarsFiles[name] = f
		}
	}

	if m.ModuleDiagnostics != nil {
		newMod.ModuleDiagnostics = make(map[string]hcl.Diagnostics, len(m.ModuleDiagnostics))
		for name, diags := range m.ModuleDiagnostics {
			newMod.ModuleDiagnostics[name] = make(hcl.Diagnostics, len(diags))
			for i, diag := range diags {
				// hcl.Diagnostic is practically immutable once it comes out of parser
				newMod.ModuleDiagnostics[name][i] = diag
			}
		}
	}

	if m.VarsDiagnostics != nil {
		newMod.VarsDiagnostics = make(map[string]hcl.Diagnostics, len(m.VarsDiagnostics))
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

	err = txn.Insert(s.tableName, newModule(modPath))
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) Remove(modPath string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	_, err := txn.DeleteAll(s.tableName, "id", modPath)
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

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ProviderSchemaErr = psErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateTerraformVersion(modPath string, tfVer *version.Version, pv map[tfaddr.Provider]*version.Version, vErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetTerraformVersionState(modPath, op.OpStateLoaded)
	})
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, modPath)
	if err != nil {
		return err
	}

	mod.TerraformVersion = tfVer
	mod.TerraformVersionErr = vErr

	err = txn.Insert(s.tableName, mod)
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

func (s *ModuleStore) UpdateParsedModuleFiles(path string, pFiles map[string]*hcl.File, pErr error) error {
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

func (s *ModuleStore) UpdateParsedVarsFiles(path string, pFiles map[string]*hcl.File, pErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetVarsParsingState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ParsedVarsFiles = pFiles

	mod.VarsParsingErr = pErr

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

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.Meta = ModuleMetadata{
		CoreRequirements:     meta.CoreRequirements,
		ProviderReferences:   meta.ProviderReferences,
		ProviderRequirements: meta.ProviderRequirements,
		Variables:            meta.Variables,
	}
	mod.MetaErr = mErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateModuleDiagnostics(path string, diags map[string]hcl.Diagnostics) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ModuleDiagnostics = diags

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateVarsDiagnostics(path string, diags map[string]hcl.Diagnostics) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.VarsDiagnostics = diags

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

	mod, err := moduleByPath(txn, path)
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

func (s *ModuleStore) UpdateReferenceTargets(path string, refs lang.ReferenceTargets, rErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetReferenceTargetsState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	mod, err := moduleByPath(txn, path)
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
