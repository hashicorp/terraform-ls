package module

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
)

type OpState uint

const (
	OpStateUnknown OpState = iota
	OpStateQueued
	OpStateLoading
	OpStateLoaded
)

type OpType uint

const (
	OpTypeUnknown OpType = iota
	OpTypeGetTerraformVersion
	OpTypeObtainSchema
	OpTypeParseConfiguration
	OpTypeParseModuleManifest
)

func (t OpType) String() string {
	switch t {
	case OpTypeUnknown:
		return "OpTypeUnknown"
	case OpTypeGetTerraformVersion:
		return "OpTypeGetTerraformVersion"
	case OpTypeObtainSchema:
		return "OpTypeObtainSchema"
	case OpTypeParseConfiguration:
		return "OpTypeParseConfiguration"
	case OpTypeParseModuleManifest:
		return "OpTypeParseModuleManifest"
	}

	return fmt.Sprintf("OpType(%d)", t)
}

type ModuleOperation struct {
	Module Module
	Type   OpType

	doneCh chan struct{}
}

func NewModuleOperation(mod Module, typ OpType) ModuleOperation {
	return ModuleOperation{
		Module: mod,
		Type:   typ,
		doneCh: make(chan struct{}, 1),
	}
}

func (mo ModuleOperation) markAsDone() {
	mo.doneCh <- struct{}{}
}

func (mo ModuleOperation) Done() <-chan struct{} {
	return mo.doneCh
}

func GetTerraformVersion(ctx context.Context, mod Module) {
	m := mod.(*module)

	m.SetTerraformVersionState(OpStateLoading)
	defer m.SetTerraformVersionState(OpStateLoaded)

	tfExec, err := TerraformExecutorForModule(ctx, mod)
	if err != nil {
		m.SetTerraformVersion(nil, err)
		m.logger.Printf("getting executor failed: %s", err)
		return
	}

	v, pv, err := tfExec.Version(ctx)
	if err != nil {
		m.logger.Printf("failed to get terraform version: %s", err)
	} else {
		m.logger.Printf("got terraform version successfully for %s", m.Path())
	}

	m.SetTerraformVersion(v, err)
	if len(pv) > 0 {
		m.SetProviderVersions(pv)
	}
}

func ObtainSchema(ctx context.Context, mod Module) {
	m := mod.(*module)
	m.SetProviderSchemaObtainingState(OpStateLoading)
	defer m.SetProviderSchemaObtainingState(OpStateLoaded)

	tfExec, err := TerraformExecutorForModule(ctx, mod)
	if err != nil {
		m.SetProviderSchemas(nil, err)
		m.logger.Printf("getting executor failed: %s", err)
		return
	}

	ps, err := tfExec.ProviderSchemas(ctx)
	if err != nil {
		m.logger.Printf("failed to obtain schema: %s", err)
	} else {
		m.logger.Printf("schema obtained successfully for %s", m.Path())
	}

	m.SetProviderSchemas(ps, err)
}

func ParseConfiguration(mod Module) {
	m := mod.(*module)
	m.SetConfigParsingState(OpStateLoading)
	defer m.SetConfigParsingState(OpStateLoaded)

	files := make(map[string]*hcl.File, 0)
	diags := make(map[string]hcl.Diagnostics, 0)

	infos, err := m.fs.ReadDir(m.Path())
	if err != nil {
		m.SetParsedFiles(files, err)
		return
	}

	for _, info := range infos {
		if info.IsDir() {
			// We only care about files
			continue
		}

		name := info.Name()
		if !strings.HasSuffix(name, ".tf") || IsIgnoredFile(name) {
			continue
		}

		// TODO: overrides

		fullPath := filepath.Join(m.Path(), name)

		src, err := m.fs.ReadFile(fullPath)
		if err != nil {
			m.SetParsedFiles(files, err)
			return
		}

		f, pDiags := hclsyntax.ParseConfig(src, name, hcl.InitialPos)
		diags[name] = pDiags
		if f != nil {
			files[name] = f
		}
	}

	m.SetParsedFiles(files, err)
	m.SetDiagnostics(diags)
	return
}

// IsIgnoredFile returns true if the given filename (which must not have a
// directory path ahead of it) should be ignored as e.g. an editor swap file.
func IsIgnoredFile(name string) bool {
	return strings.HasPrefix(name, ".") || // Unix-like hidden files
		strings.HasSuffix(name, "~") || // vim
		strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#") // emacs
}

func ParseModuleManifest(mod Module) {
	m := mod.(*module)
	m.SetModuleManifestParsingState(OpStateLoading)
	defer m.SetModuleManifestParsingState(OpStateLoaded)

	manifestPath, ok := datadir.ModuleManifestFilePath(m.fs, mod.Path())
	if !ok {
		m.logger.Printf("%s: manifest file does not exist", mod.Path())
		return
	}

	mm, err := datadir.ParseModuleManifestFromFile(manifestPath)
	if err != nil {
		m.logger.Printf("failed to parse manifest: %s", err)
	} else {
		m.logger.Printf("manifest parsed successfully for %s", m.Path())
	}

	m.SetModuleManifest(mm, err)
}
