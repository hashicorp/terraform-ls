// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import "testing"

var (
	_ ModuleReader = &ModuleStore{}
	_ SchemaReader = &ProviderSchemaStore{}
)

func TestDbSchema_Validate(t *testing.T) {
	err := dbSchema.Validate()
	if err != nil {
		t.Fatal(err)
	}
}
