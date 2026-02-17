// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import "github.com/hashicorp/go-uuid"

type PolicyTestIds struct {
	Path string
	ID   string
}

func (s *PolicyTestStore) GetPolicyTestID(path string) (string, error) {
	txn := s.db.Txn(true)
	defer txn.Abort()

	obj, err := txn.First(policytestIdsTableName, "id", path)
	if err != nil {
		return "", err
	}

	if obj != nil {
		return obj.(PolicyTestIds).ID, nil
	}

	newId, err := uuid.GenerateUUID()
	if err != nil {
		return "", err
	}

	err = txn.Insert(policytestIdsTableName, PolicyTestIds{
		ID:   newId,
		Path: path,
	})
	if err != nil {
		return "", err
	}

	txn.Commit()
	return newId, nil
}
