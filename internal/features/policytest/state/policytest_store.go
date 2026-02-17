// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/ast"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfpolicytest "github.com/hashicorp/terraform-schema/policytest"
)

type PolicyTestStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger

	// MaxPolicyTestNesting represents how many nesting levels we'd attempt
	// to parse provider requirements before returning error.
	MaxPolicyTestNesting int

	changeStore *globalState.ChangeStore
}

func (s *PolicyTestStore) SetLogger(logger *log.Logger) {
	s.logger = logger
}

func policytestByPath(txn *memdb.Txn, path string) (*PolicyTestRecord, error) {
	obj, err := txn.First(policytestTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &globalState.RecordNotFoundError{
			Source: path,
		}
	}
	return obj.(*PolicyTestRecord), nil
}

func policytestCopyByPath(txn *memdb.Txn, path string) (*PolicyTestRecord, error) {
	policytest, err := policytestByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return policytest.Copy(), nil
}

func (s *PolicyTestStore) Add(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	err := s.add(txn, path)
	if err != nil {
		return err
	}
	txn.Commit()

	return nil
}

func (s *PolicyTestStore) add(txn *memdb.Txn, path string) error {
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

	policytest := newPolicyTest(path)
	err = txn.Insert(s.tableName, policytest)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(nil, policytest)
	if err != nil {
		return err
	}

	return nil
}

func (s *PolicyTestStore) Remove(path string) error {
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

	oldPolicyTest := oldObj.(*PolicyTestRecord)
	err = s.queueRecordChange(oldPolicyTest, nil)
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

func (s *PolicyTestStore) PolicyTestRecordByPath(path string) (*PolicyTestRecord, error) {
	txn := s.db.Txn(false)

	policytest, err := policytestByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return policytest, nil
}

func (s *PolicyTestStore) AddIfNotExists(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	_, err := policytestByPath(txn, path)
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

func (s *PolicyTestStore) LocalPolicyTestMeta(path string) (*tfpolicytest.Meta, error) {
	policytest, err := s.PolicyTestRecordByPath(path)
	if err != nil {
		return nil, err
	}
	if policytest.MetaState != op.OpStateLoaded {
		return nil, fmt.Errorf("%s: policytest data not available", path)
	}
	return &tfpolicytest.Meta{
		Path:      policytest.path,
		Filenames: policytest.Meta.Filenames,
	}, nil
}

func (s *PolicyTestStore) List() ([]*PolicyTestRecord, error) {
	txn := s.db.Txn(false)

	it, err := txn.Get(s.tableName, "id")
	if err != nil {
		return nil, err
	}

	policytest := make([]*PolicyTestRecord, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		record := item.(*PolicyTestRecord)
		policytest = append(policytest, record)
	}

	return policytest, nil
}

func (s *PolicyTestStore) Exists(path string) bool {
	txn := s.db.Txn(false)

	obj, err := txn.First(s.tableName, "id", path)
	if err != nil {
		return false
	}

	return obj != nil
}

func (s *PolicyTestStore) UpdateParsedPolicyTestFiles(path string, pFiles ast.PolicyTestFiles, pErr error) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	policytest, err := policytestCopyByPath(txn, path)
	if err != nil {
		return err
	}

	policytest.ParsedPolicyTestFiles = pFiles

	policytest.PolicyTestParsingErr = pErr

	err = txn.Insert(s.tableName, policytest)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyTestStore) SetMetaState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	policytest, err := policytestCopyByPath(txn, path)
	if err != nil {
		return err
	}

	policytest.MetaState = state
	err = txn.Insert(s.tableName, policytest)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyTestStore) UpdateMetadata(path string, meta *tfpolicytest.Meta, mErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetMetaState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldPolicyTest, err := policytestByPath(txn, path)
	if err != nil {
		return err
	}

	policytest := oldPolicyTest.Copy()
	policytest.Meta = PolicyTestMetadata{
		Filenames: meta.Filenames,
	}
	policytest.MetaErr = mErr

	err = txn.Insert(s.tableName, policytest)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(oldPolicyTest, policytest)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyTestStore) UpdatePolicyTestDiagnostics(path string, source globalAst.DiagnosticSource, diags ast.PolicyTestDiags) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetPolicyTestDiagnosticsState(path, source, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldPolicyTest, err := policytestByPath(txn, path)
	if err != nil {
		return err
	}

	policytest := oldPolicyTest.Copy()
	if policytest.PolicyTestDiagnostics == nil {
		policytest.PolicyTestDiagnostics = make(ast.SourcePolicyTestDiags)
	}
	policytest.PolicyTestDiagnostics[source] = diags

	err = txn.Insert(s.tableName, policytest)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(oldPolicyTest, policytest)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyTestStore) SetPolicyTestDiagnosticsState(path string, source globalAst.DiagnosticSource, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	policytest, err := policytestCopyByPath(txn, path)
	if err != nil {
		return err
	}
	policytest.PolicyTestDiagnosticsState[source] = state

	err = txn.Insert(s.tableName, policytest)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyTestStore) SetReferenceTargetsState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	policytest, err := policytestCopyByPath(txn, path)
	if err != nil {
		return err
	}

	policytest.RefTargetsState = state
	err = txn.Insert(s.tableName, policytest)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyTestStore) UpdateReferenceTargets(path string, refs reference.Targets, rErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetReferenceTargetsState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	policytest, err := policytestCopyByPath(txn, path)
	if err != nil {
		return err
	}

	policytest.RefTargets = refs
	policytest.RefTargetsErr = rErr

	err = txn.Insert(s.tableName, policytest)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyTestStore) SetReferenceOriginsState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	policytest, err := policytestCopyByPath(txn, path)
	if err != nil {
		return err
	}

	policytest.RefOriginsState = state
	err = txn.Insert(s.tableName, policytest)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyTestStore) UpdateReferenceOrigins(path string, origins reference.Origins, roErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetReferenceOriginsState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	policytest, err := policytestCopyByPath(txn, path)
	if err != nil {
		return err
	}

	policytest.RefOrigins = origins
	policytest.RefOriginsErr = roErr

	err = txn.Insert(s.tableName, policytest)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyTestStore) queueRecordChange(oldPolicyTest, newPolicyTest *PolicyTestRecord) error {
	changes := globalState.Changes{}

	switch {
	// policytest removed
	case oldPolicyTest != nil && newPolicyTest == nil:
		changes.IsRemoval = true
	}

	oldDiags, newDiags := 0, 0
	if oldPolicyTest != nil {
		oldDiags = oldPolicyTest.PolicyTestDiagnostics.Count()
	}
	if newPolicyTest != nil {
		newDiags = newPolicyTest.PolicyTestDiagnostics.Count()
	}
	// Comparing diagnostics accurately could be expensive
	// so we just treat any non-empty diags as a change
	if oldDiags > 0 || newDiags > 0 {
		changes.Diagnostics = true
	}

	var policytestHandle document.DirHandle
	if oldPolicyTest != nil {
		policytestHandle = document.DirHandleFromPath(oldPolicyTest.Path())
	} else {
		policytestHandle = document.DirHandleFromPath(newPolicyTest.Path())
	}

	return s.changeStore.QueueChange(policytestHandle, changes)
}

func (f *PolicyTestStore) MetadataReady(dir document.DirHandle) (<-chan struct{}, bool, error) {
	rTxn := f.db.Txn(false)

	wCh, recordObj, err := rTxn.FirstWatch(f.tableName, "policytest_state", dir.Path(), op.OpStateLoaded)
	if err != nil {
		return nil, false, err
	}
	if recordObj != nil {
		return wCh, true, nil
	}

	return wCh, false, nil
}
