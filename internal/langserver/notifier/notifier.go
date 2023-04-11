// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package notifier

import (
	"context"
	"errors"
	"io/ioutil"
	"log"

	"github.com/hashicorp/terraform-ls/internal/state"
)

type moduleCtxKey struct{}
type moduleIsOpenCtxKey struct{}

type Notifier struct {
	modStore ModuleStore
	hooks    []Hook
	logger   *log.Logger
}

type ModuleStore interface {
	AwaitNextChangeBatch(ctx context.Context) (state.ModuleChangeBatch, error)
	ModuleByPath(path string) (*state.Module, error)
}

type Hook func(ctx context.Context, changes state.ModuleChanges) error

func NewNotifier(modStore ModuleStore, hooks []Hook) *Notifier {
	return &Notifier{
		modStore: modStore,
		hooks:    hooks,
		logger:   defaultLogger,
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
	changeBatch, err := n.modStore.AwaitNextChangeBatch(ctx)
	if err != nil {
		return err
	}

	mod, err := n.modStore.ModuleByPath(changeBatch.DirHandle.Path())
	if err != nil {
		return err
	}
	ctx = withModule(ctx, mod)

	ctx = withModuleIsOpen(ctx, changeBatch.IsDirOpen)

	for i, h := range n.hooks {
		err = h(ctx, changeBatch.Changes)
		if err != nil {
			n.logger.Printf("hook error (%d): %s", i, err)
			continue
		}
	}

	return nil
}

func withModule(ctx context.Context, mod *state.Module) context.Context {
	return context.WithValue(ctx, moduleCtxKey{}, mod)
}

func ModuleFromContext(ctx context.Context) (*state.Module, error) {
	mod, ok := ctx.Value(moduleCtxKey{}).(*state.Module)
	if !ok {
		return nil, errors.New("module data not found")
	}

	return mod, nil
}

func withModuleIsOpen(ctx context.Context, isOpen bool) context.Context {
	return context.WithValue(ctx, moduleIsOpenCtxKey{}, isOpen)
}

func ModuleIsOpen(ctx context.Context) (bool, error) {
	isOpen, ok := ctx.Value(moduleIsOpenCtxKey{}).(bool)
	if !ok {
		return false, errors.New("module open state not found")
	}

	return isOpen, nil
}

var defaultLogger = log.New(ioutil.Discard, "", 0)
