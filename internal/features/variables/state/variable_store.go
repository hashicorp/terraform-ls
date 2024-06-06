// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"log"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/variables/ast"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

type VariableStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger

	changeStore *globalState.ChangeStore
}

func (s *VariableStore) SetLogger(logger *log.Logger) {
	s.logger = logger
}

func (s *VariableStore) Add(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	err := s.add(txn, path)
	if err != nil {
		return err
	}
	txn.Commit()

	return nil
}

func (s *VariableStore) add(txn *memdb.Txn, path string) error {
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

	record := newVariableRecord(path)
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

func (s *VariableStore) AddIfNotExists(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	_, err := variableRecordByPath(txn, path)
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

func (s *VariableStore) Remove(path string) error {
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

	oldRecord := oldObj.(*VariableRecord)
	err = s.queueRecordChange(oldRecord, nil)
	if err != nil {
		return err
	}

	_, err = txn.DeleteAll(s.tableName, "id", path)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *VariableStore) List() ([]*VariableRecord, error) {
	txn := s.db.Txn(false)

	it, err := txn.Get(s.tableName, "id")
	if err != nil {
		return nil, err
	}

	records := make([]*VariableRecord, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		record := item.(*VariableRecord)
		records = append(records, record)
	}

	return records, nil
}

func (s *VariableStore) Exists(path string) bool {
	txn := s.db.Txn(false)

	obj, err := txn.First(s.tableName, "id", path)
	if err != nil {
		return false
	}

	return obj != nil
}

func (s *VariableStore) VariableRecordByPath(path string) (*VariableRecord, error) {
	txn := s.db.Txn(false)

	record, err := variableRecordByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func variableRecordByPath(txn *memdb.Txn, path string) (*VariableRecord, error) {
	obj, err := txn.First(variableTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &globalState.RecordNotFoundError{
			Source: path,
		}
	}
	return obj.(*VariableRecord), nil
}

func variableRecordCopyByPath(txn *memdb.Txn, path string) (*VariableRecord, error) {
	record, err := variableRecordByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return record.Copy(), nil
}

func (s *VariableStore) UpdateParsedVarsFiles(path string, vFiles ast.VarsFiles, vErr error) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := variableRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}

	record.ParsedVarsFiles = vFiles
	record.VarsParsingErr = vErr

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *VariableStore) UpdateVarsDiagnostics(path string, source globalAst.DiagnosticSource, diags ast.VarsDiags) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetVarsDiagnosticsState(path, source, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldRecord, err := variableRecordByPath(txn, path)
	if err != nil {
		return err
	}

	record := oldRecord.Copy()
	if record.VarsDiagnostics == nil {
		record.VarsDiagnostics = make(ast.SourceVarsDiags)
	}
	record.VarsDiagnostics[source] = diags

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

func (s *VariableStore) SetVarsDiagnosticsState(path string, source globalAst.DiagnosticSource, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := variableRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}
	record.VarsDiagnosticsState[source] = state

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *VariableStore) SetVarsReferenceOriginsState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := variableRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}

	record.VarsRefOriginsState = state
	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *VariableStore) UpdateVarsReferenceOrigins(path string, origins reference.Origins, roErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetVarsReferenceOriginsState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	record, err := variableRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}

	record.VarsRefOrigins = origins
	record.VarsRefOriginsErr = roErr

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *VariableStore) queueRecordChange(oldRecord, newRecord *VariableRecord) error {
	changes := globalState.Changes{}

	oldDiags, newDiags := 0, 0
	if oldRecord != nil {
		oldDiags = oldRecord.VarsDiagnostics.Count()
	}
	if newRecord != nil {
		newDiags = newRecord.VarsDiagnostics.Count()
	}
	// Comparing diagnostics accurately could be expensive
	// so we just treat any non-empty diags as a change
	if oldDiags > 0 || newDiags > 0 {
		changes.Diagnostics = true
	}

	oldOrigins := 0
	if oldRecord != nil {
		oldOrigins = len(oldRecord.VarsRefOrigins)
	}
	newOrigins := 0
	if newRecord != nil {
		newOrigins = len(newRecord.VarsRefOrigins)
	}
	if oldOrigins != newOrigins {
		changes.ReferenceOrigins = true
	}

	var dir document.DirHandle
	if oldRecord != nil {
		dir = document.DirHandleFromPath(oldRecord.Path())
	} else {
		dir = document.DirHandleFromPath(newRecord.Path())
	}

	return s.changeStore.QueueChange(dir, changes)
}
