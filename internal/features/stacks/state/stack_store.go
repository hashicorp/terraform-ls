// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"log"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/ast"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

type StackStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger

	changeStore *globalState.ChangeStore
}

func (s *StackStore) SetLogger(logger *log.Logger) {
	s.logger = logger
}

func (s *StackStore) Add(stackPath string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	err := s.add(txn, stackPath)
	if err != nil {
		return err
	}
	txn.Commit()

	return nil
}

func (s *StackStore) Remove(stackPath string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	oldObj, err := txn.First(s.tableName, "id", stackPath)
	if err != nil {
		return err
	}

	if oldObj == nil {
		// already removed
		return nil
	}

	oldRecord := oldObj.(*StackRecord)
	err = s.queueRecordChange(oldRecord, nil)
	if err != nil {
		return err
	}

	_, err = txn.DeleteAll(s.tableName, "id", stackPath)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *StackStore) List() ([]*StackRecord, error) {
	txn := s.db.Txn(false)

	it, err := txn.Get(s.tableName, "id")
	if err != nil {
		return nil, err
	}

	stacks := make([]*StackRecord, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		stack := item.(*StackRecord)
		stacks = append(stacks, stack)
	}

	return stacks, nil
}

func (s *StackStore) StackRecordByPath(path string) (*StackRecord, error) {
	txn := s.db.Txn(false)

	mod, err := stackByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return mod, nil
}

func (s *StackStore) Exists(path string) bool {
	txn := s.db.Txn(false)

	obj, err := txn.First(s.tableName, "id", path)
	if err != nil {
		return false
	}

	return obj != nil
}

func (s *StackStore) AddIfNotExists(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	_, err := stackByPath(txn, path)
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

func (s *StackStore) SetDiagnosticsState(path string, source globalAst.DiagnosticSource, state operation.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := stackCopyByPath(txn, path)
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

func (s *StackStore) UpdateParsedFiles(path string, pFiles ast.Files, pErr error) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := stackCopyByPath(txn, path)
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

func (s *StackStore) UpdateDiagnostics(path string, source globalAst.DiagnosticSource, diags ast.Diagnostics) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetDiagnosticsState(path, source, operation.OpStateLoaded)
	})
	defer txn.Abort()

	oldMod, err := stackByPath(txn, path)
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

func (s *StackStore) setTerraformVersionWithChangeNotification(path string, version *version.Version, vErr error, state operation.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	oldStack, err := stackByPath(txn, path)
	if err != nil {
		return err
	}
	stack := oldStack.Copy()

	stack.TerraformVersion = version
	stack.TerraformVersionErr = vErr
	stack.TerraformVersionState = state

	err = txn.Insert(s.tableName, stack)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(oldStack, stack)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *StackStore) SetTerraformVersion(path string, version *version.Version) error {
	return s.setTerraformVersionWithChangeNotification(path, version, nil, operation.OpStateLoaded)
}

func (s *StackStore) SetTerraformVersionError(path string, vErr error) error {
	return s.setTerraformVersionWithChangeNotification(path, nil, vErr, operation.OpStateLoaded)
}

func (s *StackStore) SetTerraformVersionState(path string, state operation.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	stack, err := stackCopyByPath(txn, path)
	if err != nil {
		return err
	}

	stack.TerraformVersionState = state
	err = txn.Insert(s.tableName, stack)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *StackStore) add(txn *memdb.Txn, stackPath string) error {
	// TODO: Introduce Exists method to Txn?
	obj, err := txn.First(s.tableName, "id", stackPath)
	if err != nil {
		return err
	}
	if obj != nil {
		return &globalState.AlreadyExistsError{
			Idx: stackPath,
		}
	}

	record := newStack(stackPath)
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

func stackByPath(txn *memdb.Txn, path string) (*StackRecord, error) {
	obj, err := txn.First(stackTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &globalState.RecordNotFoundError{
			Source: path,
		}
	}
	return obj.(*StackRecord), nil
}

func stackCopyByPath(txn *memdb.Txn, path string) (*StackRecord, error) {
	record, err := stackByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return record.Copy(), nil
}

func (s *StackStore) queueRecordChange(oldRecord, newRecord *StackRecord) error {
	changes := globalState.Changes{}

	switch {
	// new record added
	case oldRecord == nil && newRecord != nil:
		if newRecord.TerraformVersion != nil {
			changes.TerraformVersion = true
		}
	// record removed
	case oldRecord != nil && newRecord == nil:
		changes.IsRemoval = true

		if oldRecord.TerraformVersion != nil {
			changes.TerraformVersion = true
		}
	// record changed
	default:
		if oldRecord.TerraformVersion == nil || !oldRecord.TerraformVersion.Equal(newRecord.TerraformVersion) {
			changes.TerraformVersion = true
		}
	}

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
