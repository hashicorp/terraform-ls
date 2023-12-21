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
type varsCtxKey struct{}
type moduleIsOpenCtxKey struct{}

type Notifier struct {
	modStore  ModuleStore
	varsStore VarsStore
	hooks     []Hook
	logger    *log.Logger
}

type ModuleStore interface {
	AwaitNextChangeBatch(ctx context.Context) (state.ModuleChangeBatch, error)
	ModuleByPath(path string) (*state.Module, error)
}

type VarsStore interface {
	VarsByPath(path string) (*state.Vars, error)
}

type Hook func(ctx context.Context, changes state.ModuleChanges) error

func NewNotifier(modStore ModuleStore, varsStore VarsStore, hooks []Hook) *Notifier {
	return &Notifier{
		modStore:  modStore,
		varsStore: varsStore,
		hooks:     hooks,
		logger:    defaultLogger,
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

	vars, err := n.varsStore.VarsByPath(changeBatch.DirHandle.Path())
	if err != nil {
		return err
	}
	ctx = withVars(ctx, vars)

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

// The next two functions are a bit of a hack to get around the fact that
// we don't have a way to pass the vars data to the notifier hooks. This
// is should be addressed in a future notifier refactor.
func withVars(ctx context.Context, vars *state.Vars) context.Context {
	return context.WithValue(ctx, varsCtxKey{}, vars)
}

func VarsFromContext(ctx context.Context) (*state.Vars, error) {
	vars, ok := ctx.Value(varsCtxKey{}).(*state.Vars)
	if !ok {
		return nil, errors.New("vars data not found")
	}

	return vars, nil
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
