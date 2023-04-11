// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package datadir

import (
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/zclconf/go-cty/cty"
)

var pluginLockFilePathElements = [][]string{
	// Terraform >= 0.14
	{".terraform.lock.hcl"},
	// Terraform >= v0.13
	{DataDirName, "plugins", "selections.json"},
	// Terraform >= v0.12
	{DataDirName, "plugins", runtime.GOOS + "_" + runtime.GOARCH, "lock.json"},
}

func PluginLockFilePath(fs fs.StatFS, modPath string) (string, bool) {
	for _, pathElems := range pluginLockFilePathElements {
		fullPath := filepath.Join(append([]string{modPath}, pathElems...)...)
		fi, err := fs.Stat(fullPath)
		if err == nil && fi.Mode().IsRegular() {
			return fullPath, true
		}
	}

	return "", false
}

type PluginVersionMap map[tfaddr.Provider]*version.Version

type FS interface {
	ReadFile(name string) ([]byte, error)
	ReadDir(name string) ([]fs.DirEntry, error)
}

func ParsePluginVersions(filesystem FS, modPath string) (PluginVersionMap, error) {
	pvm, err := parsePluginLockFile_v014(filesystem, modPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	} else {
		return pvm, nil
	}

	pvm, err = parsePluginLockFile_v013(filesystem, modPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	} else {
		return pvm, nil
	}

	return parsePluginDir_v012(filesystem, modPath)
}

// parsePluginDir_v012 parses the 0.12-style datadir.
// See https://github.com/hashicorp/terraform/blob/v0.12.0/plugin/discovery/find.go#L45
func parsePluginDir_v012(filesystem FS, modPath string) (PluginVersionMap, error) {
	// Unfortunately the lock.json from 0.12 only contains hashes, not versions
	// so we have to imply the versions from filenames (which is what Terraform 0.12 does too)
	dirPath := filepath.Join(modPath, DataDirName, "plugins", runtime.GOOS+"_"+runtime.GOARCH)
	entries, err := filesystem.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	// terraform-provider-aws_v4.23.0_x5
	filenameRe := regexp.MustCompile(`^terraform-provider-([^_]+)_(v[^_]+)`)

	pvm := make(PluginVersionMap, 0)
	for _, entry := range entries {
		name := entry.Name()

		matches := filenameRe.FindStringSubmatch(name)
		if len(matches) != 3 {
			continue
		}

		providerName, err := tfaddr.ParseProviderPart(matches[1])
		if err != nil {
			continue
		}
		providerVersion, err := version.NewVersion(matches[2])
		if err != nil {
			continue
		}

		pvm[legacyProviderAddr(providerName)] = providerVersion
	}

	return pvm, nil
}

func legacyProviderAddr(name string) tfaddr.Provider {
	return tfaddr.Provider{
		Hostname:  tfaddr.DefaultProviderRegistryHost,
		Namespace: tfaddr.LegacyProviderNamespace,
		Type:      name,
	}
}

func parsePluginLockFile_v013(filesystem FS, modPath string) (PluginVersionMap, error) {
	fullPath := filepath.Join(modPath, DataDirName, "plugins", "selections.json")

	src, err := filesystem.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	file := selectionFile{}
	err = json.Unmarshal(src, &file)
	if err != nil {
		return nil, err
	}

	pvm := make(PluginVersionMap, 0)
	for rawAddress, sel := range file {
		pAddr, err := tfaddr.ParseProviderSource(rawAddress)
		if err != nil {
			continue
		}
		pvm[pAddr] = sel.Version
	}

	return pvm, nil
}

type selectionFile map[string]selection

type selection struct {
	Version *version.Version
}

func parsePluginLockFile_v014(filesystem FS, modPath string) (PluginVersionMap, error) {
	fullPath := filepath.Join(modPath, ".terraform.lock.hcl")

	src, err := filesystem.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	cfg, diags := hclsyntax.ParseConfig(src, ".terraform.lock.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	// We precautiosly use PartialContent, to avoid this breaking
	// in case Terraform CLI introduces new blocks
	body, _, diags := cfg.Body.PartialContent(lockFileSchema)
	if diags.HasErrors() {
		return nil, diags
	}

	pvm := make(PluginVersionMap, 0)
	for _, block := range body.Blocks.OfType("provider") {
		if len(block.Labels) != 1 {
			continue
		}

		pAddr, err := tfaddr.ParseProviderSource(block.Labels[0])
		if err != nil {
			continue
		}

		pBody, _, diags := block.Body.PartialContent(providerSchema)
		if diags.HasErrors() {
			continue
		}

		val, diags := pBody.Attributes["version"].Expr.Value(nil)
		if diags.HasErrors() {
			continue
		}
		if val.Type() != cty.String {
			continue
		}
		pVersion, err := version.NewVersion(val.AsString())
		if err != nil {
			continue
		}

		pvm[pAddr] = pVersion
	}

	return pvm, nil
}

var lockFileSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "provider",
			LabelNames: []string{"source"},
		},
	},
}

var providerSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name:     "version",
			Required: true,
		},
	},
}
