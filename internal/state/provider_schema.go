package state

import (
	"fmt"
	"sort"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-registry-address"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

type ProviderSchema struct {
	Address tfaddr.Provider
	Version *version.Version
	Source  SchemaSource

	Schema *tfschema.ProviderSchema
}

func (ps *ProviderSchema) Copy() *ProviderSchema {
	if ps == nil {
		return nil
	}

	return &ProviderSchema{
		Address: ps.Address,
		Version: ps.Version, // version.Version is immutable by design
		Source:  ps.Source,
		Schema:  ps.Schema.Copy(),
	}
}

type ProviderSchemaIterator struct {
	it memdb.ResultIterator
}

func (psi *ProviderSchemaIterator) Next() *ProviderSchema {
	item := psi.it.Next()
	if item == nil {
		return nil
	}
	return item.(*ProviderSchema)
}

func updateProviderVersions(txn *memdb.Txn, modPath string, pv map[tfaddr.Provider]*version.Version) error {
	for pAddr, pVer := range pv {
		// first check for existing record to avoid duplicates
		src := LocalSchemaSource{
			ModulePath: modPath,
		}

		obj, err := txn.First(providerSchemaTableName, "id_prefix", pAddr, src, pVer)
		if err != nil {
			return fmt.Errorf("unable to find provider schema: %w", err)
		}
		if obj != nil {
			// provider version already known for this path
			continue
		}

		// add version if schema is already present and version unknown
		obj, err = txn.First(providerSchemaTableName, "id_prefix", pAddr, src, nil)
		if err != nil {
			return fmt.Errorf("unable to find provider schema without version: %w", err)
		}
		if obj != nil {
			// TODO: Implement txn.Update?
			// See https://github.com/hashicorp/go-memdb/pull/49
			versionedPs := obj.(*ProviderSchema)

			if versionedPs.Schema != nil {
				_, err = txn.DeleteAll(providerSchemaTableName, "id_prefix", pAddr, src)
				if err != nil {
					return fmt.Errorf("unable to delete provider schema: %w", err)
				}

				psCopy := versionedPs.Copy()
				psCopy.Version = pVer
				psCopy.Schema.SetProviderVersion(psCopy.Address, pVer)

				err = txn.Insert(providerSchemaTableName, psCopy)
				if err != nil {
					return fmt.Errorf("unable to insert provider schema: %w", err)
				}
				continue
			}
		}

		// add just provider and version (no schema)
		ps := &ProviderSchema{
			Address: pAddr,
			Version: pVer,
			Source:  src,
		}
		err = txn.Insert(providerSchemaTableName, ps)
		if err != nil {
			return fmt.Errorf("unable to insert new provider schema: %w", err)
		}
	}

	return nil
}

func (s *ProviderSchemaStore) AddLocalSchema(modPath string, addr tfaddr.Provider, schema *tfschema.ProviderSchema) error {
	s.logger.Printf("PSS: adding local schema (%s, %s): %p", modPath, addr, schema)
	txn := s.db.Txn(true)
	defer txn.Abort()

	src := LocalSchemaSource{
		ModulePath: modPath,
	}

	// check for existing entries
	obj, err := txn.First(s.tableName, "id_prefix", addr, src)
	if err != nil {
		return err
	}
	ps := &ProviderSchema{
		Address: addr,
		Source:  src,
	}

	schemaCopy := schema.Copy()

	if obj != nil {
		existingEntry, ok := obj.(*ProviderSchema)
		if !ok {
			return fmt.Errorf("existing entry is not ProviderSchema")
		}

		if existingEntry.Schema == nil {
			if existingEntry.Version == nil {
				// This would be effectively a duplicate entry
				// which we just ignore and reinsert it
				// because there may be 2 (or more) schema refreshes
				// in progress at a time and we may not have obtained
				// the new version yet.
			} else {
				ps.Version = existingEntry.Version
				schemaCopy.SetProviderVersion(addr, existingEntry.Version)
			}

			err = txn.Delete(s.tableName, existingEntry)
			if err != nil {
				return err
			}
		}
	}

	ps.Schema = schemaCopy

	err = txn.Insert(s.tableName, ps)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ProviderSchemaStore) AddPreloadedSchema(addr tfaddr.Provider, pv *version.Version, schema *tfschema.ProviderSchema) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	src := PreloadedSchemaSource{}
	obj, err := txn.First(s.tableName, "id_prefix", addr, src, pv)
	if err != nil {
		return err
	}
	ps := &ProviderSchema{
		Address: addr,
		Version: pv,
		Source:  src,
	}
	if obj != nil {
		return &AlreadyExistsError{
			Idx: fmt.Sprintf("%s@%s@%s", addr, src, pv),
		}
	}

	schemaCopy := schema.Copy()

	ps.Schema = schemaCopy

	err = txn.Insert(s.tableName, ps)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *ProviderSchemaStore) ProviderSchema(modPath string, addr tfaddr.Provider, vc version.Constraints) (*tfschema.ProviderSchema, error) {
	s.logger.Printf("PSS: getting provider schema (%s, %s, %s)", modPath, addr, vc)
	txn := s.db.Txn(false)

	it, err := txn.Get(s.tableName, "id_prefix", addr)
	if err != nil {
		return nil, err
	}

	schemas := make([]*ProviderSchema, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		ps, ok := item.(*ProviderSchema)
		if ok {
			if ps.Schema == nil {
				// Incomplete entry may be a result of provider version being
				// sourced earlier where schema is yet to be sourced or sourcing failed.
				continue
			}
			schemas = append(schemas, ps)
		}
	}

	if len(schemas) == 0 && addr.Equals(tfaddr.NewDefaultProvider("terraform")) {
		// assume that hashicorp/terraform is just the builtin provider
		return s.ProviderSchema(modPath, tfaddr.NewBuiltInProvider("terraform"), vc)
	}

	if len(schemas) == 0 && addr.IsLegacy() {
		if addr.Type == "terraform" {
			return s.ProviderSchema(modPath, tfaddr.NewBuiltInProvider("terraform"), vc)
		}

		// Schema may be missing e.g. because Terraform 0.12
		// required relevant provider block to be present
		// to dump its schema in JSON output.

		// First we try to find a provider
		// by assuming the legacy provider is hashicorp's.
		addr.Namespace = "hashicorp"
		obj, err := txn.First(s.tableName, "id_prefix", addr)
		if err != nil {
			return nil, err
		}
		if obj != nil {
			ps := obj.(*ProviderSchema)
			if ps.Schema != nil {
				return ps.Schema, nil
			}
		}

		// Last we just try to loosely match the provider type
		it, err := txn.Get(s.tableName, "id")
		if err != nil {
			return nil, err
		}
		for item := it.Next(); item != nil; item = it.Next() {
			ps, ok := item.(*ProviderSchema)
			if ok && ps.Schema != nil && ps.Address.Type == addr.Type {
				schemas = append(schemas, ps)
			}
		}
	}

	if len(schemas) == 0 {
		return nil, &NoSchemaError{}
	}

	ss := sortableSchemas{
		schemas: schemas,
		lookupModule: func(modPath string) (*Module, error) {
			return moduleByPath(txn, modPath)
		},
		requiredModPath: modPath,
		requiredVersion: vc,
	}

	sort.Stable(ss)

	return ss.schemas[0].Schema, nil
}

type ModuleLookupFunc func(string) (*Module, error)

type sortableSchemas struct {
	schemas         []*ProviderSchema
	lookupModule    ModuleLookupFunc
	requiredModPath string
	requiredVersion version.Constraints
}

func (ss sortableSchemas) Len() int {
	return len(ss.schemas)
}

func (ss sortableSchemas) Less(i, j int) bool {
	var leftRank, rightRank int

	// TODO: Rank by version constraints match

	// TODO: Rank by hierarchy proximity

	// TODO: Rank by version

	leftRank += ss.rankBySource(ss.schemas[i].Source)
	rightRank += ss.rankBySource(ss.schemas[j].Source)

	return leftRank > rightRank
}

func (ss sortableSchemas) rankBySource(src SchemaSource) int {
	switch s := src.(type) {
	case PreloadedSchemaSource:
		return -1
	case LocalSchemaSource:
		if s.ModulePath == ss.requiredModPath {
			return 2
		}

		mod, err := ss.lookupModule(s.ModulePath)
		if err == nil && mod.ModManifest != nil &&
			mod.ModManifest.ContainsLocalModule(ss.requiredModPath) {
			return 1
		}
	}

	return 0
}

func (ss sortableSchemas) Swap(i, j int) {
	ss.schemas[i], ss.schemas[j] = ss.schemas[j], ss.schemas[i]
}

func (s *ProviderSchemaStore) ListSchemas() (*ProviderSchemaIterator, error) {
	txn := s.db.Txn(false)

	ri, err := txn.Get(s.tableName, "id")
	if err != nil {
		return nil, err
	}

	return &ProviderSchemaIterator{ri}, nil
}
