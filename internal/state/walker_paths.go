// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/terraform-ls/internal/document"
)

type WalkerPathStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger

	nextOpenDirMu   *sync.Mutex
	nextClosedDirMu *sync.Mutex
}

type WalkerPath struct {
	Dir       document.DirHandle
	IsDirOpen bool
	State     PathState
}

//go:generate go run golang.org/x/tools/cmd/stringer -type=PathState -output=path_state_string.go
type PathState uint

const (
	PathStateQueued PathState = iota
	PathStateWalking
)

func (wp *WalkerPath) Copy() *WalkerPath {
	return &WalkerPath{
		Dir:       wp.Dir,
		IsDirOpen: wp.IsDirOpen,
	}
}

type PathAwaiter struct {
	wps     *WalkerPathStore
	openDir bool
}

func (pa *PathAwaiter) AwaitNextDir(ctx context.Context) (document.DirHandle, error) {
	wp, err := pa.wps.AwaitNextDir(ctx, pa.openDir)
	if err != nil {
		return document.DirHandle{}, err
	}
	return wp.Dir, nil
}

func (pa *PathAwaiter) RemoveDir(dir document.DirHandle) error {
	return pa.wps.RemoveDir(dir)
}

func NewPathAwaiter(wps *WalkerPathStore, openDir bool) *PathAwaiter {
	return &PathAwaiter{
		wps:     wps,
		openDir: openDir,
	}
}

func (wps *WalkerPathStore) EnqueueDir(dir document.DirHandle) error {
	txn := wps.db.Txn(true)
	defer txn.Abort()

	wp, err := txn.First(wps.tableName, "id", dir)
	if err != nil {
		return err
	}
	if wp != nil {
		// dir already enqueued
		return nil
	}

	err = txn.Insert(wps.tableName, &WalkerPath{
		Dir:       dir,
		IsDirOpen: false,
		State:     PathStateQueued,
	})
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

func (wps *WalkerPathStore) DequeueDir(dir document.DirHandle) error {
	txn := wps.db.Txn(true)
	defer txn.Abort()

	obj, err := txn.First(wps.tableName, "id", dir)
	if err != nil {
		return err
	}

	if obj == nil {
		// dir not enqueued
		return nil
	}

	wp := obj.(*WalkerPath)
	if wp.State == PathStateWalking {
		// avoid dequeuing dir which is already being walked
		return nil
	}

	_, err = txn.DeleteAll(wps.tableName, "id", dir)
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

func (wps *WalkerPathStore) RemoveDir(dir document.DirHandle) error {
	txn := wps.db.Txn(true)
	defer txn.Abort()

	_, err := txn.DeleteAll(wps.tableName, "id", dir)
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

func (wps *WalkerPathStore) AwaitNextDir(ctx context.Context, openDir bool) (*WalkerPath, error) {
	// Locking is needed if same query is executed in multiple threads,
	// i.e. this method is called at the same time from different threads, at
	// which point txn.FirstWatch would return the same job to more than
	// one thread and we would then end up executing it more than once.
	if openDir {
		wps.nextOpenDirMu.Lock()
		defer wps.nextOpenDirMu.Unlock()
	} else {
		wps.nextClosedDirMu.Lock()
		defer wps.nextClosedDirMu.Unlock()
	}

	return wps.awaitNextDir(ctx, openDir)
}

func (wps *WalkerPathStore) WaitForDirs(ctx context.Context, dirs []document.DirHandle) error {
	if len(dirs) == 0 {
		return nil
	}

	doneCh := make(chan struct{})
	go func() {
		defer func() {
			close(doneCh)
		}()

		for _, dirHandle := range dirs {
			err := wps.waitForDir(ctx, dirHandle)
			if err != nil {
				wps.logger.Printf("error waiting for dir %q: %s", dirHandle, err)
				return
			}
		}
	}()

	select {
	case <-doneCh:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (wps *WalkerPathStore) waitForDir(ctx context.Context, dir document.DirHandle) error {
	txn := wps.db.Txn(false)

	wCh, obj, err := txn.FirstWatch(wps.tableName, "id", dir)
	if err != nil {
		return err
	}

	if obj == nil {
		return nil
	}

	select {
	case <-wCh:
	case <-ctx.Done():
		return ctx.Err()
	}

	return wps.waitForDir(ctx, dir)
}

func (wps *WalkerPathStore) awaitNextDir(ctx context.Context, openDir bool) (*WalkerPath, error) {
	txn := wps.db.Txn(false)

	wCh, obj, err := txn.FirstWatch(wps.tableName, "is_dir_open_state", openDir, PathStateQueued)
	if err != nil {
		return nil, err
	}

	if obj == nil {
		select {
		case <-wCh:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		return wps.awaitNextDir(ctx, openDir)
	}

	wp := obj.(*WalkerPath)

	err = wps.markDirAsWalking(wp.Dir)
	if err != nil {
		// Although we hold a write db-wide lock when marking dir as walking
		// we may still end up passing the same dir from the above read-only
		// transaction, which does *not* hold a db-wide lock.
		//
		// Instead of adding more sync primitives here we simply retry.
		if errors.Is(err, pathAlreadyWalking{Dir: wp.Dir}) || errors.Is(err, walkerPathNotFound{Dir: wp.Dir}) {
			wps.logger.Printf("retrying next dir: %s", err)
			return wps.awaitNextDir(ctx, openDir)
		}

		return nil, err
	}

	wps.logger.Printf("walking next dir: %q", wp.Dir)
	return wp, nil
}

func (wps *WalkerPathStore) markDirAsWalking(dir document.DirHandle) error {
	txn := wps.db.Txn(true)
	defer txn.Abort()

	wp, err := copyWalkerPath(txn, dir)
	if err != nil {
		return err
	}

	if wp.State == PathStateWalking {
		return pathAlreadyWalking{Dir: dir}
	}

	_, err = txn.DeleteAll(wps.tableName, "id", dir)
	if err != nil {
		return err
	}

	wp.State = PathStateWalking

	err = txn.Insert(wps.tableName, wp)
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

func copyWalkerPath(txn *memdb.Txn, dir document.DirHandle) (*WalkerPath, error) {
	obj, err := txn.First(walkerPathsTableName, "id", dir)
	if err != nil {
		return nil, err
	}
	if obj != nil {
		wp := obj.(*WalkerPath)
		return wp.Copy(), nil
	}
	return nil, walkerPathNotFound{Dir: dir}
}

func updateWalkerDirOpenMark(txn *memdb.Txn, dirHandle document.DirHandle, isDirOpen bool) error {
	obj, err := txn.First(walkerPathsTableName, "id", dirHandle)
	if err != nil {
		return fmt.Errorf("failed to find queued directory %q: %w", dirHandle, err)
	}
	if obj == nil {
		return nil
	}

	existingWp := obj.(*WalkerPath)
	wp := existingWp.Copy()

	_, err = txn.DeleteAll(walkerPathsTableName, "id", wp.Dir)
	if err != nil {
		return err
	}

	wp.IsDirOpen = isDirOpen

	err = txn.Insert(walkerPathsTableName, wp)
	if err != nil {
		return err
	}

	return nil
}
