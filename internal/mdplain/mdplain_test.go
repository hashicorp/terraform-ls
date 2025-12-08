// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package mdplain_test

import (
	"testing"

	"github.com/hashicorp/terraform-ls/internal/mdplain"
)

func TestClean(t *testing.T) {
	for _, c := range []struct {
		markdown string
		expected string
	}{
		{"", ""},

		{"_foo_", "foo"},
		{"__foo__", "foo"},
		{"foo_bar", "foo_bar"},

		{"*foo*", "foo"},
		{"**foo**", "foo"},
		{"Desc **2**", "Desc 2"},
		{"1 * 3 = 3", "1 * 3 = 3"},

		{"## Header", "Header"},
		{"Header\n====\n\nSome text", "Header\n\nSome text"},

		{"* item 1\n* item 2\n\n\nSome text", "* item 1\n* item 2\n\nSome text"},
	} {
		t.Run(c.expected, func(t *testing.T) {
			actual := mdplain.Clean(c.markdown)

			if c.expected != actual {
				t.Fatalf("expected:\n%s\n\ngot:\n%s\n", c.expected, actual)
			}
		})
	}
}
