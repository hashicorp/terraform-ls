// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"io"
	"log"

	"github.com/hashicorp/go-memdb"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
)

const (
	moduleTableName    = "module"
	moduleIdsTableName = "module_ids"
)

var dbSchema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		moduleTableName: {
			Name: moduleTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "path"},
				},
				"module_state": {
					Name: "module_state",
					Indexer: &memdb.CompoundIndex{
						Indexes: []memdb.Indexer{
							&memdb.StringFieldIndex{Field: "path"},
							&memdb.UintFieldIndex{Field: "MetaState"},
						},
					},
				},
			},
		},
		moduleIdsTableName: {
			Name: moduleIdsTableName,
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

func NewModuleStore(providerSchemasStore *globalState.ProviderSchemaStore, registryModuleStore *globalState.RegistryModuleStore, changeStore *globalState.ChangeStore) (*ModuleStore, error) {
	db, err := memdb.NewMemDB(dbSchema)
	if err != nil {
		return nil, err
	}

	discardLogger := log.New(io.Discard, "", 0)

	return &ModuleStore{
		db:                   db,
		tableName:            moduleTableName,
		logger:               discardLogger,
		MaxModuleNesting:     50,
		changeStore:          changeStore,
		providerSchemasStore: providerSchemasStore,
		registryModuleStore:  registryModuleStore,
	}, nil
}
