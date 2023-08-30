// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"fmt"
	"sort"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
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

func (s *ProviderSchemaStore) AllSchemasExist(pvm map[tfaddr.Provider]version.Constraints) (bool, error) {
	for pAddr, pCons := range pvm {
		exists, err := s.schemaExists(pAddr, pCons)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
	}

	return true, nil
}

// MissingSchemas checks which schemas are missing in order to preload them from the bundled schemas.
// Since we don't know the version of a schema on disk before loading it, we assume it's
// good to just load it by address and ignore the version constraints.
func (s *ProviderSchemaStore) MissingSchemas(pvm map[tfaddr.Provider]version.Constraints) ([]tfaddr.Provider, error) {
	missingSchemas := make([]tfaddr.Provider, 0)

	for pAddr := range pvm {
		if pAddr.IsLegacy() && pAddr.Type == "terraform" {
			// The terraform provider is built into Terraform 0.11+
			// and while it's possible, users typically don't declare
			// entry in required_providers block for it.
			pAddr = tfaddr.NewProvider(tfaddr.BuiltInProviderHost, tfaddr.BuiltInProviderNamespace, "terraform")
		} else if pAddr.IsLegacy() {
			// Since we use recent version of Terraform to generate
			// embedded schemas, these will never contain legacy
			// addresses.
			//
			// A legacy namespace may come from missing
			// required_providers entry & implied requirement
			// from the provider block or 0.12-style entry,
			// such as { grafana = "1.0" }.
			//
			// Implying "hashicorp" namespace here mimics behaviour
			// of all recent (0.14+) Terraform versions.
			pAddr.Namespace = "hashicorp"
		}

		exists, err := s.schemaExists(pAddr, version.Constraints{})
		if err != nil {
			return nil, err
		}
		if !exists {
			missingSchemas = append(missingSchemas, pAddr)
		}
	}
	return missingSchemas, nil
}

func (s *ProviderSchemaStore) schemaExists(addr tfaddr.Provider, pCons version.Constraints) (bool, error) {
	txn := s.db.Txn(false)

	it, err := txn.Get(s.tableName, "id_prefix", addr)
	if err != nil {
		return false, err
	}

	for item := it.Next(); item != nil; item = it.Next() {
		ps, ok := item.(*ProviderSchema)
		if !ok {
			continue
		}
		if ps.Schema == nil {
			// Incomplete entry may be a result of provider version being
			// sourced earlier where schema is yet to be sourced or sourcing failed.
			continue
		}
		if ps.Version == nil {
			// Obtaining schema is always done *after* getting the version.
			// Therefore, this can only happen in a rare case when getting
			// provider versions fails but getting schema was successful.
			// e.g. custom plugin cache location in combination with 0.12
			// (where lock files didn't exist) [1], or user-triggered race
			// condition when the lock file is deleted/created.
			// [1] See https://github.com/hashicorp/terraform-ls/issues/24
			continue
		}

		if providerAddrEquals(ps.Address, addr) && pCons.Check(ps.Version) {
			return true, nil
		}
	}

	return false, nil
}

func providerAddrEquals(a, b tfaddr.Provider) bool {
	if a.Equals(b) {
		return true
	}

	// Account for legacy addresses which may come from Terraform
	// 0.12 or 0.13 running locally or just lack of required_providers
	// entry in configuration.
	if a.IsLegacy() {
		a.Namespace = "hashicorp"
	}
	if b.IsLegacy() {
		b.Namespace = "hashicorp"
	}

	return a.Equals(b)
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

	if len(schemas) == 0 && addr.Equals(NewDefaultProvider("terraform")) {
		// assume that hashicorp/terraform is just the builtin provider
		return s.ProviderSchema(modPath, NewBuiltInProvider("terraform"), vc)
	}

	if len(schemas) == 0 && addr.IsLegacy() {
		if addr.Type == "terraform" {
			return s.ProviderSchema(modPath, NewBuiltInProvider("terraform"), vc)
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

func NewDefaultProvider(name string) tfaddr.Provider {
	return tfaddr.Provider{
		Type:      tfaddr.MustParseProviderPart(name),
		Namespace: "hashicorp",
		Hostname:  tfaddr.DefaultProviderRegistryHost,
	}
}

func NewBuiltInProvider(name string) tfaddr.Provider {
	return tfaddr.Provider{
		Type:      tfaddr.MustParseProviderPart(name),
		Namespace: tfaddr.BuiltInProviderNamespace,
		Hostname:  tfaddr.BuiltInProviderHost,
	}
}

func NewLegacyProvider(name string) tfaddr.Provider {
	return tfaddr.Provider{
		Type:      tfaddr.MustParseProviderPart(name),
		Namespace: tfaddr.LegacyProviderNamespace,
		Hostname:  tfaddr.DefaultProviderRegistryHost,
	}
}

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

	leftRank += ss.rankByVersionMatch(ss.schemas[i].Version)
	rightRank += ss.rankByVersionMatch(ss.schemas[j].Version)

	// TODO: Rank by hierarchy proximity

	// TODO: Rank by version (higher wins)

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

func (ss sortableSchemas) rankByVersionMatch(v *version.Version) int {
	if v != nil && ss.requiredVersion.Check(v) {
		return 2
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
