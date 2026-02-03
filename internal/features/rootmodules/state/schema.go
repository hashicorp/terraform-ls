// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"io"
	"log"

	"github.com/hashicorp/go-memdb"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
)

const rootTableName = "root"

var dbSchema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		rootTableName: {
			Name: rootTableName,
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

func NewRootStore(changeStore *globalState.ChangeStore, providerSchemaStore *globalState.ProviderSchemaStore) (*RootStore, error) {
	db, err := memdb.NewMemDB(dbSchema)
	if err != nil {
		return nil, err
	}

	discardLogger := log.New(io.Discard, "", 0)

	return &RootStore{
		db:                  db,
		tableName:           rootTableName,
		logger:              discardLogger,
		changeStore:         changeStore,
		providerSchemaStore: providerSchemaStore,
	}, nil
}
