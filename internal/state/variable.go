package state

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// VariableRecord contains all information about variable definition files
// we have for a certain path
type VariableRecord struct {
	path string

	VarsRefOrigins      reference.Origins
	VarsRefOriginsErr   error
	VarsRefOriginsState op.OpState

	ParsedVarsFiles ast.VarsFiles
	VarsParsingErr  error

	VarsDiagnostics      ast.SourceVarsDiags
	VarsDiagnosticsState ast.DiagnosticSourceState
}

func (v *VariableRecord) Copy() *VariableRecord {
	if v == nil {
		return nil
	}
	newMod := &VariableRecord{
		path: v.path,

		VarsRefOrigins:      v.VarsRefOrigins.Copy(),
		VarsRefOriginsErr:   v.VarsRefOriginsErr,
		VarsRefOriginsState: v.VarsRefOriginsState,

		VarsParsingErr: v.VarsParsingErr,

		VarsDiagnosticsState: v.VarsDiagnosticsState.Copy(),
	}

	if v.ParsedVarsFiles != nil {
		newMod.ParsedVarsFiles = make(ast.VarsFiles, len(v.ParsedVarsFiles))
		for name, f := range v.ParsedVarsFiles {
			// hcl.File is practically immutable once it comes out of parser
			newMod.ParsedVarsFiles[name] = f
		}
	}

	if v.VarsDiagnostics != nil {
		newMod.VarsDiagnostics = make(ast.SourceVarsDiags, len(v.VarsDiagnostics))

		for source, varsDiags := range v.VarsDiagnostics {
			newMod.VarsDiagnostics[source] = make(ast.VarsDiags, len(varsDiags))

			for name, diags := range varsDiags {
				newMod.VarsDiagnostics[source][name] = make(hcl.Diagnostics, len(diags))
				copy(newMod.VarsDiagnostics[source][name], diags)
			}
		}
	}

	return newMod
}

func (v *VariableRecord) Path() string {
	return v.path
}

func newVariableRecord(modPath string) *VariableRecord {
	return &VariableRecord{
		path: modPath,
		VarsDiagnosticsState: ast.DiagnosticSourceState{
			ast.HCLParsingSource:          op.OpStateUnknown,
			ast.SchemaValidationSource:    op.OpStateUnknown,
			ast.ReferenceValidationSource: op.OpStateUnknown,
			ast.TerraformValidateSource:   op.OpStateUnknown,
		},
	}
}

// NewVariableRecordTest is a test helper to create a new VariableRecord
func NewVariableRecordTest(path string) *VariableRecord {
	return &VariableRecord{
		path: path,
	}
}

func (s *VariableStore) Add(modPath string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	err := s.add(txn, modPath)
	if err != nil {
		return err
	}
	txn.Commit()

	return nil
}

func (s *VariableStore) add(txn *memdb.Txn, modPath string) error {
	// TODO: Introduce Exists method to Txn?
	obj, err := txn.First(s.tableName, "id", modPath)
	if err != nil {
		return err
	}
	if obj != nil {
		return &AlreadyExistsError{
			Idx: modPath,
		}
	}

	mod := newVariableRecord(modPath)
	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	// TODO! queue change
	// err = s.queueModuleChange(txn, nil, mod)
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (s *VariableStore) AddIfNotExists(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	_, err := variableRecordByPath(txn, path)
	if err != nil {
		if IsRecordNotFound(err) {
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

func (s *VariableStore) Remove(modPath string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	oldObj, err := txn.First(s.tableName, "id", modPath)
	if err != nil {
		return err
	}

	if oldObj == nil {
		// already removed
		return nil
	}

	// TODO! queue change
	// oldMod := oldObj.(*VariableRecord)
	// err = s.queueModuleChange(txn, oldMod, nil)
	// if err != nil {
	// 	return err
	// }

	_, err = txn.DeleteAll(s.tableName, "id", modPath)
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

	mod, err := variableRecordByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return mod, nil
}

func variableRecordByPath(txn *memdb.Txn, path string) (*VariableRecord, error) {
	obj, err := txn.First(variableTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &RecordNotFoundError{
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

	mod, err := variableRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.ParsedVarsFiles = vFiles

	mod.VarsParsingErr = vErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *VariableStore) UpdateVarsDiagnostics(path string, source ast.DiagnosticSource, diags ast.VarsDiags) error {
	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetVarsDiagnosticsState(path, source, op.OpStateLoaded)
	})
	defer txn.Abort()

	oldMod, err := variableRecordByPath(txn, path)
	if err != nil {
		return err
	}

	mod := oldMod.Copy()
	if mod.VarsDiagnostics == nil {
		mod.VarsDiagnostics = make(ast.SourceVarsDiags)
	}
	mod.VarsDiagnostics[source] = diags

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	// TODO! queue change
	// err = s.queueModuleChange(txn, oldMod, mod)
	// if err != nil {
	// 	return err
	// }

	txn.Commit()
	return nil
}

func (s *VariableStore) SetVarsDiagnosticsState(path string, source ast.DiagnosticSource, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := variableRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}
	mod.VarsDiagnosticsState[source] = state

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *VariableStore) SetVarsReferenceOriginsState(path string, state op.OpState) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	mod, err := variableRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.VarsRefOriginsState = state
	err = txn.Insert(s.tableName, mod)
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

	mod, err := variableRecordCopyByPath(txn, path)
	if err != nil {
		return err
	}

	mod.VarsRefOrigins = origins
	mod.VarsRefOriginsErr = roErr

	err = txn.Insert(s.tableName, mod)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}
