// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package datadir

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

func TestParseModuleManifestFromFile(t *testing.T) {
	modPath := t.TempDir()
	manifestDir := filepath.Join(modPath, ".terraform", "modules")
	err := os.MkdirAll(manifestDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	expectedManifest := &ModuleManifest{
		rootDir: modPath,
		Records: []ModuleRecord{
			{
				Key:           "web_server_sg1",
				SourceAddr:    tfmod.ParseModuleSourceAddr("terraform-aws-modules/security-group/aws//modules/http-80"),
				RawSourceAddr: "terraform-aws-modules/security-group/aws//modules/http-80",
				VersionStr:    "3.10.0",
				Version:       version.Must(version.NewVersion("3.10.0")),
				Dir:           filepath.Join(".terraform", "modules", "web_server_sg", "terraform-aws-security-group-3.10.0", "modules", "http-80"),
			},
			{
				Key:           "web_server_sg2",
				SourceAddr:    tfmod.ParseModuleSourceAddr("terraform-aws-modules/security-group/aws//modules/http-80"),
				RawSourceAddr: "terraform-aws-modules/security-group/aws//modules/http-80",
				VersionStr:    "3.10.0",
				Version:       version.Must(version.NewVersion("3.10.0")),
				Dir:           filepath.Join(".terraform", "modules", "web_server_sg", "terraform-aws-security-group-3.10.0", "modules", "http-80"),
			},
			{
				Dir: ".",
			},
			{
				Key:           "local",
				SourceAddr:    tfmod.ParseModuleSourceAddr("./nested/path"),
				RawSourceAddr: "./nested/path",
				Dir:           filepath.Join("nested", "path"),
			},
		},
	}

	path := filepath.Join(manifestDir, "modules.json")
	err = ioutil.WriteFile(path, []byte(testManifestContent), 0755)
	if err != nil {
		t.Fatal(err)
	}
	mm, err := ParseModuleManifestFromFile(path)
	if err != nil {
		t.Fatal(err)
	}

	opts := cmp.AllowUnexported(ModuleManifest{})
	if diff := cmp.Diff(expectedManifest, mm, opts); diff != "" {
		t.Fatalf("manifest mismatch: %s", diff)
	}
}

const testManifestContent = `{
    "Modules": [
        {
            "Key": "web_server_sg1",
            "Source": "terraform-aws-modules/security-group/aws//modules/http-80",
            "Version": "3.10.0",
            "Dir": ".terraform/modules/web_server_sg/terraform-aws-security-group-3.10.0/modules/http-80"
        },
        {
            "Key": "web_server_sg2",
            "Source": "terraform-aws-modules/security-group/aws//modules/http-80",
            "Version": "3.10.0",
            "Dir": ".terraform/modules/web_server_sg/terraform-aws-security-group-3.10.0/modules/something/../http-80"
        },
        {
            "Key": "",
            "Source": "",
            "Dir": "."
        },
        {
            "Key": "local",
            "Source": "./nested/path",
            "Dir": "nested/path"
        }
    ]
}`

const moduleManifestRecord_external = `{
    "Key": "web_server_sg",
    "Source": "terraform-aws-modules/security-group/aws//modules/http-80",
    "Version": "3.10.0",
    "Dir": ".terraform/modules/web_server_sg/terraform-aws-security-group-3.10.0/modules/http-80"
}`

const moduleManifestRecord_externalDirtyPath = `{
    "Key": "web_server_sg",
    "Source": "terraform-aws-modules/security-group/aws//modules/http-80",
    "Version": "3.10.0",
    "Dir": ".terraform/modules/web_server_sg/terraform-aws-security-group-3.10.0/modules/something/../http-80"
}`

const moduleManifestRecord_local = `{
    "Key": "local",
    "Source": "./nested/path",
    "Dir": "nested/path"
}`

const moduleManifestRecord_root = `{
    "Key": "",
    "Source": "",
    "Dir": "."
}`
