package state

import (
	"path/filepath"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

// RootRecord contains all information about a module root path, like
// anything related to .terraform/ or .terraform.lock.hcl.
type RootRecord struct {
	path string

	ModManifest      *datadir.ModuleManifest
	ModManifestErr   error
	ModManifestState op.OpState

	TerraformVersion      *version.Version
	TerraformVersionErr   error
	TerraformVersionState op.OpState

	InstalledProviders      InstalledProviders
	InstalledProvidersErr   error
	InstalledProvidersState op.OpState
}

func (m *RootRecord) Copy() *RootRecord {
	if m == nil {
		return nil
	}
	newMod := &RootRecord{
		path: m.path,

		ModManifest:      m.ModManifest.Copy(),
		ModManifestErr:   m.ModManifestErr,
		ModManifestState: m.ModManifestState,

		// version.Version is practically immutable once parsed
		TerraformVersion:      m.TerraformVersion,
		TerraformVersionErr:   m.TerraformVersionErr,
		TerraformVersionState: m.TerraformVersionState,

		InstalledProvidersErr:   m.InstalledProvidersErr,
		InstalledProvidersState: m.InstalledProvidersState,
	}

	if m.InstalledProviders != nil {
		newMod.InstalledProviders = make(InstalledProviders, 0)
		for addr, pv := range m.InstalledProviders {
			// version.Version is practically immutable once parsed
			newMod.InstalledProviders[addr] = pv
		}
	}

	return newMod
}

func (m *RootRecord) Path() string {
	return m.path
}

func newRootRecord(modPath string) *RootRecord {
	return &RootRecord{
		path:                    modPath,
		ModManifestState:        op.OpStateUnknown,
		TerraformVersionState:   op.OpStateUnknown,
		InstalledProvidersState: op.OpStateUnknown,
	}
}

// NewRootRecordTest is a test helper to create a new Module object
func NewRootRecordTest(path string) *RootRecord {
	return &RootRecord{
		path: path,
	}
}

func (s *RootStore) RootRecordByPath(path string) (*RootRecord, error) {
	txn := s.db.Txn(false)

	mod, err := rootRecordByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return mod, nil
}

func rootRecordByPath(txn *memdb.Txn, path string) (*RootRecord, error) {
	obj, err := txn.First(rootTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &RecordNotFoundError{
			Source: path,
		}
	}
	return obj.(*RootRecord), nil
}

func rootRecordCopyByPath(txn *memdb.Txn, path string) (*RootRecord, error) {
	record, err := rootRecordByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return record.Copy(), nil
}

func (s *RootStore) UpdateInstalledProviders(path string, pvs map[tfaddr.Provider]*version.Version, pvErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetInstalledProvidersState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldMod, err := rootRecordByPath(txn, path)
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

	// TODO! queue module change
	// err = s.queueModuleChange(txn, oldMod, mod)
	// if err != nil {
	// 	return err
	// }

	err = updateProviderVersions(txn, path, pvs)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *RootStore) SetInstalledProvidersState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := rootRecordCopyByPath(txn, path)
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

func (s *RootStore) SetModManifestState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := rootRecordCopyByPath(txn, path)
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

func (s *RootStore) UpdateModManifest(path string, manifest *datadir.ModuleManifest, mErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetModManifestState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	mod, err := rootRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ModManifest = manifest
	mod.ModManifestErr = mErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	// TODO! queue module change
	// err = s.queueModuleChange(txn, nil, mod)
	// if err != nil {
	// 	return err
	// }

	txn.Commit()
	return nil
}

func (s *RootStore) SetTerraformVersionState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := rootRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.TerraformVersionState = state
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	// TODO! queue module change
	// err = s.queueModuleChange(txn, nil, mod)
	// if err != nil {
	// 	return err
	// }

	txn.Commit()
	return nil
}

func (s *RootStore) UpdateTerraformAndProviderVersions(modPath string, tfVer *version.Version, pv map[tfaddr.Provider]*version.Version, vErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetTerraformVersionState(modPath, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldMod, err := rootRecordByPath(txn, modPath)
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

	// TODO! queue module change
	// err = s.queueModuleChange(txn, oldMod, mod)
	// if err != nil {
	// 	return err
	// }

	err = updateProviderVersions(txn, modPath, pv)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *RootStore) CallersOfModule(modPath string) ([]*RootRecord, error) {
	txn := s.db.Txn(false)
	it, err := txn.Get(s.tableName, "id")
	if err != nil {
		return nil, err
	}

	callers := make([]*RootRecord, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		record := item.(*RootRecord)

		if record.ModManifest == nil {
			continue
		}
		if record.ModManifest.ContainsLocalModule(modPath) {
			callers = append(callers, record)
		}
	}

	return callers, nil
}

func (s *RootStore) InstalledModuleCalls(modPath string) (map[string]tfmod.InstalledModuleCall, error) {
	mod, err := s.RootRecordByPath(modPath)
	if err != nil {
		return map[string]tfmod.InstalledModuleCall{}, err
	}

	installed := make(map[string]tfmod.InstalledModuleCall)
	if mod.ModManifest != nil {
		for _, record := range mod.ModManifest.Records {
			if record.IsRoot() {
				continue
			}
			installed[record.Key] = tfmod.InstalledModuleCall{
				LocalName:  record.Key,
				SourceAddr: record.SourceAddr,
				Version:    record.Version,
				Path:       filepath.Join(modPath, record.Dir),
			}
		}
	}

	return installed, err
}
