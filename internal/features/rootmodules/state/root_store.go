// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"log"
	"path/filepath"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/document"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

type RootStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger

	changeStore         *globalState.ChangeStore
	providerSchemaStore *globalState.ProviderSchemaStore
}

func (s *RootStore) SetLogger(logger *log.Logger) {
	s.logger = logger
}

func (s *RootStore) Add(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	err := s.add(txn, path)
	if err != nil {
		return err
	}
	txn.Commit()

	return nil
}

func (s *RootStore) add(txn *memdb.Txn, path string) error {
	// TODO: Introduce Exists method to Txn?
	obj, err := txn.First(s.tableName, "id", path)
	if err != nil {
		return err
	}
	if obj != nil {
		return &globalState.AlreadyExistsError{
			Idx: path,
		}
	}

	record := newRootRecord(path)
	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	return nil
}

func (s *RootStore) Remove(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	oldObj, err := txn.First(s.tableName, "id", path)
	if err != nil {
		return err
	}

	if oldObj == nil {
		// already removed
		return nil
	}

	_, err = txn.DeleteAll(s.tableName, "id", path)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *RootStore) RootRecordByPath(path string) (*RootRecord, error) {
	txn := s.db.Txn(false)

	record, err := rootRecordByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *RootStore) AddIfNotExists(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	_, err := rootRecordByPath(txn, path)
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

func (s *RootStore) Exists(path string) bool {
	txn := s.db.Txn(false)

	obj, err := txn.First(s.tableName, "id", path)
	if err != nil {
		return false
	}

	return obj != nil
}

func (s *RootStore) List() ([]*RootRecord, error) {
	txn := s.db.Txn(false)

	it, err := txn.Get(s.tableName, "id")
	if err != nil {
		return nil, err
	}

	records := make([]*RootRecord, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		record := item.(*RootRecord)
		records = append(records, record)
	}

	return records, nil
}

func rootRecordByPath(txn *memdb.Txn, path string) (*RootRecord, error) {
	obj, err := txn.First(rootTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &globalState.RecordNotFoundError{
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

	oldRecord, err := rootRecordByPath(txn, path)
	if err != nil {
		return err
	}

	record := oldRecord.Copy()
	record.InstalledProviders = pvs
	record.InstalledProvidersErr = pvErr

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(oldRecord, record)
	if err != nil {
		return err
	}

	err = s.providerSchemaStore.UpdateProviderVersions(path, pvs)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *RootStore) SetInstalledProvidersState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := rootRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}

	record.InstalledProvidersState = state

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *RootStore) SetModManifestState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := rootRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}

	record.ModManifestState = state

	err = txn.Insert(s.tableName, record)
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

	record, err := rootRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}

	record.ModManifest = manifest
	record.ModManifestErr = mErr

	// Only overwrite InstalledModules if manifest is not nil to not overwrite modules that might have
	// been set via UpdateTerraformSources() – we should probably refactor this to not share the same field
	if manifest != nil {
		record.InstalledModules = InstalledModulesFromManifest(manifest)
	}

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(nil, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *RootStore) SetTerraformSourcesState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := rootRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}

	record.TerraformSourcesState = state

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *RootStore) UpdateTerraformSources(path string, manifest *datadir.TerraformSources, mErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetTerraformSourcesState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	record, err := rootRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}

	record.TerraformSources = manifest
	record.TerraformSourcesErr = mErr
	// Only overwrite InstalledModules if manifest is not nil to not overwrite modules that might have
	// been set via UpdateModManifest() – we should probably refactor this to not share the same field
	if manifest != nil {
		record.InstalledModules = InstalledModulesFromTerraformSources(path, manifest, s.logger)
	}

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(nil, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *RootStore) SetTerraformVersionState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := rootRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}

	record.TerraformVersionState = state
	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(nil, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *RootStore) UpdateTerraformAndProviderVersions(path string, tfVer *version.Version, pv map[tfaddr.Provider]*version.Version, vErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetTerraformVersionState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldRecord, err := rootRecordByPath(txn, path)
	if err != nil {
		return err
	}

	record := oldRecord.Copy()
	record.TerraformVersion = tfVer
	record.TerraformVersionErr = vErr

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(oldRecord, record)
	if err != nil {
		return err
	}

	err = s.providerSchemaStore.UpdateProviderVersions(path, pv)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *RootStore) CallersOfModule(path string) ([]string, error) {
	txn := s.db.Txn(false)
	it, err := txn.Get(s.tableName, "id")
	if err != nil {
		return nil, err
	}

	callers := make([]string, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		record := item.(*RootRecord)

		if record.ModManifest == nil {
			continue
		}
		if record.ModManifest.ContainsLocalModule(path) {
			callers = append(callers, record.path)
		}
	}

	return callers, nil
}

func (s *RootStore) InstalledModuleCalls(path string) (map[string]tfmod.InstalledModuleCall, error) {
	record, err := s.RootRecordByPath(path)
	if err != nil {
		return map[string]tfmod.InstalledModuleCall{}, err
	}

	installed := make(map[string]tfmod.InstalledModuleCall)
	if record.ModManifest != nil {
		for _, record := range record.ModManifest.Records {
			if record.IsRoot() {
				continue
			}
			installed[record.Key] = tfmod.InstalledModuleCall{
				LocalName:  record.Key,
				SourceAddr: record.SourceAddr,
				Version:    record.Version,
				Path:       filepath.Join(path, record.Dir),
			}
		}
	}

	return installed, err
}

func (s *RootStore) TerraformSourcesDirectories(path string) []string {
	dirs := make([]string, 0)

	record, err := s.RootRecordByPath(path)
	if err != nil {
		return dirs
	}

	// If terraform-sources.json file was loaded, we assume that InstalledModules
	// contains them as modules.json and terraform-sources.json are not expected to exist at the same time
	if record.TerraformSourcesState == op.OpStateLoaded {
		for _, dir := range record.InstalledModules {
			dirs = append(dirs, dir)
		}
	}

	return dirs
}

func (s *RootStore) queueRecordChange(oldRecord, newRecord *RootRecord) error {
	changes := globalState.Changes{}

	switch {
	// new record added
	case oldRecord == nil && newRecord != nil:
		if newRecord.TerraformVersion != nil {
			changes.TerraformVersion = true
		}
		if len(newRecord.InstalledProviders) > 0 {
			changes.InstalledProviders = true
		}
	// record removed
	case oldRecord != nil && newRecord == nil:
		changes.IsRemoval = true

		if oldRecord.TerraformVersion != nil {
			changes.TerraformVersion = true
		}
		if len(oldRecord.InstalledProviders) > 0 {
			changes.InstalledProviders = true
		}
	// record changed
	default:
		if !oldRecord.TerraformVersion.Equal(newRecord.TerraformVersion) {
			changes.TerraformVersion = true
		}
		if !oldRecord.InstalledProviders.Equals(newRecord.InstalledProviders) {
			changes.InstalledProviders = true
		}
	}

	var dir document.DirHandle
	if oldRecord != nil {
		dir = document.DirHandleFromPath(oldRecord.Path())
	} else {
		dir = document.DirHandleFromPath(newRecord.Path())
	}

	return s.changeStore.QueueChange(dir, changes)
}

func (s *RootStore) SetProviderSchemaState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := rootRecordCopyByPath(txn, path)
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

func (s *RootStore) FinishProviderSchemaLoading(path string, psErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetProviderSchemaState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldMod, err := rootRecordByPath(txn, path)
	if err != nil {
		return err
	}

	mod := oldMod.Copy()
	mod.ProviderSchemaErr = psErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(oldMod, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

// RecordWithVersion returns the first record that has a Terraform version
func (s *RootStore) RecordWithVersion() (*RootRecord, error) {
	txn := s.db.Txn(false)

	it, err := txn.Get(s.tableName, "id")
	if err != nil {
		return nil, err
	}

	for item := it.Next(); item != nil; item = it.Next() {
		record := item.(*RootRecord)
		if record.TerraformVersion != nil {
			return record, nil
		}
	}

	return nil, &globalState.RecordNotFoundError{}
}
