// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package context

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/settings"
)

type contextKey struct {
	Name string
}

func (k *contextKey) String() string {
	return k.Name
}

type RPCContextData struct {
	Method string
	URI    string
}

func (rpcc RPCContextData) Copy() RPCContextData {
	return RPCContextData{
		Method: rpcc.Method,
		URI:    rpcc.URI,
	}
}

var (
	ctxTfExecPath           = &contextKey{"terraform executable path"}
	ctxTfExecLogPath        = &contextKey{"terraform executor log path"}
	ctxTfExecTimeout        = &contextKey{"terraform execution timeout"}
	ctxRootDir              = &contextKey{"root directory"}
	ctxCommandPrefix        = &contextKey{"command prefix"}
	ctxDiagsNotifier        = &contextKey{"diagnostics notifier"}
	ctxLsVersion            = &contextKey{"language server version"}
	ctxProgressToken        = &contextKey{"progress token"}
	ctxExperimentalFeatures = &contextKey{"experimental features"}
	ctxRPCContext           = &contextKey{"rpc context"}
	ctxLanguageId           = &contextKey{"language ID"}
	ctxValidationOptions    = &contextKey{"validation options"}
)

func missingContextErr(ctxKey *contextKey) *MissingContextErr {
	return &MissingContextErr{ctxKey}
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

func WithTerraformExecPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, ctxTfExecPath, path)
}

func TerraformExecPath(ctx context.Context) (string, bool) {
	path, ok := ctx.Value(ctxTfExecPath).(string)
	return path, ok
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

func WithDiagnosticsNotifier(ctx context.Context, diags *diagnostics.Notifier) context.Context {
	return context.WithValue(ctx, ctxDiagsNotifier, diags)
}

func DiagnosticsNotifier(ctx context.Context) (*diagnostics.Notifier, error) {
	diags, ok := ctx.Value(ctxDiagsNotifier).(*diagnostics.Notifier)
	if !ok {
		return nil, missingContextErr(ctxDiagsNotifier)
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

func WithRPCContext(ctx context.Context, rpcc RPCContextData) context.Context {
	return context.WithValue(ctx, ctxRPCContext, rpcc)
}

func RPCContext(ctx context.Context) RPCContextData {
	return ctx.Value(ctxRPCContext).(RPCContextData)
}

func (ctxData RPCContextData) IsDidChangeRequest() bool {
	return ctxData.Method == "textDocument/didChange"
}

func WithLanguageId(ctx context.Context, languageId string) context.Context {
	return context.WithValue(ctx, ctxLanguageId, languageId)
}

func IsLanguageId(ctx context.Context, expectedLangId string) bool {
	langId, ok := ctx.Value(ctxLanguageId).(string)
	if !ok {
		return false
	}
	return langId == expectedLangId
}

func WithValidationOptions(ctx context.Context, validationOptions *settings.ValidationOptions) context.Context {
	return context.WithValue(ctx, ctxValidationOptions, validationOptions)
}

func SetValidationOptions(ctx context.Context, validationOptions settings.ValidationOptions) error {
	e, ok := ctx.Value(ctxValidationOptions).(*settings.ValidationOptions)
	if !ok {
		return missingContextErr(ctxValidationOptions)
	}

	*e = validationOptions
	return nil
}

func ValidationOptions(ctx context.Context) (settings.ValidationOptions, error) {
	validationOptions, ok := ctx.Value(ctxValidationOptions).(*settings.ValidationOptions)
	if !ok {
		return settings.ValidationOptions{}, missingContextErr(ctxValidationOptions)
	}
	return *validationOptions, nil
}
