// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package datadir

import (
	"io/fs"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

func TestParsePluginVersions_basic012(t *testing.T) {
	// TODO: Replace OS-specific separator with '/'
	// See https://github.com/hashicorp/terraform-ls/issues/1025
	fs := fstest.MapFS{
		"foo-module": &fstest.MapFile{Mode: fs.ModeDir},
		filepath.Join("foo-module", ".terraform"):                                                                                       &fstest.MapFile{Mode: fs.ModeDir},
		filepath.Join("foo-module", ".terraform", "plugins"):                                                                            &fstest.MapFile{Mode: fs.ModeDir},
		filepath.Join("foo-module", ".terraform", "plugins", runtime.GOOS+"_"+runtime.GOARCH):                                           &fstest.MapFile{Mode: fs.ModeDir},
		filepath.Join("foo-module", ".terraform", "plugins", runtime.GOOS+"_"+runtime.GOARCH) + "/terraform-provider-aws_v4.23.0_x5":    &fstest.MapFile{},
		filepath.Join("foo-module", ".terraform", "plugins", runtime.GOOS+"_"+runtime.GOARCH) + "/terraform-provider-google_v4.29.0_x5": &fstest.MapFile{},
	}
	expectedVersions := PluginVersionMap{
		legacyProviderAddr("aws"):    version.Must(version.NewVersion("4.23.0")),
		legacyProviderAddr("google"): version.Must(version.NewVersion("4.29.0")),
	}
	versions, err := ParsePluginVersions(fs, "foo-module")
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(expectedVersions, versions); diff != "" {
		t.Fatalf("unexpected versions: %s", diff)
	}
}

func TestParsePluginVersions_basic013(t *testing.T) {
	fs := fstest.MapFS{
		"foo-module": &fstest.MapFile{Mode: fs.ModeDir},
		filepath.Join("foo-module", ".terraform"):            &fstest.MapFile{Mode: fs.ModeDir},
		filepath.Join("foo-module", ".terraform", "plugins"): &fstest.MapFile{Mode: fs.ModeDir},
		filepath.Join("foo-module", ".terraform", "plugins", "selections.json"): &fstest.MapFile{
			Data: []byte(`{
  "registry.terraform.io/hashicorp/aws": {
    "hash": "h1:j6RGCfnoLBpzQVOKUbGyxf4EJtRvQClKplO+WdXL5O0=",
    "version": "4.23.0"
  },
  "registry.terraform.io/hashicorp/google": {
    "hash": "h1:vZdocusWLMUSeRLI3W3dd3bgKYovGntsaHiXFIfM484=",
    "version": "4.29.0"
  }
}`),
		},
	}
	expectedVersions := PluginVersionMap{
		tfaddr.MustParseProviderSource("hashicorp/aws"):    version.Must(version.NewVersion("4.23.0")),
		tfaddr.MustParseProviderSource("hashicorp/google"): version.Must(version.NewVersion("4.29.0")),
	}
	versions, err := ParsePluginVersions(fs, "foo-module")
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(expectedVersions, versions); diff != "" {
		t.Fatalf("unexpected versions: %s", diff)
	}
}

func TestParsePluginVersions_basic014(t *testing.T) {
	fs := fstest.MapFS{
		"foo-module": &fstest.MapFile{Mode: fs.ModeDir},
		filepath.Join("foo-module", ".terraform.lock.hcl"): &fstest.MapFile{
			Data: []byte(`# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/aws" {
  version = "4.23.0"
  hashes = [
    "h1:j6RGCfnoLBpzQVOKUbGyxf4EJtRvQClKplO+WdXL5O0=",
    "zh:17adbedc9a80afc571a8de7b9bfccbe2359e2b3ce1fffd02b456d92248ec9294",
    "zh:23d8956b031d78466de82a3d2bbe8c76cc58482c931af311580b8eaef4e6a38f",
    "zh:343fe19e9a9f3021e26f4af68ff7f4828582070f986b6e5e5b23d89df5514643",
    "zh:6b8ff83d884b161939b90a18a4da43dd464c4b984f54b5f537b2870ce6bd94bc",
    "zh:7777d614d5e9d589ad5508eecf4c6d8f47d50fcbaf5d40fa7921064240a6b440",
    "zh:82f4578861a6fd0cde9a04a1926920bd72d993d524e5b34d7738d4eff3634c44",
    "zh:9b12af85486a96aedd8d7984b0ff811a4b42e3d88dad1a3fb4c0b580d04fa425",
    "zh:a08fefc153bbe0586389e814979cf7185c50fcddbb2082725991ed02742e7d1e",
    "zh:ae789c0e7cb777d98934387f8888090ccb2d8973ef10e5ece541e8b624e1fb00",
    "zh:b4608aab78b4dbb32c629595797107fc5a84d1b8f0682f183793d13837f0ecf0",
    "zh:ed2c791c2354764b565f9ba4be7fc845c619c1a32cefadd3154a5665b312ab00",
    "zh:f94ac0072a8545eebabf417bc0acbdc77c31c006ad8760834ee8ee5cdb64e743",
  ]
}

provider "registry.terraform.io/hashicorp/google" {
  version = "4.29.0"
  hashes = [
    "h1:vZdocusWLMUSeRLI3W3dd3bgKYovGntsaHiXFIfM484=",
    "zh:00ac3a2c7006d349147809961839be1ceda83d5c620aa30541064e2507b72f35",
    "zh:1602bdc71667abfbcc34c15944decabc5e05e167e49ce4045dc13ba234a27995",
    "zh:173c2fb837c9c1a9b103ca9f9ade456effc705a5539ddab2a7de0b1e3d59af73",
    "zh:231c28cc9698c9ce87218f9a8073dd30aa51b97511bf57e533b7780581cb2e4f",
    "zh:2423c1f8065b309fc7340b880fa898f877e715c734b5322c12d004335c7591d4",
    "zh:2c0d650520e32d8d884a4fb83cf3527605a8cadab557a0857290a3b14b85f6e5",
    "zh:8ef536b0cb362a377e058c4105d4748cd7c4b083376abc829ce8d66396c589c7",
    "zh:9da3e2987cd737b843f0a8558b400af1f0fe60929cd23788800a1114818d982d",
    "zh:ad727c5eba4cce83a44f3747637876462686465e64ac40099a084935a538bb57",
    "zh:b3895af9e06d0142ef5c6bbdd8dd0b2acb4dffa9c6631b9b6b984719c157cc1b",
    "zh:d7be31e59a254f952f4e03bedbf4dfbd6717f5e9e5d31e1add52711f6da4aedb",
    "zh:f569b65999264a9416862bca5cd2a6177d94ccb0424f3a4ef424428912b9cb3c",
  ]
}
`),
		},
	}
	expectedVersions := PluginVersionMap{
		tfaddr.MustParseProviderSource("hashicorp/aws"):    version.Must(version.NewVersion("4.23.0")),
		tfaddr.MustParseProviderSource("hashicorp/google"): version.Must(version.NewVersion("4.29.0")),
	}
	versions, err := ParsePluginVersions(fs, "foo-module")
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(expectedVersions, versions); diff != "" {
		t.Fatalf("unexpected versions: %s", diff)
	}
}
