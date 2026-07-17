// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"io"
	"log"

	"github.com/hashicorp/go-memdb"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
)

const (
	policytestTableName    = "policytest"
	policytestIdsTableName = "policytest_ids"
)

var dbSchema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		policytestTableName: {
			Name: policytestTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "path"},
				},
				"policytest_state": {
					Name: "policytest_state",
					Indexer: &memdb.CompoundIndex{
						Indexes: []memdb.Indexer{
							&memdb.StringFieldIndex{Field: "path"},
							&memdb.UintFieldIndex{Field: "MetaState"},
						},
					},
				},
			},
		},
		policytestIdsTableName: {
			Name: policytestIdsTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "Path"},
				},
			},
		},
	},
}

func NewPolicyTestStore(changeStore *globalState.ChangeStore) (*PolicyTestStore, error) {
	db, err := memdb.NewMemDB(dbSchema)
	if err != nil {
		return nil, err
	}

	discardLogger := log.New(io.Discard, "", 0)

	return &PolicyTestStore{
		db:                   db,
		tableName:            policytestTableName,
		logger:               discardLogger,
		MaxPolicyTestNesting: 50,
		changeStore:          changeStore,
	}, nil
}
