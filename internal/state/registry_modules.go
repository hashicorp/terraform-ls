package state

import (
	"fmt"

	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/registry"
)

type RegistryModuleData struct {
	Source  tfaddr.Module
	Version *version.Version
	Inputs  []registry.Input
	Outputs []registry.Output
}

func (s *RegistryModuleStore) Exists(sourceAddr tfaddr.Module, constraint version.Constraints) (bool, error) {
	txn := s.db.Txn(false)

	iter, err := txn.Get(s.tableName, "source_addr", sourceAddr)
	if err != nil {
		return false, err
	}

	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		p := obj.(*RegistryModuleData)
		if constraint.Check(p.Version) {
			return true, nil
		}
	}

	return false, nil
}

func (s *RegistryModuleStore) Cache(sourceAddr tfaddr.Module, modVer *version.Version,
	inputs []registry.Input, outputs []registry.Output) error {

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

	modData := &RegistryModuleData{
		Source:  sourceAddr,
		Version: modVer,
		Inputs:  inputs,
		Outputs: outputs,
	}

	err = txn.Insert(s.tableName, modData)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}
