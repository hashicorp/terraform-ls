package state

import "github.com/hashicorp/go-memdb"

func (d *DiagnosticsStore) Add(modPath string) error {
	txn := d.db.Txn(true)
	defer txn.Abort()

	err := d.add(txn, modPath)
	if err != nil {
		return err
	}
	txn.Commit()

	return nil
}

func (d *DiagnosticsStore) add(txn *memdb.Txn, modPath string) error {
	obj, err := txn.First(d.tableName, "id", modPath)
	if err != nil {
		return err
	}
	if obj != nil {
		return &AlreadyExistsError{
			Idx: modPath,
		}
	}

	mod := newModule(modPath)
	err = txn.Insert(d.tableName, mod)
	if err != nil {
		return err
	}

	// err = d.queueModuleChange(txn, nil, mod)
	// if err != nil {
	// 	return err
	// }

	return nil
}
