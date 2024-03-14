// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	"github.com/hashicorp/terraform-schema/registry"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

const (
	documentsTableName      = "documents"
	jobsTableName           = "jobs"
	moduleTableName         = "module"
	variableTableName       = "variable"
	rootTableName           = "root"
	moduleIdsTableName      = "module_ids"
	moduleChangesTableName  = "module_changes"
	providerSchemaTableName = "provider_schema"
	providerIdsTableName    = "provider_ids"
	walkerPathsTableName    = "walker_paths"
	registryModuleTableName = "registry_module"

	tracerName = "github.com/hashicorp/terraform-ls/internal/state"
)

var dbSchema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
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
		moduleTableName: {
			Name: moduleTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "path"},
				},
			},
		},
		variableTableName: {
			Name: variableTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "path"},
				},
			},
		},
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
		moduleChangesTableName: {
			Name: moduleChangesTableName,
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
	DocumentStore   *DocumentStore
	JobStore        *JobStore
	Modules         *ModuleStore
	Variables       *VariableStore
	Roots           *RootStore
	ProviderSchemas *ProviderSchemaStore
	WalkerPaths     *WalkerPathStore
	RegistryModules *RegistryModuleStore

	db *memdb.MemDB
}

type ModuleStore struct {
	db        *memdb.MemDB
	Changes   *ModuleChangeStore
	tableName string
	logger    *log.Logger

	// TimeProvider provides current time (for mocking time.Now in tests)
	TimeProvider func() time.Time

	// MaxModuleNesting represents how many nesting levels we'd attempt
	// to parse provider requirements before returning error.
	MaxModuleNesting int
}

type VariableStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger
}

type RootStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger
}

type ModuleChangeStore struct {
	db *memdb.MemDB
}

type ModuleReader interface {
	CallersOfModule(modPath string) ([]*ModuleRecord, error)
	ModuleByPath(modPath string) (*ModuleRecord, error)
	List() ([]*ModuleRecord, error)
}

type ModuleCallReader interface {
	ModuleCalls(modPath string) (tfmod.ModuleCalls, error)
	LocalModuleMeta(modPath string) (*tfmod.Meta, error)
	RegistryModuleMeta(addr tfaddr.Module, cons version.Constraints) (*registry.ModuleData, error)
}

type StateReader interface {
	DeclaredModuleCalls(modPath string) (map[string]tfmod.DeclaredModuleCall, error)
	InstalledModuleCalls(modPath string) (map[string]tfmod.InstalledModuleCall, error)
	LocalModuleMeta(modPath string) (*tfmod.Meta, error)
	RegistryModuleMeta(addr tfaddr.Module, cons version.Constraints) (*registry.ModuleData, error)
	ProviderSchema(modPath string, addr tfaddr.Provider, vc version.Constraints) (*tfschema.ProviderSchema, error)
	TerraformVersion(modPath string) *version.Version

	ModuleRecordByPath(modPath string) (*ModuleRecord, error)
	VariableRecordByPath(modPath string) (*VariableRecord, error)

	ListModuleRecords() ([]*ModuleRecord, error)
	ListVariableRecords() ([]*VariableRecord, error)
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

type SchemaReader interface {
	ProviderSchema(modPath string, addr tfaddr.Provider, vc version.Constraints) (*tfschema.ProviderSchema, error)
}

func NewStateStore() (*StateStore, error) {
	db, err := memdb.NewMemDB(dbSchema)
	if err != nil {
		return nil, err
	}

	return &StateStore{
		db: db,
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
		Modules: &ModuleStore{
			db:               db,
			tableName:        moduleTableName,
			logger:           defaultLogger,
			TimeProvider:     time.Now,
			MaxModuleNesting: 50,
		},
		Variables: &VariableStore{
			db:        db,
			tableName: variableTableName,
			logger:    defaultLogger,
		},
		Roots: &RootStore{
			db:        db,
			tableName: rootTableName,
			logger:    defaultLogger,
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
	s.DocumentStore.logger = logger
	s.JobStore.logger = logger
	s.Modules.logger = logger
	s.Variables.logger = logger
	s.Roots.logger = logger
	s.ProviderSchemas.logger = logger
	s.WalkerPaths.logger = logger
	s.RegistryModules.logger = logger
}

var defaultLogger = log.New(ioutil.Discard, "", 0)

type RecordStores struct {
	Modules         *ModuleStore
	ProviderSchemas *ProviderSchemaStore
	RegistryModules *RegistryModuleStore
	Roots           *RootStore
	Variables       *VariableStore
}

var (
	_ StateReader = &RecordStores{}
)

type RecordStore interface {
	Path() string
}

func NewRecordStores(modules *ModuleStore, roots *RootStore, variables *VariableStore,
	registryModules *RegistryModuleStore, providerSchemas *ProviderSchemaStore) *RecordStores {
	return &RecordStores{
		Modules:         modules,
		ProviderSchemas: providerSchemas,
		RegistryModules: registryModules,
		Roots:           roots,
		Variables:       variables,
	}
}

func (ds *RecordStores) ByPath(path string, recordType ast.RecordType) (RecordStore, error) {
	if recordType == ast.RecordTypeModule {
		return ds.Modules.ModuleByPath(path)
	}
	if recordType == ast.RecordTypeVariable {
		return ds.Variables.VariableRecordByPath(path)
	}
	if recordType == ast.RecordTypeRoot {
		return ds.Roots.RootRecordByPath(path)
	}

	return nil, fmt.Errorf("unknown record type: %q", recordType)
}

func (ds *RecordStores) Add(path string, recordType ast.RecordType) error {
	if recordType == ast.RecordTypeModule {
		return ds.Modules.Add(path)
	}
	if recordType == ast.RecordTypeVariable {
		return ds.Variables.Add(path)
	}
	if recordType == ast.RecordTypeRoot {
		return ds.Roots.Add(path)
	}

	return fmt.Errorf("unknown record type: %q", recordType)
}

func (ds *RecordStores) AddIfNotExists(path string, recordType ast.RecordType) error {
	if recordType == ast.RecordTypeModule {
		return ds.Modules.AddIfNotExists(path)
	}
	if recordType == ast.RecordTypeVariable {
		return ds.Variables.AddIfNotExists(path)
	}
	if recordType == ast.RecordTypeRoot {
		return ds.Roots.AddIfNotExists(path)
	}

	return fmt.Errorf("unknown record type: %q", recordType)
}

func (ds *RecordStores) Remove(path string) error {
	var errs *multierror.Error

	err := ds.Modules.Remove(path)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	err = ds.Variables.Remove(path)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	err = ds.Roots.Remove(path)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	return errs.ErrorOrNil()

}

func (ds *RecordStores) DeclaredModuleCalls(modPath string) (map[string]tfmod.DeclaredModuleCall, error) {
	return ds.Modules.DeclaredModuleCalls(modPath)
}

func (ds *RecordStores) InstalledModuleCalls(modPath string) (map[string]tfmod.InstalledModuleCall, error) {
	return ds.Roots.InstalledModuleCalls(modPath)
}

func (ds *RecordStores) LocalModuleMeta(modPath string) (*tfmod.Meta, error) {
	return ds.Modules.LocalModuleMeta(modPath)
}

func (ds *RecordStores) RegistryModuleMeta(addr tfaddr.Module, cons version.Constraints) (*registry.ModuleData, error) {
	return ds.RegistryModules.RegistryModuleMeta(addr, cons)
}

func (ds *RecordStores) ProviderSchema(modPath string, addr tfaddr.Provider, vc version.Constraints) (*tfschema.ProviderSchema, error) {
	return ds.ProviderSchemas.ProviderSchema(modPath, addr, vc)
}

func (ds *RecordStores) TerraformVersion(modPath string) *version.Version {
	// TODO! some path magic
	root, err := ds.Roots.RootRecordByPath(modPath)
	if err != nil {
		return nil
	}

	return root.TerraformVersion
}

func (ds *RecordStores) ModuleRecordByPath(modPath string) (*ModuleRecord, error) {
	return ds.Modules.ModuleByPath(modPath)
}

func (ds *RecordStores) VariableRecordByPath(modPath string) (*VariableRecord, error) {
	return ds.Variables.VariableRecordByPath(modPath)
}

func (ds *RecordStores) ListModuleRecords() ([]*ModuleRecord, error) {
	return ds.Modules.List()
}

func (ds *RecordStores) ListVariableRecords() ([]*VariableRecord, error) {
	return ds.Variables.List()
}
