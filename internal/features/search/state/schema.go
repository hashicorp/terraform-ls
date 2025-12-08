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
	searchTableName = "search"
)

var dbSchema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		searchTableName: {
			Name: searchTableName,
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

func NewSearchStore(changeStore *globalState.ChangeStore, providerSchemasStore *globalState.ProviderSchemaStore) (*SearchStore, error) {
	db, err := memdb.NewMemDB(dbSchema)
	if err != nil {
		return nil, err
	}

	discardLogger := log.New(io.Discard, "", 0)

	return &SearchStore{
		db:                   db,
		tableName:            searchTableName,
		logger:               discardLogger,
		changeStore:          changeStore,
		providerSchemasStore: providerSchemasStore,
	}, nil
}
