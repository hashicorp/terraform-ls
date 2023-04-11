// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package source

import (
	"testing"
)

func TestMakeSourceLines_empty(t *testing.T) {
	lines := MakeSourceLines("/test.tf", []byte{})
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line from empty file, %d parsed:\n%#v",
			len(lines), lines)
	}
}

func TestMakeSourceLines_success(t *testing.T) {
	lines := MakeSourceLines("/test.tf", []byte("\n\n\n\n"))
	expectedLines := 5
	if len(lines) != expectedLines {
		t.Fatalf("Expected exactly %d lines, %d parsed",
			expectedLines, len(lines))
	}
}
