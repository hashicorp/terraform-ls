package context

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/settings"
	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
	"github.com/hashicorp/terraform-ls/internal/watcher"
)

type contextKey struct {
	Name string
}

func (k *contextKey) String() string {
	return k.Name
}

var (
	ctxDs                   = &contextKey{"document storage"}
	ctxClientCapsSetter     = &contextKey{"client capabilities setter"}
	ctxClientCaps           = &contextKey{"client capabilities"}
	ctxTfExecPath           = &contextKey{"terraform executable path"}
	ctxTfExecLogPath        = &contextKey{"terraform executor log path"}
	ctxTfExecTimeout        = &contextKey{"terraform execution timeout"}
	ctxWatcher              = &contextKey{"watcher"}
	ctxRootModuleMngr       = &contextKey{"root module manager"}
	ctxTfFormatterFinder    = &contextKey{"terraform formatter finder"}
	ctxRootModuleCaFi       = &contextKey{"root module candidate finder"}
	ctxRootModuleWalker     = &contextKey{"root module walker"}
	ctxRootModuleLoader     = &contextKey{"root module loader"}
	ctxRootDir              = &contextKey{"root directory"}
	ctxCommandPrefix        = &contextKey{"command prefix"}
	ctxDiags                = &contextKey{"diagnostics"}
	ctxLsVersion            = &contextKey{"language server version"}
	ctxProgressToken        = &contextKey{"progress token"}
	ctxExperimentalFeatures = &contextKey{"experimental features"}
)

func missingContextErr(ctxKey *contextKey) *MissingContextErr {
	return &MissingContextErr{ctxKey}
}

func WithDocumentStorage(ctx context.Context, fs filesystem.DocumentStorage) context.Context {
	return context.WithValue(ctx, ctxDs, fs)
}

func DocumentStorage(ctx context.Context) (filesystem.DocumentStorage, error) {
	fs, ok := ctx.Value(ctxDs).(filesystem.DocumentStorage)
	if !ok {
		return nil, missingContextErr(ctxDs)
	}

	return fs, nil
}

func WithClientCapabilitiesSetter(ctx context.Context, caps *lsp.ClientCapabilities) context.Context {
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

func WithClientCapabilities(ctx context.Context, caps *lsp.ClientCapabilities) context.Context {
	return context.WithValue(ctx, ctxClientCaps, caps)
}

func ClientCapabilities(ctx context.Context) (lsp.ClientCapabilities, error) {
	caps, ok := ctx.Value(ctxClientCaps).(*lsp.ClientCapabilities)
	if !ok {
		return lsp.ClientCapabilities{}, missingContextErr(ctxClientCaps)
	}

	return *caps, nil
}

func WithTerraformExecLogPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, ctxTfExecLogPath, path)
}

func TerraformExecLogPath(ctx context.Context) (string, bool) {
	path, ok := ctx.Value(ctxTfExecLogPath).(string)
	return path, ok
}

func WithTerraformExecTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, ctxTfExecTimeout, timeout)
}

func TerraformExecTimeout(ctx context.Context) (time.Duration, bool) {
	path, ok := ctx.Value(ctxTfExecTimeout).(time.Duration)
	return path, ok
}

func WithWatcher(ctx context.Context, w watcher.Watcher) context.Context {
	return context.WithValue(ctx, ctxWatcher, w)
}

func Watcher(ctx context.Context) (watcher.Watcher, error) {
	w, ok := ctx.Value(ctxWatcher).(watcher.Watcher)
	if !ok {
		return nil, missingContextErr(ctxWatcher)
	}
	return w, nil
}

func WithRootModuleManager(ctx context.Context, wm rootmodule.RootModuleManager) context.Context {
	return context.WithValue(ctx, ctxRootModuleMngr, wm)
}

func RootModuleManager(ctx context.Context) (rootmodule.RootModuleManager, error) {
	wm, ok := ctx.Value(ctxRootModuleMngr).(rootmodule.RootModuleManager)
	if !ok {
		return nil, missingContextErr(ctxRootModuleMngr)
	}
	return wm, nil
}

func WithTerraformFormatterFinder(ctx context.Context, tef rootmodule.TerraformFormatterFinder) context.Context {
	return context.WithValue(ctx, ctxTfFormatterFinder, tef)
}

func TerraformFormatterFinder(ctx context.Context) (rootmodule.TerraformFormatterFinder, error) {
	pf, ok := ctx.Value(ctxTfFormatterFinder).(rootmodule.TerraformFormatterFinder)
	if !ok {
		return nil, missingContextErr(ctxTfFormatterFinder)
	}
	return pf, nil
}

func WithTerraformExecPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, ctxTfExecPath, path)
}

func TerraformExecPath(ctx context.Context) (string, bool) {
	path, ok := ctx.Value(ctxTfExecPath).(string)
	return path, ok
}

func WithRootModuleFinder(ctx context.Context, rmcf rootmodule.RootModuleFinder) context.Context {
	return context.WithValue(ctx, ctxRootModuleCaFi, rmcf)
}

func RootModuleFinder(ctx context.Context) (rootmodule.RootModuleFinder, error) {
	cf, ok := ctx.Value(ctxRootModuleCaFi).(rootmodule.RootModuleFinder)
	if !ok {
		return nil, missingContextErr(ctxRootModuleCaFi)
	}
	return cf, nil
}

func WithRootDirectory(ctx context.Context, dir *string) context.Context {
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

func WithCommandPrefix(ctx context.Context, prefix *string) context.Context {
	return context.WithValue(ctx, ctxCommandPrefix, prefix)
}

func SetCommandPrefix(ctx context.Context, prefix string) error {
	commandPrefix, ok := ctx.Value(ctxCommandPrefix).(*string)
	if !ok {
		return missingContextErr(ctxCommandPrefix)
	}

	*commandPrefix = prefix
	return nil
}

func CommandPrefix(ctx context.Context) (string, bool) {
	commandPrefix, ok := ctx.Value(ctxCommandPrefix).(*string)
	if !ok {
		return "", false
	}
	return *commandPrefix, true
}

func WithRootModuleWalker(ctx context.Context, w *rootmodule.Walker) context.Context {
	return context.WithValue(ctx, ctxRootModuleWalker, w)
}

func RootModuleWalker(ctx context.Context) (*rootmodule.Walker, error) {
	w, ok := ctx.Value(ctxRootModuleWalker).(*rootmodule.Walker)
	if !ok {
		return nil, missingContextErr(ctxRootModuleWalker)
	}
	return w, nil
}

func WithRootModuleLoader(ctx context.Context, rml rootmodule.RootModuleLoader) context.Context {
	return context.WithValue(ctx, ctxRootModuleLoader, rml)
}

func RootModuleLoader(ctx context.Context) (rootmodule.RootModuleLoader, error) {
	w, ok := ctx.Value(ctxRootModuleLoader).(rootmodule.RootModuleLoader)
	if !ok {
		return nil, missingContextErr(ctxRootModuleLoader)
	}
	return w, nil
}

func WithDiagnostics(ctx context.Context, diags *diagnostics.Notifier) context.Context {
	return context.WithValue(ctx, ctxDiags, diags)
}

func Diagnostics(ctx context.Context) (*diagnostics.Notifier, error) {
	diags, ok := ctx.Value(ctxDiags).(*diagnostics.Notifier)
	if !ok {
		return nil, missingContextErr(ctxDiags)
	}

	return diags, nil
}

func WithLanguageServerVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, ctxLsVersion, version)
}

func LanguageServerVersion(ctx context.Context) (string, bool) {
	version, ok := ctx.Value(ctxLsVersion).(string)
	if !ok {
		return "", false
	}
	return version, true
}

func WithProgressToken(ctx context.Context, pt lsp.ProgressToken) context.Context {
	return context.WithValue(ctx, ctxProgressToken, pt)
}

func ProgressToken(ctx context.Context) (lsp.ProgressToken, bool) {
	pt, ok := ctx.Value(ctxProgressToken).(lsp.ProgressToken)
	if !ok {
		return "", false
	}
	return pt, true
}

func WithExperimentalFeatures(ctx context.Context, expFeatures *settings.ExperimentalFeatures) context.Context {
	return context.WithValue(ctx, ctxExperimentalFeatures, expFeatures)
}

func SetExperimentalFeatures(ctx context.Context, expFeatures settings.ExperimentalFeatures) error {
	e, ok := ctx.Value(ctxExperimentalFeatures).(*settings.ExperimentalFeatures)
	if !ok {
		return missingContextErr(ctxExperimentalFeatures)
	}

	*e = expFeatures
	return nil
}

func ExperimentalFeatures(ctx context.Context) (settings.ExperimentalFeatures, error) {
	expFeatures, ok := ctx.Value(ctxExperimentalFeatures).(*settings.ExperimentalFeatures)
	if !ok {
		return settings.ExperimentalFeatures{}, missingContextErr(ctxExperimentalFeatures)
	}
	return *expFeatures, nil
}
