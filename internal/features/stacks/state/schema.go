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
	stackTableName = "stacks"
)

var dbSchema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		stackTableName: {
			Name: stackTableName,
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

func NewStackStore(changeStore *globalState.ChangeStore, providerSchemasStore *globalState.ProviderSchemaStore) (*StackStore, error) {
	db, err := memdb.NewMemDB(dbSchema)
	if err != nil {
		return nil, err
	}

	discardLogger := log.New(io.Discard, "", 0)

	return &StackStore{
		db:                   db,
		tableName:            stackTableName,
		logger:               discardLogger,
		changeStore:          changeStore,
		providerSchemasStore: providerSchemasStore,
	}, nil
}
