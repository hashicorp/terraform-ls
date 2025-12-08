// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package notifier

import (
	"context"
	"errors"
	"io"
	"log"

	"github.com/hashicorp/terraform-ls/internal/state"
)

type recordPathCtxKey struct{}
type recordIsOpenCtxKey struct{}

type Notifier struct {
	changeStore ChangeStore
	hooks       []Hook
	logger      *log.Logger
}

type ChangeStore interface {
	AwaitNextChangeBatch(ctx context.Context) (state.ChangeBatch, error)
}

type Hook func(ctx context.Context, changes state.Changes) error

func NewNotifier(changeStore ChangeStore, hooks []Hook) *Notifier {
	return &Notifier{
		changeStore: changeStore,
		hooks:       hooks,
		logger:      defaultLogger,
	}
}

func (n *Notifier) SetLogger(logger *log.Logger) {
	n.logger = logger
}

func (n *Notifier) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				n.logger.Printf("stopping notifier: %s", ctx.Err())
				return
			default:
			}

			err := n.notify(ctx)
			if err != nil {
				n.logger.Printf("failed to notify a change batch: %s", err)
			}
		}
	}()
}

func (n *Notifier) notify(ctx context.Context) error {
	changeBatch, err := n.changeStore.AwaitNextChangeBatch(ctx)
	if err != nil {
		return err
	}

	ctx = withRecordPath(ctx, changeBatch.DirHandle.Path())

	ctx = withRecordIsOpen(ctx, changeBatch.IsDirOpen)

	for i, h := range n.hooks {
		err = h(ctx, changeBatch.Changes)
		if err != nil {
			n.logger.Printf("hook error (%d): %s", i, err)
			continue
		}
	}

	return nil
}

func withRecordPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, recordPathCtxKey{}, path)
}

func RecordPathFromContext(ctx context.Context) (string, error) {
	records, ok := ctx.Value(recordPathCtxKey{}).(string)
	if !ok {
		return "", errors.New("record path not found")
	}

	return records, nil
}

func withRecordIsOpen(ctx context.Context, isOpen bool) context.Context {
	return context.WithValue(ctx, recordIsOpenCtxKey{}, isOpen)
}

func RecordIsOpen(ctx context.Context) (bool, error) {
	isOpen, ok := ctx.Value(recordIsOpenCtxKey{}).(bool)
	if !ok {
		return false, errors.New("record open state not found")
	}

	return isOpen, nil
}

var defaultLogger = log.New(io.Discard, "", 0)
