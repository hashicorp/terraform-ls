// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"log"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/search/ast"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	tfsearch "github.com/hashicorp/terraform-schema/search"
)

type SearchStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger

	changeStore          *globalState.ChangeStore
	providerSchemasStore *globalState.ProviderSchemaStore
}

func (s *SearchStore) SetLogger(logger *log.Logger) {
	s.logger = logger
}

func (s *SearchStore) Add(searchPath string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	err := s.add(txn, searchPath)
	if err != nil {
		return err
	}
	txn.Commit()

	return nil
}

func (s *SearchStore) Remove(searchPath string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	oldObj, err := txn.First(s.tableName, "id", searchPath)
	if err != nil {
		return err
	}

	if oldObj == nil {
		// already removed
		return nil
	}

	oldRecord := oldObj.(*SearchRecord)
	err = s.queueRecordChange(oldRecord, nil)
	if err != nil {
		return err
	}

	_, err = txn.DeleteAll(s.tableName, "id", searchPath)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *SearchStore) List() ([]*SearchRecord, error) {
	txn := s.db.Txn(false)

	it, err := txn.Get(s.tableName, "id")
	if err != nil {
		return nil, err
	}

	searchRecords := make([]*SearchRecord, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		search := item.(*SearchRecord)
		searchRecords = append(searchRecords, search)
	}

	return searchRecords, nil
}

func (s *SearchStore) GetSearchRecordByPath(path string) (*SearchRecord, error) {
	txn := s.db.Txn(false)

	mod, err := searchByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return mod, nil
}

func (s *SearchStore) Exists(path string) bool {
	txn := s.db.Txn(false)

	obj, err := txn.First(s.tableName, "id", path)
	if err != nil {
		return false
	}

	return obj != nil
}

func (s *SearchStore) AddIfNotExists(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	_, err := searchByPath(txn, path)
	if err == nil {
		return nil
	}

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

func (s *SearchStore) SetDiagnosticsState(path string, source globalAst.DiagnosticSource, state operation.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := searchCopyByPath(txn, path)
	if err != nil {
		return err
	}
	record.DiagnosticsState[source] = state

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *SearchStore) UpdateParsedFiles(path string, pFiles ast.Files, pErr error) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := searchCopyByPath(txn, path)
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

func (s *SearchStore) UpdateDiagnostics(path string, source globalAst.DiagnosticSource, diags ast.Diagnostics) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetDiagnosticsState(path, source, operation.OpStateLoaded)
	})
	defer txn.Abort()

	oldMod, err := searchByPath(txn, path)
	if err != nil {
		return err
	}

	mod := oldMod.Copy()
	if mod.Diagnostics == nil {
		mod.Diagnostics = make(ast.SourceDiagnostics)
	}
	mod.Diagnostics[source] = diags

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

func (s *SearchStore) SetMetaState(path string, state operation.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	search, err := searchCopyByPath(txn, path)
	if err != nil {
		return err
	}

	search.MetaState = state
	err = txn.Insert(s.tableName, search)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *SearchStore) UpdateMetadata(path string, meta *tfsearch.Meta, mErr error, providerReqs tfsearch.ProviderRequirements, coreRequirements version.Constraints) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetMetaState(path, operation.OpStateLoaded)
	})
	defer txn.Abort()

	oldRecord, err := searchByPath(txn, path)
	if err != nil {
		return err
	}

	record := oldRecord.Copy()
	record.Meta = SearchMetadata{
		Lists:                meta.Lists,
		Variables:            meta.Variables,
		Filenames:            meta.Filenames,
		ProviderReferences:   meta.ProviderReferences,
		ProviderRequirements: providerReqs,
		CoreRequirements:     coreRequirements,
	}
	record.MetaErr = mErr

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(oldRecord, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *SearchStore) SetPreloadEmbeddedSchemaState(path string, state operation.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := searchCopyByPath(txn, path)
	if err != nil {
		return err
	}

	record.PreloadEmbeddedSchemaState = state
	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *SearchStore) SetReferenceTargetsState(path string, state operation.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := searchCopyByPath(txn, path)
	if err != nil {
		return err
	}

	record.RefTargetsState = state
	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *SearchStore) UpdateReferenceTargets(path string, refs reference.Targets, rErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetReferenceTargetsState(path, operation.OpStateLoaded)
	})
	defer txn.Abort()

	record, err := searchCopyByPath(txn, path)
	if err != nil {
		return err
	}

	record.RefTargets = refs
	record.RefTargetsErr = rErr

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *SearchStore) SetReferenceOriginsState(path string, state operation.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	search, err := searchCopyByPath(txn, path)
	if err != nil {
		return err
	}

	search.RefOriginsState = state
	err = txn.Insert(s.tableName, search)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *SearchStore) UpdateReferenceOrigins(path string, origins reference.Origins, roErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetReferenceOriginsState(path, operation.OpStateLoaded)
	})
	defer txn.Abort()

	search, err := searchCopyByPath(txn, path)
	if err != nil {
		return err
	}

	search.RefOrigins = origins
	search.RefOriginsErr = roErr

	err = txn.Insert(s.tableName, search)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *SearchStore) add(txn *memdb.Txn, searchPath string) error {
	// TODO: Introduce Exists method to Txn?
	obj, err := txn.First(s.tableName, "id", searchPath)
	if err != nil {
		return err
	}
	if obj != nil {
		return &globalState.AlreadyExistsError{
			Idx: searchPath,
		}
	}

	record := newSearch(searchPath)
	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(nil, record)
	if err != nil {
		return err
	}

	return nil
}

func searchByPath(txn *memdb.Txn, path string) (*SearchRecord, error) {
	obj, err := txn.First(searchTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &globalState.RecordNotFoundError{
			Source: path,
		}
	}
	return obj.(*SearchRecord), nil
}

func searchCopyByPath(txn *memdb.Txn, path string) (*SearchRecord, error) {
	record, err := searchByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return record.Copy(), nil
}

func (s *SearchStore) queueRecordChange(oldRecord, newRecord *SearchRecord) error {
	changes := globalState.Changes{}

	oldDiags, newDiags := 0, 0
	if oldRecord != nil {
		oldDiags = oldRecord.Diagnostics.Count()
	}
	if newRecord != nil {
		newDiags = newRecord.Diagnostics.Count()
	}
	// Comparing diagnostics accurately could be expensive
	// so we just treat any non-empty diags as a change
	if oldDiags > 0 || newDiags > 0 {
		changes.Diagnostics = true
	}

	var dir document.DirHandle
	if oldRecord != nil {
		dir = document.DirHandleFromPath(oldRecord.Path())
	} else {
		dir = document.DirHandleFromPath(newRecord.Path())
	}

	return s.changeStore.QueueChange(dir, changes)
}

func (s *SearchStore) ProviderSchema(modPath string, addr tfaddr.Provider, vc version.Constraints) (*tfschema.ProviderSchema, error) {
	return s.providerSchemasStore.ProviderSchema(modPath, addr, vc)
}

func (s *SearchStore) ProviderRequirementsForModule(modPath string) (tfsearch.ProviderRequirements, error) {
	return s.providerRequirementsForModule(modPath, 0)
}

func (s *SearchStore) providerRequirementsForModule(searchPath string, level int) (tfsearch.ProviderRequirements, error) {
	mod, err := s.GetSearchRecordByPath(searchPath)
	if err != nil {
		// It's possible that the configuration contains a module with an
		// invalid local source, so we just ignore it if it can't be found.
		// This allows us to still return provider requirements for other modules
		return tfsearch.ProviderRequirements{}, nil
	}

	level++

	requirements := make(tfsearch.ProviderRequirements, 0)
	for k, v := range mod.Meta.ProviderRequirements {
		requirements[k] = v
	}

	return requirements, nil
}
