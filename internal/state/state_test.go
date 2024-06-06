// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import "testing"

func TestDbSchema_Validate(t *testing.T) {
	err := dbSchema.Validate()
	if err != nil {
		t.Fatal(err)
	}
}
