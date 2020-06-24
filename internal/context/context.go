package context

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
	"github.com/hashicorp/terraform-ls/internal/watcher"
	"github.com/sourcegraph/go-lsp"
)

type contextKey struct {
	Name string
}

func (k *contextKey) String() string {
	return k.Name
}

var (
	ctxFs               = &contextKey{"filesystem"}
	ctxClientCapsSetter = &contextKey{"client capabilities setter"}
	ctxClientCaps       = &contextKey{"client capabilities"}
	ctxTfExecPath       = &contextKey{"terraform executable path"}
	ctxTfExecLogPath    = &contextKey{"terraform executor log path"}
	ctxTfExecTimeout    = &contextKey{"terraform execution timeout"}
	ctxWatcher          = &contextKey{"watcher"}
	ctxRootModuleMngr   = &contextKey{"root module manager"}
	ctxParserFinder     = &contextKey{"parser finder"}
	ctxTfExecFinder     = &contextKey{"terraform exec finder"}
	ctxRootModuleCaFi   = &contextKey{"root module candidate finder"}
	ctxRootDir          = &contextKey{"root directory"}
)

func missingContextErr(ctxKey *contextKey) *MissingContextErr {
	return &MissingContextErr{ctxKey}
}

func WithFilesystem(fs filesystem.Filesystem, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxFs, fs)
}

func Filesystem(ctx context.Context) (filesystem.Filesystem, error) {
	fs, ok := ctx.Value(ctxFs).(filesystem.Filesystem)
	if !ok {
		return nil, missingContextErr(ctxFs)
	}

	return fs, nil
}

func WithClientCapabilitiesSetter(caps *lsp.ClientCapabilities, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxClientCapsSetter, caps)
}

func SetClientCapabilities(ctx context.Context, caps *lsp.ClientCapabilities) error {
	cc, ok := ctx.Value(ctxClientCapsSetter).(*lsp.ClientCapabilities)
	if !ok {
		return missingContextErr(ctxClientCapsSetter)
	}

	*cc = *caps
	return nil
}

func WithClientCapabilities(caps *lsp.ClientCapabilities, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxClientCaps, caps)
}

func ClientCapabilities(ctx context.Context) (lsp.ClientCapabilities, error) {
	caps, ok := ctx.Value(ctxClientCaps).(*lsp.ClientCapabilities)
	if !ok {
		return lsp.ClientCapabilities{}, missingContextErr(ctxClientCaps)
	}

	return *caps, nil
}

func WithTerraformExecLogPath(path string, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfExecLogPath, path)
}

func TerraformExecLogPath(ctx context.Context) (string, bool) {
	path, ok := ctx.Value(ctxTfExecLogPath).(string)
	return path, ok
}

func WithTerraformExecTimeout(timeout time.Duration, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfExecTimeout, timeout)
}

func TerraformExecTimeout(ctx context.Context) (time.Duration, bool) {
	path, ok := ctx.Value(ctxTfExecTimeout).(time.Duration)
	return path, ok
}

func WithWatcher(w watcher.Watcher, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxWatcher, w)
}

func Watcher(ctx context.Context) (watcher.Watcher, error) {
	w, ok := ctx.Value(ctxWatcher).(watcher.Watcher)
	if !ok {
		return nil, missingContextErr(ctxWatcher)
	}
	return w, nil
}

func WithRootModuleManager(wm rootmodule.RootModuleManager, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxRootModuleMngr, wm)
}

func RootModuleManager(ctx context.Context) (rootmodule.RootModuleManager, error) {
	wm, ok := ctx.Value(ctxRootModuleMngr).(rootmodule.RootModuleManager)
	if !ok {
		return nil, missingContextErr(ctxRootModuleMngr)
	}
	return wm, nil
}

func WithParserFinder(pf rootmodule.ParserFinder, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxParserFinder, pf)
}

func ParserFinder(ctx context.Context) (rootmodule.ParserFinder, error) {
	pf, ok := ctx.Value(ctxParserFinder).(rootmodule.ParserFinder)
	if !ok {
		return nil, missingContextErr(ctxParserFinder)
	}
	return pf, nil
}

func WithTerraformExecFinder(tef rootmodule.TerraformExecFinder, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfExecFinder, tef)
}

func TerraformExecutorFinder(ctx context.Context) (rootmodule.TerraformExecFinder, error) {
	pf, ok := ctx.Value(ctxTfExecFinder).(rootmodule.TerraformExecFinder)
	if !ok {
		return nil, missingContextErr(ctxTfExecFinder)
	}
	return pf, nil
}

func WithTerraformExecPath(path string, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfExecPath, path)
}

func TerraformExecPath(ctx context.Context) (string, bool) {
	path, ok := ctx.Value(ctxTfExecPath).(string)
	return path, ok
}

func WithRootModuleCandidateFinder(rmcf rootmodule.RootModuleCandidateFinder, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxRootModuleCaFi, rmcf)
}

func RootModuleCandidateFinder(ctx context.Context) (rootmodule.RootModuleCandidateFinder, error) {
	cf, ok := ctx.Value(ctxRootModuleCaFi).(rootmodule.RootModuleCandidateFinder)
	if !ok {
		return nil, missingContextErr(ctxRootModuleCaFi)
	}
	return cf, nil
}

func WithRootDirectory(dir *string, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxRootDir, dir)
}

func SetRootDirectory(ctx context.Context, dir string) error {
	rootDir, ok := ctx.Value(ctxRootDir).(*string)
	if !ok {
		return missingContextErr(ctxRootDir)
	}

	*rootDir = dir
	return nil
}

func RootDirectory(ctx context.Context) (string, bool) {
	rootDir, ok := ctx.Value(ctxRootDir).(*string)
	if !ok {
		return "", false
	}
	return *rootDir, true
}
