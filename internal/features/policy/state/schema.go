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
	policyTableName    = "policy"
	policyIdsTableName = "policy_ids"
)

var dbSchema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		policyTableName: {
			Name: policyTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "path"},
				},
				"policy_state": {
					Name: "policy_state",
					Indexer: &memdb.CompoundIndex{
						Indexes: []memdb.Indexer{
							&memdb.StringFieldIndex{Field: "path"},
							&memdb.UintFieldIndex{Field: "MetaState"},
						},
					},
				},
			},
		},
		policyIdsTableName: {
			Name: policyIdsTableName,
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

func NewPolicyStore(changeStore *globalState.ChangeStore) (*PolicyStore, error) {
	db, err := memdb.NewMemDB(dbSchema)
	if err != nil {
		return nil, err
	}

	discardLogger := log.New(io.Discard, "", 0)

	return &PolicyStore{
		db:               db,
		tableName:        policyTableName,
		logger:           discardLogger,
		MaxPolicyNesting: 50,
		changeStore:      changeStore,
	}, nil
}
