// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"log"

	"github.com/hashicorp/go-memdb"
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

	// TODO: Implement queueStackChange?
	// oldMod := oldObj.(*StackRecord)
	// err = s.queueModuleChange(oldMod, nil)
	// if err != nil {
	// 	return err
	// }

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

func (s *StackStore) SetStackDiagnosticsState(path string, source globalAst.DiagnosticSource, state operation.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := stackCopyByPath(txn, path)
	if err != nil {
		return err
	}
	record.StackDiagnosticsState[source] = state

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *StackStore) UpdateParsedStackFiles(path string, pFiles ast.StackFiles, pErr error) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := stackCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ParsedStackFiles = pFiles

	mod.StackParsingErr = pErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *StackStore) UpdateStackDiagnostics(path string, source globalAst.DiagnosticSource, diags ast.StackDiags) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetStackDiagnosticsState(path, source, operation.OpStateLoaded)
	})
	defer txn.Abort()

	oldMod, err := stackByPath(txn, path)
	if err != nil {
		return err
	}

	mod := oldMod.Copy()
	if mod.StackDiagnostics == nil {
		mod.StackDiagnostics = make(ast.SourceStackDiags)
	}
	mod.StackDiagnostics[source] = diags

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

func (s *StackStore) SetDeployDiagnosticsState(path string, source globalAst.DiagnosticSource, state operation.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	record, err := stackCopyByPath(txn, path)
	if err != nil {
		return err
	}
	record.DeployDiagnosticsState[source] = state

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *StackStore) UpdateParsedDeployFiles(path string, pFiles ast.DeployFiles, pErr error) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := stackCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ParsedDeployFiles = pFiles

	mod.DeployParsingErr = pErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *StackStore) UpdateDeployDiagnostics(path string, source globalAst.DiagnosticSource, diags ast.DeployDiags) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetStackDiagnosticsState(path, source, operation.OpStateLoaded)
	})
	defer txn.Abort()

	oldStack, err := stackByPath(txn, path)
	if err != nil {
		return err
	}

	stack := oldStack.Copy()
	if stack.DeployDiagnostics == nil {
		stack.DeployDiagnostics = make(ast.SourceDeployDiags)
	}
	stack.DeployDiagnostics[source] = diags

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

	mod := newStack(stackPath)
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	// TODO: Implement queueStackChange?
	// err = s.queueModuleChange(nil, mod)
	// if err != nil {
	// 	return err
	// }

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

	oldDiags, newDiags := 0, 0
	if oldRecord != nil {
		oldDiags = oldRecord.StackDiagnostics.Count() + oldRecord.DeployDiagnostics.Count()
	}
	if newRecord != nil {
		newDiags = newRecord.StackDiagnostics.Count() + newRecord.DeployDiagnostics.Count()
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
