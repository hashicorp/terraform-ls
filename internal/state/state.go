// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"io"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/go-memdb"
)

const (
	changesTableName        = "changes"
	documentsTableName      = "documents"
	jobsTableName           = "jobs"
	providerSchemaTableName = "provider_schema"
	providerIdsTableName    = "provider_ids"
	walkerPathsTableName    = "walker_paths"
	registryModuleTableName = "registry_module"

	tracerName = "github.com/hashicorp/terraform-ls/internal/state"
)

var dbSchema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		changesTableName: {
			Name: changesTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &DirHandleFieldIndexer{Field: "DirHandle"},
				},
				"time": {
					Name:    "time",
					Indexer: &TimeFieldIndex{Field: "FirstChangeTime"},
				},
			},
		},
		documentsTableName: {
			Name: documentsTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:   "id",
					Unique: true,
					Indexer: &memdb.CompoundIndex{
						Indexes: []memdb.Indexer{
							&DirHandleFieldIndexer{Field: "Dir"},
							&memdb.StringFieldIndex{Field: "Filename"},
						},
					},
				},
				"dir": {
					Name:    "dir",
					Indexer: &DirHandleFieldIndexer{Field: "Dir"},
				},
			},
		},
		jobsTableName: {
			Name: jobsTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &StringerFieldIndexer{Field: "ID"},
				},
				"priority_dependecies_state": {
					Name: "priority_dependecies_state",
					Indexer: &memdb.CompoundIndex{
						Indexes: []memdb.Indexer{
							&JobPriorityIndex{
								PriorityIntField:   "Priority",
								IsDirOpenBoolField: "IsDirOpen",
							},
							&SliceLengthIndex{Field: "DependsOn"},
							&memdb.UintFieldIndex{Field: "State"},
						},
					},
				},
				"dir_state": {
					Name: "dir_state",
					Indexer: &memdb.CompoundIndex{
						Indexes: []memdb.Indexer{
							&DirHandleFieldIndexer{Field: "Dir"},
							&memdb.UintFieldIndex{Field: "State"},
						},
					},
				},
				"dir_state_type": {
					Name: "dir_state_type",
					Indexer: &memdb.CompoundIndex{
						Indexes: []memdb.Indexer{
							&DirHandleFieldIndexer{Field: "Dir"},
							&memdb.UintFieldIndex{Field: "State"},
							&memdb.StringFieldIndex{Field: "Type"},
						},
					},
				},
				"state_type": {
					Name: "state_type",
					Indexer: &memdb.CompoundIndex{
						Indexes: []memdb.Indexer{
							&memdb.UintFieldIndex{Field: "State"},
							&memdb.StringFieldIndex{Field: "Type"},
						},
					},
				},
				"state": {
					Name: "state",
					Indexer: &memdb.CompoundIndex{
						Indexes: []memdb.Indexer{
							&memdb.UintFieldIndex{Field: "State"},
						},
					},
				},
				"depends_on": {
					Name: "depends_on",
					Indexer: &JobIdSliceIndex{
						Field: "DependsOn",
					},
					AllowMissing: true,
				},
			},
		},
		providerSchemaTableName: {
			Name: providerSchemaTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:   "id",
					Unique: true,
					Indexer: &memdb.CompoundIndex{
						Indexes: []memdb.Indexer{
							&StringerFieldIndexer{Field: "Address"},
							&StringerFieldIndexer{Field: "Source"},
							&VersionFieldIndexer{Field: "Version"},
						},
						AllowMissing: true,
					},
				},
			},
		},
		registryModuleTableName: {
			Name: registryModuleTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:   "id",
					Unique: true,
					Indexer: &memdb.CompoundIndex{
						Indexes: []memdb.Indexer{
							&StringerFieldIndexer{Field: "Source"},
							&VersionFieldIndexer{Field: "Version"},
						},
						AllowMissing: true,
					},
				},
				"source_addr": {
					Name:    "source_addr",
					Indexer: &StringerFieldIndexer{Field: "Source"},
				},
			},
		},
		providerIdsTableName: {
			Name: providerIdsTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "Address"},
				},
			},
		},
		walkerPathsTableName: {
			Name: walkerPathsTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &DirHandleFieldIndexer{Field: "Dir"},
				},
				"is_dir_open_state": {
					Name: "is_dir_open_state",
					Indexer: &memdb.CompoundIndex{
						Indexes: []memdb.Indexer{
							&memdb.BoolFieldIndex{Field: "IsDirOpen"},
							&memdb.UintFieldIndex{Field: "State"},
						},
					},
				},
			},
		},
	},
}

type StateStore struct {
	ChangeStore     *ChangeStore
	DocumentStore   *DocumentStore
	JobStore        *JobStore
	ProviderSchemas *ProviderSchemaStore
	WalkerPaths     *WalkerPathStore
	RegistryModules *RegistryModuleStore

	db *memdb.MemDB
}

type ChangeStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger

	// TimeProvider provides current time (for mocking time.Now in tests)
	TimeProvider func() time.Time
}
type ProviderSchemaStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger
}
type RegistryModuleStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger
}

func NewStateStore() (*StateStore, error) {
	db, err := memdb.NewMemDB(dbSchema)
	if err != nil {
		return nil, err
	}

	return &StateStore{
		db: db,
		ChangeStore: &ChangeStore{
			db:           db,
			tableName:    changesTableName,
			logger:       defaultLogger,
			TimeProvider: time.Now,
		},
		DocumentStore: &DocumentStore{
			db:           db,
			tableName:    documentsTableName,
			logger:       defaultLogger,
			TimeProvider: time.Now,
		},
		JobStore: &JobStore{
			db:                db,
			tableName:         jobsTableName,
			logger:            defaultLogger,
			nextJobHighPrioMu: &sync.Mutex{},
			nextJobLowPrioMu:  &sync.Mutex{},
		},
		ProviderSchemas: &ProviderSchemaStore{
			db:        db,
			tableName: providerSchemaTableName,
			logger:    defaultLogger,
		},
		RegistryModules: &RegistryModuleStore{
			db:        db,
			tableName: registryModuleTableName,
			logger:    defaultLogger,
		},
		WalkerPaths: &WalkerPathStore{
			db:              db,
			tableName:       walkerPathsTableName,
			logger:          defaultLogger,
			nextOpenDirMu:   &sync.Mutex{},
			nextClosedDirMu: &sync.Mutex{},
		},
	}, nil
}

func (s *StateStore) SetLogger(logger *log.Logger) {
	s.ChangeStore.logger = logger
	s.DocumentStore.logger = logger
	s.JobStore.logger = logger
	s.ProviderSchemas.logger = logger
	s.WalkerPaths.logger = logger
	s.RegistryModules.logger = logger
}

var defaultLogger = log.New(io.Discard, "", 0)
