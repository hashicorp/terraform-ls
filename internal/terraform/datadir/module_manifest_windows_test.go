// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package datadir

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

func TestRecord_UnmarshalJSON_basic(t *testing.T) {
	var record ModuleRecord
	err := json.Unmarshal([]byte(moduleManifestRecord_external), &record)
	if err != nil {
		t.Fatal(err)
	}

	expectedVersion, err := version.NewVersion("3.10.0")
	if err != nil {
		t.Fatal(err)
	}
	expectedRecord := ModuleRecord{
		Key:           "web_server_sg",
		SourceAddr:    tfmod.ParseModuleSourceAddr("terraform-aws-modules/security-group/aws//modules/http-80"),
		RawSourceAddr: "terraform-aws-modules/security-group/aws//modules/http-80",
		VersionStr:    "3.10.0",
		Version:       expectedVersion,
		Dir:           `.terraform\modules\web_server_sg\terraform-aws-security-group-3.10.0\modules\http-80`,
	}
	if diff := cmp.Diff(expectedRecord, record); diff != "" {
		t.Fatalf("version mismatch: %s", diff)
	}
}

func TestRecord_UnmarshalJSON_dirtyPath(t *testing.T) {
	var record ModuleRecord
	err := json.Unmarshal([]byte(moduleManifestRecord_externalDirtyPath), &record)
	if err != nil {
		t.Fatal(err)
	}

	expectedDir := `.terraform\modules\web_server_sg\terraform-aws-security-group-3.10.0\modules\http-80`
	if expectedDir != record.Dir {
		t.Fatalf("expected dir: %s, given: %s", expectedDir, record.Dir)
	}
}

func TestRecord_UnmarshalJSON_isExternal(t *testing.T) {
	var localRecord ModuleRecord
	err := json.Unmarshal([]byte(moduleManifestRecord_local), &localRecord)
	if err != nil {
		t.Fatal(err)
	}

	localExpected := false
	localGiven := localRecord.IsExternal()
	if localExpected != localGiven {
		t.Fatalf("expected IsExternal(): %t, given: %t", localExpected, localGiven)
	}

	var extRecord ModuleRecord
	err = json.Unmarshal([]byte(moduleManifestRecord_external), &extRecord)
	if err != nil {
		t.Fatal(err)
	}

	extExpected := true
	extGiven := extRecord.IsExternal()
	if extExpected != extGiven {
		t.Fatalf("expected IsExternal(): %t, given: %t", extExpected, extGiven)
	}
}

func TestRecord_UnmarshalJSON_isRoot(t *testing.T) {
	var rootRecord ModuleRecord
	err := json.Unmarshal([]byte(moduleManifestRecord_root), &rootRecord)
	if err != nil {
		t.Fatal(err)
	}

	rootExpected := true
	rootGiven := rootRecord.IsRoot()
	if rootExpected != rootGiven {
		t.Fatalf("expected IsRoot(): %t, given: %t", rootExpected, rootGiven)
	}

	var extRecord ModuleRecord
	err = json.Unmarshal([]byte(moduleManifestRecord_external), &extRecord)
	if err != nil {
		t.Fatal(err)
	}

	extExpected := false
	extGiven := extRecord.IsRoot()
	if extExpected != extGiven {
		t.Fatalf("expected IsRoot(): %t, given: %t", extExpected, extGiven)
	}
}
