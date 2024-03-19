package state

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-version"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// TerraformVersionRecord tracks the installed Terraform version. It can be
// extended to track multiple Terraform versions in the future.
type TerraformVersionRecord struct {
	path string

	TerraformVersion      *version.Version
	TerraformVersionErr   error
	TerraformVersionState op.OpState
}

func (m *TerraformVersionRecord) Copy() *TerraformVersionRecord {
	if m == nil {
		return nil
	}
	newMod := &TerraformVersionRecord{
		path: m.path,

		// version.Version is practically immutable once parsed
		TerraformVersion:      m.TerraformVersion,
		TerraformVersionErr:   m.TerraformVersionErr,
		TerraformVersionState: m.TerraformVersionState,
	}

	return newMod
}

func newTerraformVersionRecord(path string) *TerraformVersionRecord {
	return &TerraformVersionRecord{
		path:                  path,
		TerraformVersionState: op.OpStateUnknown,
	}
}

func (s *TerraformVersionStore) Add(path string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	err := s.add(txn, path)
	if err != nil {
		return err
	}
	txn.Commit()

	return nil
}

func (s *TerraformVersionStore) add(txn *memdb.Txn, path string) error {
	obj, err := txn.First(s.tableName, "id", path)
	if err != nil {
		return err
	}
	if obj != nil {
		return &AlreadyExistsError{
			Idx: path,
		}
	}

	record := newTerraformVersionRecord(path)
	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	return nil
}

func (s *TerraformVersionStore) AddIfNotExists() error {
	path := "global" // We only track a single Terraform version for now

	txn := s.db.Txn(true)
	defer txn.Abort()

	_, err := terraformVersionRecordByPath(txn, path)
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

func (s *TerraformVersionStore) TerraformVersionRecord() (*TerraformVersionRecord, error) {
	path := "global" // We only track a single Terraform version for now

	txn := s.db.Txn(false)

	record, err := terraformVersionRecordByPath(txn, path)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func terraformVersionRecordByPath(txn *memdb.Txn, path string) (*TerraformVersionRecord, error) {
	obj, err := txn.First(terraformVersionTableName, "id", path)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &RecordNotFoundError{
			Source: path,
		}
	}
	return obj.(*TerraformVersionRecord), nil
}

func (s *TerraformVersionStore) SetTerraformVersionState(state op.OpState) error {
	path := "global" // We only track a single Terraform version for now

	txn := s.db.Txn(true)
	defer txn.Abort()

	oldRecord, err := terraformVersionRecordByPath(txn, path)
	if err != nil {
		return err
	}

	record := oldRecord.Copy()
	record.TerraformVersionState = state

	err = txn.Insert(s.tableName, record)
	if err != nil {
		return err
	}

	// TODO! queue module change
	// err = s.queueModuleChange(txn, nil, mod)
	// if err != nil {
	// 	return err
	// }

	txn.Commit()
	return nil
}

func (s *TerraformVersionStore) UpdateTerraformVersion(tfVer *version.Version, vErr error) error {
	path := "global" // We only track a single Terraform version for now
	s.logger.Printf("TVS: storing installed Terraform version: %s, Err: %w", tfVer, vErr)

	txn := s.db.Txn(true)
	txn.Defer(func() {
		s.SetTerraformVersionState(op.OpStateLoaded)
	})
	defer txn.Abort()

	oldRecord, err := terraformVersionRecordByPath(txn, path)
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

	// TODO! queue module change
	// err = s.queueModuleChange(txn, oldMod, mod)
	// if err != nil {
	// 	return err
	// }

	txn.Commit()
	return nil
}
