package state

import (
	"io/ioutil"
	"log"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

const (
	moduleTableName         = "module"
	moduleIdsTableName      = "module_ids"
	providerSchemaTableName = "provider_schema"
	providerIdsTableName    = "provider_ids"
)

var dbSchema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		moduleTableName: {
			Name: moduleTableName,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "Path"},
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
	},
}

type StateStore struct {
	Modules         *ModuleStore
	ProviderSchemas *ProviderSchemaStore

	db *memdb.MemDB
}

type ModuleStore struct {
	db          *memdb.MemDB
	ChangeHooks ModuleChangeHooks
	tableName   string
	logger      *log.Logger
}

type ModuleReader interface {
	CallersOfModule(modPath string) ([]*Module, error)
	ModuleByPath(modPath string) (*Module, error)
	List() ([]*Module, error)
}

type ModuleCallReader interface {
	ModuleCalls(modPath string) ([]tfmod.ModuleCall, error)
	ModuleMeta(modPath string) (*tfmod.Meta, error)
}

type ProviderSchemaStore struct {
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
		Modules: &ModuleStore{
			db:          db,
			ChangeHooks: make(ModuleChangeHooks, 0),
			tableName:   moduleTableName,
			logger:      defaultLogger,
		},
		ProviderSchemas: &ProviderSchemaStore{
			db:        db,
			tableName: providerSchemaTableName,
			logger:    defaultLogger,
		},
	}, nil
}

func (s *StateStore) SetLogger(logger *log.Logger) {
	s.Modules.logger = logger
	s.ProviderSchemas.logger = logger
}

var defaultLogger = log.New(ioutil.Discard, "", 0)
