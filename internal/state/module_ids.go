// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import "github.com/hashicorp/go-uuid"

type ModuleIds struct {
	Path string
	ID   string
}

func (s *StateStore) GetModuleID(path string) (string, error) {
	txn := s.db.Txn(true)
	defer txn.Abort()

	obj, err := txn.First(moduleIdsTableName, "id", path)
	if err != nil {
		return "", err
	}

	if obj != nil {
		return obj.(ModuleIds).ID, nil
	}

	newId, err := uuid.GenerateUUID()
	if err != nil {
		return "", err
	}

	err = txn.Insert(moduleIdsTableName, ModuleIds{
		ID:   newId,
		Path: path,
	})
	if err != nil {
		return "", err
	}

	txn.Commit()
	return newId, nil
}
