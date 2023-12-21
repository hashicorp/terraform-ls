// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

type Vars struct {
	path string

	VarsRefOrigins      reference.Origins
	VarsRefOriginsErr   error
	VarsRefOriginsState op.OpState

	ParsedVarsFiles ast.VarsFiles
	VarsParsingErr  error

	VarsDiagnostics      ast.SourceVarsDiags
	VarsDiagnosticsState ast.DiagnosticSourceState
}

func (v *Vars) Path() string {
	return v.path
}

func (v *Vars) Copy() *Vars {
	if v == nil {
		return nil
	}
	newVar := &Vars{
		path: v.path,

		VarsRefOrigins:      v.VarsRefOrigins.Copy(),
		VarsRefOriginsErr:   v.VarsRefOriginsErr,
		VarsRefOriginsState: v.VarsRefOriginsState,

		VarsParsingErr: v.VarsParsingErr,

		VarsDiagnosticsState: v.VarsDiagnosticsState.Copy(),
	}

	if v.ParsedVarsFiles != nil {
		newVar.ParsedVarsFiles = make(ast.VarsFiles, len(v.ParsedVarsFiles))
		for name, f := range v.ParsedVarsFiles {
			// hcl.File is practically immutable once it comes out of parser
			newVar.ParsedVarsFiles[name] = f
		}
	}

	if v.VarsDiagnostics != nil {
		newVar.VarsDiagnostics = make(ast.SourceVarsDiags, len(v.VarsDiagnostics))

		for source, varsDiags := range v.VarsDiagnostics {
			newVar.VarsDiagnostics[source] = make(ast.VarsDiags, len(varsDiags))

			for name, diags := range varsDiags {
				newVar.VarsDiagnostics[source][name] = make(hcl.Diagnostics, len(diags))
				copy(newVar.VarsDiagnostics[source][name], diags)
			}
		}
	}

	return newVar
}

func newVars(modPath string) *Vars {
	return &Vars{
		path: modPath,
		VarsDiagnosticsState: ast.DiagnosticSourceState{
			ast.HCLParsingSource:          op.OpStateUnknown,
			ast.SchemaValidationSource:    op.OpStateUnknown,
			ast.ReferenceValidationSource: op.OpStateUnknown,
			ast.TerraformValidateSource:   op.OpStateUnknown,
		},
	}
}

func varsByPath(txn *memdb.Txn, path string) (*Vars, error) {
	obj, err := txn.First(varsTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &VarsNotFoundError{
			Source: path,
		}
	}
	return obj.(*Vars), nil
}

func varsCopyByPath(txn *memdb.Txn, path string) (*Vars, error) {
	vars, err := varsByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return vars.Copy(), nil
}

func (v *VarsStore) Add(modPath string) error {
	txn := v.db.Txn(true)
	defer txn.Abort()

	err := v.add(txn, modPath)
	if err != nil {
		return err
	}
	txn.Commit()

	return nil
}

func (v *VarsStore) add(txn *memdb.Txn, modPath string) error {
	// TODO: Introduce Exists method to Txn?
	obj, err := txn.First(v.tableName, "id", modPath)
	if err != nil {
		return err
	}
	if obj != nil {
		return &AlreadyExistsError{
			Idx: modPath,
		}
	}

	mod := newVars(modPath)
	err = txn.Insert(v.tableName, mod)
	if err != nil {
		return err
	}

	// TODO queue vars changes

	return nil
}

func (v *VarsStore) List() ([]*Vars, error) {
	txn := v.db.Txn(false)

	it, err := txn.Get(v.tableName, "id")
	if err != nil {
		return nil, err
	}

	vars := make([]*Vars, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		mod := item.(*Vars)
		vars = append(vars, mod)
	}

	return vars, nil
}

func (v *VarsStore) VarsByPath(path string) (*Vars, error) {
	txn := v.db.Txn(false)

	vars, err := varsByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return vars, nil
}

func (v *VarsStore) AddIfNotExists(path string) error {
	txn := v.db.Txn(true)
	defer txn.Abort()

	_, err := varsByPath(txn, path)
	if err != nil {
		if IsVarsNotFound(err) {
			err := v.add(txn, path)
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

func (v *VarsStore) UpdateParsedVarsFiles(path string, vFiles ast.VarsFiles, vErr error) error {
	txn := v.db.Txn(true)
	defer txn.Abort()

	vars, err := varsCopyByPath(txn, path)
	if err != nil {
		return err
	}

	vars.ParsedVarsFiles = vFiles

	vars.VarsParsingErr = vErr

	err = txn.Insert(v.tableName, vars)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (v *VarsStore) UpdateVarsDiagnostics(path string, source ast.DiagnosticSource, diags ast.VarsDiags) error {
	txn := v.db.Txn(true)
	txn.Defer(func() {
		v.SetVarsDiagnosticsState(path, source, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldVars, err := varsByPath(txn, path)
	if err != nil {
		return err
	}

	vars := oldVars.Copy()
	if vars.VarsDiagnostics == nil {
		vars.VarsDiagnostics = make(ast.SourceVarsDiags)
	}
	vars.VarsDiagnostics[source] = diags

	err = txn.Insert(v.tableName, vars)
	if err != nil {
		return err
	}

	// TODO queue vars changes

	txn.Commit()
	return nil
}

func (v *VarsStore) SetVarsDiagnosticsState(path string, source ast.DiagnosticSource, state op.OpState) error {
	txn := v.db.Txn(true)
	defer txn.Abort()

	vars, err := varsCopyByPath(txn, path)
	if err != nil {
		return err
	}
	vars.VarsDiagnosticsState[source] = state

	err = txn.Insert(v.tableName, vars)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (v *VarsStore) SetVarsReferenceOriginsState(path string, state op.OpState) error {
	txn := v.db.Txn(true)
	defer txn.Abort()

	vars, err := varsCopyByPath(txn, path)
	if err != nil {
		return err
	}

	vars.VarsRefOriginsState = state
	err = txn.Insert(v.tableName, vars)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (v *VarsStore) UpdateVarsReferenceOrigins(path string, origins reference.Origins, roErr error) error {
	txn := v.db.Txn(true)
	txn.Defer(func() {
		v.SetVarsReferenceOriginsState(path, op.OpStateLoaded)
	})
	defer txn.Abort()

	vars, err := varsCopyByPath(txn, path)
	if err != nil {
		return err
	}

	vars.VarsRefOrigins = origins
	vars.VarsRefOriginsErr = roErr

	err = txn.Insert(v.tableName, vars)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}
