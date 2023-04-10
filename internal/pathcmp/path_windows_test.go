// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
			"file path the same",
			`c:\Users\user\Documents\tf`,
			`c:\Users\user\Documents\tf`,
			true,
		},
		{
			"volume case insensitive",
			`c:\Users\user\Documents\tf`,
			`C:\Users\user\Documents\tf`,
			true,
		},
		{
			"path folder case different",
			`c:\Users\user\Documents\tf`,
			`c:\Users\user\documents\tf`,
			false,
		},
		{
			"file path different",
			`c:\Users\user\Documents\tf`,
			`c:\Users\user\Documents\tf\test`,
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
