// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/terraform-ls/internal/document"
)

type ChangeBatch struct {
	DirHandle       document.DirHandle
	FirstChangeTime time.Time
	IsDirOpen       bool
	Changes         Changes
}

func (mcb ChangeBatch) Copy() ChangeBatch {
	return ChangeBatch{
		DirHandle:       mcb.DirHandle,
		FirstChangeTime: mcb.FirstChangeTime,
		IsDirOpen:       mcb.IsDirOpen,
		Changes:         mcb.Changes,
	}
}

type Changes struct {
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

func (s *ChangeStore) QueueChange(dir document.DirHandle, changes Changes) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	obj, err := txn.First(s.tableName, "id", dir)
	if err != nil {
		return err
	}

	var cb ChangeBatch
	if obj != nil {
		batch := obj.(ChangeBatch)
		cb = batch.Copy()

		// Update the existing change batch with the incoming changes.
		// The incoming change should never change a flag that is true back to false
		cb.Changes = Changes{
			IsRemoval:            cb.Changes.IsRemoval || changes.IsRemoval,
			CoreRequirements:     cb.Changes.CoreRequirements || changes.CoreRequirements,
			Backend:              cb.Changes.Backend || changes.Backend,
			Cloud:                cb.Changes.Cloud || changes.Cloud,
			ProviderRequirements: cb.Changes.ProviderRequirements || changes.ProviderRequirements,
			TerraformVersion:     cb.Changes.TerraformVersion || changes.TerraformVersion,
			InstalledProviders:   cb.Changes.InstalledProviders || changes.InstalledProviders,
			Diagnostics:          cb.Changes.Diagnostics || changes.Diagnostics,
			ReferenceOrigins:     cb.Changes.ReferenceOrigins || changes.ReferenceOrigins,
			ReferenceTargets:     cb.Changes.ReferenceTargets || changes.ReferenceTargets,
		}
	} else {
		// create new change batch
		isDirOpen, err := DirHasOpenDocuments(txn, dir)
		if err != nil {
			return err
		}
		cb = ChangeBatch{
			DirHandle:       dir,
			FirstChangeTime: s.TimeProvider(),
			Changes:         changes,
			IsDirOpen:       isDirOpen,
		}
	}

	// update change batch
	_, err = txn.DeleteAll(s.tableName, "id", dir)
	if err != nil {
		return err
	}

	err = txn.Insert(s.tableName, cb)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func updateModuleChangeDirOpenMark(txn *memdb.Txn, dirHandle document.DirHandle, isDirOpen bool) error {
	it, err := txn.Get(changesTableName, "id", dirHandle)
	if err != nil {
		return fmt.Errorf("failed to find module changes for %q: %w", dirHandle, err)
	}

	for obj := it.Next(); obj != nil; obj = it.Next() {
		batch := obj.(ChangeBatch)
		mcb := batch.Copy()

		_, err = txn.DeleteAll(changesTableName, "id", batch.DirHandle)
		if err != nil {
			return err
		}

		mcb.IsDirOpen = isDirOpen

		err = txn.Insert(changesTableName, mcb)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ChangeStore) AwaitNextChangeBatch(ctx context.Context) (ChangeBatch, error) {
	rTxn := s.db.Txn(false)
	wCh, obj, err := rTxn.FirstWatch(s.tableName, "time")
	if err != nil {
		return ChangeBatch{}, err
	}

	if obj == nil {
		select {
		case <-wCh:
		case <-ctx.Done():
			return ChangeBatch{}, ctx.Err()
		}

		return s.AwaitNextChangeBatch(ctx)
	}

	batch := obj.(ChangeBatch)

	timeout := batch.FirstChangeTime.Add(maxTimespan)
	if time.Now().After(timeout) {
		err := s.deleteChangeBatch(batch)
		if err != nil {
			return ChangeBatch{}, err
		}
		return batch, nil
	}

	wCh, jobsExist, err := JobsExistForDirHandle(rTxn, batch.DirHandle)
	if err != nil {
		return ChangeBatch{}, err
	}
	if !jobsExist {
		err := s.deleteChangeBatch(batch)
		if err != nil {
			return ChangeBatch{}, err
		}
		return batch, nil
	}

	select {
	// wait for another job to get processed
	case <-wCh:
	// or for the remaining time to pass
	case <-time.After(time.Until(timeout)):
	// or context cancellation
	case <-ctx.Done():
		return ChangeBatch{}, ctx.Err()
	}

	return s.AwaitNextChangeBatch(ctx)
}

func (s *ChangeStore) deleteChangeBatch(batch ChangeBatch) error {
	txn := s.db.Txn(true)
	defer txn.Abort()
	err := txn.Delete(s.tableName, batch)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}
