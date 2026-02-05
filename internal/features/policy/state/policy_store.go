// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/policy/ast"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfpolicy "github.com/hashicorp/terraform-schema/policy"
)

type PolicyStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger

	// MaxPolicyNesting represents how many nesting levels we'd attempt
	// to parse provider requirements before returning error.
	MaxPolicyNesting int

	changeStore *globalState.ChangeStore
}

func (s *PolicyStore) SetLogger(logger *log.Logger) {
	s.logger = logger
}

func policyByPath(txn *memdb.Txn, path string) (*PolicyRecord, error) {
	obj, err := txn.First(policyTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &globalState.RecordNotFoundError{
			Source: path,
		}
	}
	return obj.(*PolicyRecord), nil
}

func policyCopyByPath(txn *memdb.Txn, path string) (*PolicyRecord, error) {
	policy, err := policyByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return policy.Copy(), nil
}

func (s *PolicyStore) Add(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	err := s.add(txn, path)
	if err != nil {
		return err
	}
	txn.Commit()

	return nil
}

func (s *PolicyStore) add(txn *memdb.Txn, path string) error {
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

	policy := newPolicy(path)
	err = txn.Insert(s.tableName, policy)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(nil, policy)
	if err != nil {
		return err
	}

	return nil
}

func (s *PolicyStore) Remove(path string) error {
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

	oldPolicy := oldObj.(*PolicyRecord)
	err = s.queueRecordChange(oldPolicy, nil)
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

func (s *PolicyStore) PolicyRecordByPath(path string) (*PolicyRecord, error) {
	txn := s.db.Txn(false)

	policy, err := policyByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return policy, nil
}

func (s *PolicyStore) AddIfNotExists(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	_, err := policyByPath(txn, path)
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

func (s *PolicyStore) LocalPolicyMeta(path string) (*tfpolicy.Meta, error) {
	policy, err := s.PolicyRecordByPath(path)
	if err != nil {
		return nil, err
	}
	if policy.MetaState != op.OpStateLoaded {
		return nil, fmt.Errorf("%s: policy data not available", path)
	}
	return &tfpolicy.Meta{
		Path:      policy.path,
		Filenames: policy.Meta.Filenames,

		CoreRequirements: policy.Meta.CoreRequirements,

		ResourcePolicies: policy.Meta.ResourcePolicies,
		ProviderPolicies: policy.Meta.ProviderPolicies,
		ModulePolicies:   policy.Meta.ModulePolicies,
	}, nil
}

func (s *PolicyStore) List() ([]*PolicyRecord, error) {
	txn := s.db.Txn(false)

	it, err := txn.Get(s.tableName, "id")
	if err != nil {
		return nil, err
	}

	policy := make([]*PolicyRecord, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		record := item.(*PolicyRecord)
		policy = append(policy, record)
	}

	return policy, nil
}

func (s *PolicyStore) Exists(path string) bool {
	txn := s.db.Txn(false)

	obj, err := txn.First(s.tableName, "id", path)
	if err != nil {
		return false
	}

	return obj != nil
}

func (s *PolicyStore) UpdateParsedPolicyFiles(path string, pFiles ast.PolicyFiles, pErr error) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	policy, err := policyCopyByPath(txn, path)
	if err != nil {
		return err
	}

	policy.ParsedPolicyFiles = pFiles

	policy.PolicyParsingErr = pErr

	err = txn.Insert(s.tableName, policy)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyStore) SetMetaState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	policy, err := policyCopyByPath(txn, path)
	if err != nil {
		return err
	}

	policy.MetaState = state
	err = txn.Insert(s.tableName, policy)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyStore) UpdateMetadata(path string, meta *tfpolicy.Meta, mErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetMetaState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldPolicy, err := policyByPath(txn, path)
	if err != nil {
		return err
	}

	policy := oldPolicy.Copy()
	policy.Meta = PolicyMetadata{
		CoreRequirements: meta.CoreRequirements,
		ResourcePolicies: meta.ResourcePolicies,
		ProviderPolicies: meta.ProviderPolicies,
		ModulePolicies:   meta.ModulePolicies,

		Filenames: meta.Filenames,
	}
	policy.MetaErr = mErr

	err = txn.Insert(s.tableName, policy)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(oldPolicy, policy)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyStore) UpdatePolicyDiagnostics(path string, source globalAst.DiagnosticSource, diags ast.PolicyDiags) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetPolicyDiagnosticsState(path, source, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldPolicy, err := policyByPath(txn, path)
	if err != nil {
		return err
	}

	policy := oldPolicy.Copy()
	if policy.PolicyDiagnostics == nil {
		policy.PolicyDiagnostics = make(ast.SourcePolicyDiags)
	}
	policy.PolicyDiagnostics[source] = diags

	err = txn.Insert(s.tableName, policy)
	if err != nil {
		return err
	}

	err = s.queueRecordChange(oldPolicy, policy)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyStore) SetPolicyDiagnosticsState(path string, source globalAst.DiagnosticSource, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	policy, err := policyCopyByPath(txn, path)
	if err != nil {
		return err
	}
	policy.PolicyDiagnosticsState[source] = state

	err = txn.Insert(s.tableName, policy)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyStore) SetReferenceTargetsState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	policy, err := policyCopyByPath(txn, path)
	if err != nil {
		return err
	}

	policy.RefTargetsState = state
	err = txn.Insert(s.tableName, policy)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyStore) UpdateReferenceTargets(path string, refs reference.Targets, rErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetReferenceTargetsState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	policy, err := policyCopyByPath(txn, path)
	if err != nil {
		return err
	}

	policy.RefTargets = refs
	policy.RefTargetsErr = rErr

	err = txn.Insert(s.tableName, policy)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyStore) SetReferenceOriginsState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	policy, err := policyCopyByPath(txn, path)
	if err != nil {
		return err
	}

	policy.RefOriginsState = state
	err = txn.Insert(s.tableName, policy)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyStore) UpdateReferenceOrigins(path string, origins reference.Origins, roErr error) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetReferenceOriginsState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	policy, err := policyCopyByPath(txn, path)
	if err != nil {
		return err
	}

	policy.RefOrigins = origins
	policy.RefOriginsErr = roErr

	err = txn.Insert(s.tableName, policy)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *PolicyStore) queueRecordChange(oldPolicy, newPolicy *PolicyRecord) error {
	changes := globalState.Changes{}

	switch {
	// new policy added
	case oldPolicy == nil && newPolicy != nil:
		if len(newPolicy.Meta.CoreRequirements) > 0 {
			changes.CoreRequirements = true
		}
	// policy removed
	case oldPolicy != nil && newPolicy == nil:
		changes.IsRemoval = true

		if len(oldPolicy.Meta.CoreRequirements) > 0 {
			changes.CoreRequirements = true
		}

	// policy changed
	default:
		if !oldPolicy.Meta.CoreRequirements.Equals(newPolicy.Meta.CoreRequirements) {
			changes.CoreRequirements = true
		}
	}

	oldDiags, newDiags := 0, 0
	if oldPolicy != nil {
		oldDiags = oldPolicy.PolicyDiagnostics.Count()
	}
	if newPolicy != nil {
		newDiags = newPolicy.PolicyDiagnostics.Count()
	}
	// Comparing diagnostics accurately could be expensive
	// so we just treat any non-empty diags as a change
	if oldDiags > 0 || newDiags > 0 {
		changes.Diagnostics = true
	}

	var policyHandle document.DirHandle
	if oldPolicy != nil {
		policyHandle = document.DirHandleFromPath(oldPolicy.Path())
	} else {
		policyHandle = document.DirHandleFromPath(newPolicy.Path())
	}

	return s.changeStore.QueueChange(policyHandle, changes)
}

func (f *PolicyStore) MetadataReady(dir document.DirHandle) (<-chan struct{}, bool, error) {
	rTxn := f.db.Txn(false)

	wCh, recordObj, err := rTxn.FirstWatch(f.tableName, "policy_state", dir.Path(), op.OpStateLoaded)
	if err != nil {
		return nil, false, err
	}
	if recordObj != nil {
		return wCh, true, nil
	}

	return wCh, false, nil
}
