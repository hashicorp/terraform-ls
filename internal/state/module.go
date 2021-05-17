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

	References      lang.References
	ReferencesErr   error
	ReferencesState op.OpState

	ParsedFiles  map[string]*hcl.File
	ParsingErr   error
	ParsingState op.OpState

	Meta      ModuleMetadata
	MetaErr   error
	MetaState op.OpState

	Diagnostics map[string]hcl.Diagnostics
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

		References:      m.References.Copy(),
		ReferencesErr:   m.ReferencesErr,
		ReferencesState: m.ReferencesState,

		ParsingErr:   m.ParsingErr,
		ParsingState: m.ParsingState,

		Meta:      m.Meta.Copy(),
		MetaErr:   m.MetaErr,
		MetaState: m.MetaState,
	}

	if m.ParsedFiles != nil {
		newMod.ParsedFiles = make(map[string]*hcl.File, len(m.ParsedFiles))
		for name, f := range m.ParsedFiles {
			// hcl.File is practically immutable once it comes out of parser
			newMod.ParsedFiles[name] = f
		}
	}

	if m.Diagnostics != nil {
		newMod.Diagnostics = make(map[string]hcl.Diagnostics, len(m.Diagnostics))
		for name, diags := range m.Diagnostics {
			newMod.Diagnostics[name] = make(hcl.Diagnostics, len(diags))
			for i, diag := range diags {
				// hcl.Diagnostic is practically immutable once it comes out of parser
				newMod.Diagnostics[name][i] = diag
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
		ReferencesState:       op.OpStateUnknown,
		ParsingState:          op.OpStateUnknown,
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

func (s *ModuleStore) SetParsingState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ParsingState = state
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateParsedFiles(path string, pFiles map[string]*hcl.File, pErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetParsingState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ParsedFiles = pFiles
	mod.ParsingErr = pErr

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
	}
	mod.MetaErr = mErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateDiagnostics(path string, diags map[string]hcl.Diagnostics) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.Diagnostics = diags

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) SetReferencesState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := moduleByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ReferencesState = state
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ModuleStore) UpdateReferences(path string, refs lang.References, rErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetReferencesState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	mod, err := moduleByPath(txn, path)
	if err != nil {
		return err
	}

	mod.References = refs
	mod.ReferencesErr = rErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}
