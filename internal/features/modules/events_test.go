// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package modules

import (
	"path/filepath"
	"testing"
)

func TestUniqueModulePaths(t *testing.T) {
	root := t.TempDir()
	sharedPath := filepath.Join(root, "shared")
	paths := []string{
		filepath.Join(root, "first", "..", "shared"),
		filepath.Join(root, "second"),
		sharedPath,
	}

	seen := make([]string, 0, len(paths))
	for _, path := range paths {
		if containsModulePath(seen, path) {
			continue
		}
		seen = append(seen, path)
	}

	want := []string{sharedPath, filepath.Join(root, "second")}
	if len(seen) != len(want) {
		t.Fatalf("expected %d unique module paths, got %d: %#v", len(want), len(seen), seen)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("unexpected module path at index %d: got %q, want %q", i, seen[i], want[i])
		}
	}
}
