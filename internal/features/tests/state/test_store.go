// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"log"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/tests/ast"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	tftest "github.com/hashicorp/terraform-schema/test"
)

type TestStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger

	changeStore          *globalState.ChangeStore
	providerSchemasStore *globalState.ProviderSchemaStore
}

func (s *TestStore) SetLogger(logger *log.Logger) {
	s.logger = logger
}

func (s *TestStore) Add(testPath string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	err := s.add(txn, testPath)
	if err != nil {
		return err
	}
	txn.Commit()

	return nil
}

func (s *TestStore) Remove(testPath string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	oldObj, err := txn.First(s.tableName, "id", testPath)
	if err != nil {
		return err
	}

	if oldObj == nil {
		// already removed
		return nil
	}

	oldRecord := oldObj.(*TestRecord)
	err = s.queueRecordChange(oldRecord, nil)
	if err != nil {
		return err
	}

	_, err = txn.DeleteAll(s.tableName, "id", testPath)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *TestStore) List() ([]*TestRecord, error) {
	txn := s.db.Txn(false)

	it, err := txn.Get(s.tableName, "id")
	if err != nil {
		return nil, err
	}

	records := make([]*TestRecord, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		record := item.(*TestRecord)
		records = append(records, record)
	}

	return records, nil
}

func (s *TestStore) TestRecordByPath(path string) (*TestRecord, error) {
	txn := s.db.Txn(false)

	mod, err := testByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return mod, nil
}

func (s *TestStore) Exists(path string) bool {
	txn := s.db.Txn(false)

	obj, err := txn.First(s.tableName, "id", path)
	if err != nil {
		return false
	}

	return obj != nil
}

func (s *TestStore) AddIfNotExists(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	_, err := testByPath(txn, path)
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

func (s *TestStore) SetDiagnosticsState(path string, source globalAst.DiagnosticSource, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := testCopyByPath(txn, path)
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

func (s *TestStore) UpdateParsedFiles(path string, pFiles ast.Files, pErr error) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := testCopyByPath(txn, path)
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

func (s *TestStore) UpdateDiagnostics(path string, source globalAst.DiagnosticSource, diags ast.Diagnostics) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetDiagnosticsState(path, source, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldMod, err := testByPath(txn, path)
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

func (s *TestStore) SetMetaState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := testCopyByPath(txn, path)
	if err != nil {
		return err
	}

	record.MetaState = state
	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *TestStore) UpdateMetadata(path string, meta *tftest.Meta, mErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetMetaState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldRecord, err := testByPath(txn, path)
	if err != nil {
		return err
	}

	record := oldRecord.Copy()
	record.Meta = TestMetadata{
		Filenames: meta.Filenames,
	}
	record.MetaErr = mErr

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

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

func (s *TestStore) SetPreloadEmbeddedSchemaState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := testCopyByPath(txn, path)
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

func (s *TestStore) add(txn *memdb.Txn, testPath string) error {
	// TODO: Introduce Exists method to Txn?
	obj, err := txn.First(s.tableName, "id", testPath)
	if err != nil {
		return err
	}
	if obj != nil {
		return &globalState.AlreadyExistsError{
			Idx: testPath,
		}
	}

	record := newTest(testPath)
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

func (s *TestStore) SetReferenceTargetsState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := testCopyByPath(txn, path)
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

func (s *TestStore) UpdateReferenceTargets(path string, refs reference.Targets, rErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetReferenceTargetsState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	mod, err := testCopyByPath(txn, path)
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

func (s *TestStore) SetReferenceOriginsState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := testCopyByPath(txn, path)
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

func (s *TestStore) UpdateReferenceOrigins(path string, origins reference.Origins, roErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetReferenceOriginsState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	mod, err := testCopyByPath(txn, path)
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

func testByPath(txn *memdb.Txn, path string) (*TestRecord, error) {
	obj, err := txn.First(testsTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &globalState.RecordNotFoundError{
			Source: path,
		}
	}
	return obj.(*TestRecord), nil
}

func testCopyByPath(txn *memdb.Txn, path string) (*TestRecord, error) {
	record, err := testByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return record.Copy(), nil
}

func (s *TestStore) queueRecordChange(oldRecord, newRecord *TestRecord) error {
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

func (s *TestStore) ProviderSchema(modPath string, addr tfaddr.Provider, vc version.Constraints) (*tfschema.ProviderSchema, error) {
	return s.providerSchemasStore.ProviderSchema(modPath, addr, vc)
}
