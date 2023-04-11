// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !windows
// +build !windows

package pathcmp

import (
	"fmt"
	"testing"
)

func TestPathEquals(t *testing.T) {
	testCases := []struct {
		name     string
		path1    string
		path2    string
		expected bool
	}{
		{
			"path the same",
			`/home/user/documents/tf`,
			`/home/user/documents/tf`,
			true,
		},
		{
			"path case not the same",
			`/home/user/documents/tf`,
			`/Home/user/documents/tf`,
			false,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			result := PathEquals(tc.path1, tc.path2)
			if result != tc.expected {
				t.Fatalf("expected: %t Got: %t", tc.expected, result)
			}
		})
	}
}
