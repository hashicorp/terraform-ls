package state

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/terraform-ls/internal/document"
)

type ModuleChangeBatch struct {
	DirHandle       document.DirHandle
	FirstChangeTime time.Time
	IsDirOpen       bool
	Changes         ModuleChanges
}

func (mcb ModuleChangeBatch) Copy() ModuleChangeBatch {
	return ModuleChangeBatch{
		DirHandle:       mcb.DirHandle,
		FirstChangeTime: mcb.FirstChangeTime,
		IsDirOpen:       mcb.IsDirOpen,
		Changes:         mcb.Changes,
	}
}

type ModuleChanges struct {
	// IsRemoval indicates whether this batch represents removal of a module
	IsRemoval bool

	CoreRequirements     bool
	Backend              bool
	Cloud                bool
	ProviderRequirements bool
	TerraformVersion     bool
	InstalledProviders   bool
	Diagnostics          bool
	ReferenceOrigins     bool
	ReferenceTargets     bool
}

const maxTimespan = 1 * time.Second

func (s *ModuleStore) queueModuleChange(txn *memdb.Txn, oldMod, newMod *Module) error {
	var modHandle document.DirHandle
	if oldMod != nil {
		modHandle = document.DirHandleFromPath(oldMod.Path)
	} else {
		modHandle = document.DirHandleFromPath(newMod.Path)
	}
	obj, err := txn.First(moduleChangesTableName, "id", modHandle)
	if err != nil {
		return err
	}

	var cb ModuleChangeBatch
	if obj != nil {
		batch := obj.(ModuleChangeBatch)
		cb = batch.Copy()
	} else {
		// create new change batch
		isDirOpen, err := dirHasOpenDocuments(txn, modHandle)
		if err != nil {
			return err
		}
		cb = ModuleChangeBatch{
			DirHandle:       modHandle,
			FirstChangeTime: s.TimeProvider(),
			Changes:         ModuleChanges{},
			IsDirOpen:       isDirOpen,
		}
	}

	switch {
	// new module added
	case oldMod == nil && newMod != nil:
		if len(newMod.Meta.CoreRequirements) > 0 {
			cb.Changes.CoreRequirements = true
		}
		if newMod.Meta.Cloud != nil {
			cb.Changes.Cloud = true
		}
		if newMod.Meta.Backend != nil {
			cb.Changes.Backend = true
		}
		if len(newMod.Meta.ProviderRequirements) > 0 {
			cb.Changes.ProviderRequirements = true
		}
		if newMod.TerraformVersion != nil {
			cb.Changes.TerraformVersion = true
		}
		if len(newMod.InstalledProviders) > 0 {
			cb.Changes.InstalledProviders = true
		}
	// module removed
	case oldMod != nil && newMod == nil:
		cb.Changes.IsRemoval = true

		if len(oldMod.Meta.CoreRequirements) > 0 {
			cb.Changes.CoreRequirements = true
		}
		if oldMod.Meta.Cloud != nil {
			cb.Changes.Cloud = true
		}
		if oldMod.Meta.Backend != nil {
			cb.Changes.Backend = true
		}
		if len(oldMod.Meta.ProviderRequirements) > 0 {
			cb.Changes.ProviderRequirements = true
		}
		if oldMod.TerraformVersion != nil {
			cb.Changes.TerraformVersion = true
		}
		if len(oldMod.InstalledProviders) > 0 {
			cb.Changes.InstalledProviders = true
		}
	// module changed
	default:
		if !oldMod.Meta.CoreRequirements.Equals(newMod.Meta.CoreRequirements) {
			cb.Changes.CoreRequirements = true
		}
		if !oldMod.Meta.Backend.Equals(newMod.Meta.Backend) {
			cb.Changes.Backend = true
		}
		if !oldMod.Meta.Cloud.Equals(newMod.Meta.Cloud) {
			cb.Changes.Cloud = true
		}
		if !oldMod.Meta.ProviderRequirements.Equals(newMod.Meta.ProviderRequirements) {
			cb.Changes.ProviderRequirements = true
		}
		if !oldMod.TerraformVersion.Equal(newMod.TerraformVersion) {
			cb.Changes.TerraformVersion = true
		}
		if !oldMod.InstalledProviders.Equals(newMod.InstalledProviders) {
			cb.Changes.InstalledProviders = true
		}
	}

	oldDiags, newDiags := 0, 0
	if oldMod != nil {
		oldDiags = oldMod.ModuleDiagnostics.Count() + oldMod.VarsDiagnostics.Count()
	}
	if newMod != nil {
		newDiags = newMod.ModuleDiagnostics.Count() + newMod.VarsDiagnostics.Count()
	}
	// Comparing diagnostics accurately could be expensive
	// so we just treat any non-empty diags as a change
	if oldDiags > 0 || newDiags > 0 {
		cb.Changes.Diagnostics = true
	}

	oldOrigins, oldTargets := 0, 0
	if oldMod != nil {
		oldOrigins = len(oldMod.RefOrigins)
		oldTargets = len(oldMod.RefTargets)
	}
	newOrigins, newTargets := 0, 0
	if newMod != nil {
		newOrigins = len(newMod.RefOrigins)
		newTargets = len(newMod.RefTargets)
	}
	if oldOrigins != newOrigins {
		cb.Changes.ReferenceOrigins = true
	}
	if oldTargets != newTargets {
		cb.Changes.ReferenceTargets = true
	}

	// update change batch
	_, err = txn.DeleteAll(moduleChangesTableName, "id", modHandle)
	if err != nil {
		return err
	}
	return txn.Insert(moduleChangesTableName, cb)
}

func updateModuleChangeDirOpenMark(txn *memdb.Txn, dirHandle document.DirHandle, isDirOpen bool) error {
	it, err := txn.Get(moduleChangesTableName, "id", dirHandle)
	if err != nil {
		return fmt.Errorf("failed to find module changes for %q: %w", dirHandle, err)
	}

	for obj := it.Next(); obj != nil; obj = it.Next() {
		batch := obj.(ModuleChangeBatch)
		mcb := batch.Copy()

		_, err = txn.DeleteAll(moduleChangesTableName, "id", batch.DirHandle)
		if err != nil {
			return err
		}

		mcb.IsDirOpen = isDirOpen

		err = txn.Insert(moduleChangesTableName, mcb)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ms *ModuleStore) AwaitNextChangeBatch(ctx context.Context) (ModuleChangeBatch, error) {
	rTxn := ms.db.Txn(false)
	wCh, obj, err := rTxn.FirstWatch(moduleChangesTableName, "time")
	if err != nil {
		return ModuleChangeBatch{}, err
	}

	if obj == nil {
		select {
		case <-wCh:
		case <-ctx.Done():
			return ModuleChangeBatch{}, ctx.Err()
		}

		return ms.AwaitNextChangeBatch(ctx)
	}

	batch := obj.(ModuleChangeBatch)

	timeout := batch.FirstChangeTime.Add(maxTimespan)
	if time.Now().After(timeout) {
		err := ms.deleteChangeBatch(batch)
		if err != nil {
			return ModuleChangeBatch{}, err
		}
		return batch, nil
	}

	wCh, jobsExist, err := jobsExistForDirHandle(rTxn, batch.DirHandle)
	if err != nil {
		return ModuleChangeBatch{}, err
	}
	if !jobsExist {
		err := ms.deleteChangeBatch(batch)
		if err != nil {
			return ModuleChangeBatch{}, err
		}
		return batch, nil
	}

	select {
	// wait for another job to get processed
	case <-wCh:
	// or for the remaining time to pass
	case <-time.After(timeout.Sub(time.Now())):
	// or context cancellation
	case <-ctx.Done():
		return ModuleChangeBatch{}, ctx.Err()
	}

	return ms.AwaitNextChangeBatch(ctx)
}

func (ms *ModuleStore) deleteChangeBatch(batch ModuleChangeBatch) error {
	txn := ms.db.Txn(true)
	defer txn.Abort()
	err := txn.Delete(moduleChangesTableName, batch)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}
