// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"io"
	"log"

	"github.com/hashicorp/go-memdb"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
)

const (
	testsTableName = "tests"
)

var dbSchema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		testsTableName: {
			Name: testsTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "path"},
				},
			},
		},
	},
}

func NewTestStore(changeStore *globalState.ChangeStore, providerSchemasStore *globalState.ProviderSchemaStore) (*TestStore, error) {
	db, err := memdb.NewMemDB(dbSchema)
	if err != nil {
		return nil, err
	}

	discardLogger := log.New(io.Discard, "", 0)

	return &TestStore{
		db:                   db,
		tableName:            testsTableName,
		logger:               discardLogger,
		changeStore:          changeStore,
		providerSchemasStore: providerSchemasStore,
	}, nil
}
