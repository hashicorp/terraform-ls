package state

import (
	"fmt"

	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/module"
)

type RegistryModuleMetadataSchema struct {
	Source  tfaddr.ModuleSourceRegistry
	Version *version.Version
	Inputs  []module.RegistryInput
	Outputs []module.RegistryOutput
}

func (s *RegistryModuleMetadataSchemaStore) Exists(sourceAddr tfaddr.ModuleSourceRegistry, constraint version.Constraints) bool {
	txn := s.db.Txn(false)

	iter, err := txn.Get(s.tableName, "id")
	if err != nil {
		return false
	}
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		p := obj.(*RegistryModuleMetadataSchema)
		if constraint.Check(p.Version) {
			return true
		}
	}

	return false
}

func (s *RegistryModuleMetadataSchemaStore) Cache(
	sourceAddr tfaddr.ModuleSourceRegistry,
	modVer *version.Version,
	inputs []module.RegistryInput,
	outputs []module.RegistryOutput,
) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	obj, err := txn.First(s.tableName, "id", sourceAddr, modVer)
	if err != nil {
		return err
	}
	if obj != nil {
		return &AlreadyExistsError{
			Idx: fmt.Sprintf("%s@%v", sourceAddr, modVer),
		}
	}

	thing := &RegistryModuleMetadataSchema{
		Source:  sourceAddr,
		Version: modVer,
		Inputs:  inputs,
		Outputs: outputs,
	}

	err = txn.Insert(s.tableName, thing)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}
