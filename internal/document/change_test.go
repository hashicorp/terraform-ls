// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package document

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestApplyChanges_fullUpdate(t *testing.T) {
	original := []byte("hello world")

	changes := []Change{
		&testChange{text: "something else"},
	}

	given, err := ApplyChanges(original, changes)
	if err != nil {
		t.Fatal(err)
	}

	expectedText := "something else"
	if diff := cmp.Diff(expectedText, string(given)); diff != "" {
		t.Fatalf("content mismatch: %s", diff)
	}
}

func TestApplyChanges_partialUpdate(t *testing.T) {
	testCases := []struct {
		Name     string
		Original string
		Change   *testChange
		Expect   string
	}{
		{
			Name:     "length grow: 4",
			Original: "hello world",
			Change: &testChange{
				text: "terraform",
				rng: &Range{
					Start: Pos{
						Line:   0,
						Column: 6,
					},
					End: Pos{
						Line:   0,
						Column: 11,
					},
				},
			},
			Expect: "hello terraform",
		},
		{
			Name:     "length the same",
			Original: "hello world",
			Change: &testChange{
				text: "earth",
				rng: &Range{
					Start: Pos{
						Line:   0,
						Column: 6,
					},
					End: Pos{
						Line:   0,
						Column: 11,
					},
				},
			},
			Expect: "hello earth",
		},
		{
			Name:     "length grow: -2",
			Original: "hello world",
			Change: &testChange{
				text: "HCL",
				rng: &Range{
					Start: Pos{
						Line:   0,
						Column: 6,
					},
					End: Pos{
						Line:   0,
						Column: 11,
					},
				},
			},
			Expect: "hello HCL",
		},
		{
			Name:     "zero-length range",
			Original: "hello world",
			Change: &testChange{
				text: "abc ",
				rng: &Range{
					Start: Pos{
						Line:   0,
						Column: 6,
					},
					End: Pos{
						Line:   0,
						Column: 6,
					},
				},
			},
			Expect: "hello abc world",
		},
		{
			Name:     "add utf-18 character",
			Original: "hello world",
			Change: &testChange{
				text: "𐐀𐐀 ",
				rng: &Range{
					Start: Pos{
						Line:   0,
						Column: 6,
					},
					End: Pos{
						Line:   0,
						Column: 6,
					},
				},
			},
			Expect: "hello 𐐀𐐀 world",
		},
		{
			Name:     "modify when containing utf-18 character",
			Original: "hello 𐐀𐐀 world",
			Change: &testChange{
				text: "aa𐐀",
				rng: &Range{
					Start: Pos{
						Line:   0,
						Column: 8,
					},
					End: Pos{
						Line:   0,
						Column: 10,
					},
				},
			},
			Expect: "hello 𐐀aa𐐀 world",
		},
	}

	for _, tc := range testCases {
		changes := []Change{tc.Change}

		given, err := ApplyChanges([]byte(tc.Original), changes)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(tc.Expect, string(given)); diff != "" {
			t.Fatalf("%s: content mismatch: %s", tc.Name, diff)
		}
	}
}

func TestApplyChanges_partialUpdateMultipleChanges(t *testing.T) {
	testCases := []struct {
		Original string
		Changes  Changes
		Expect   string
	}{
		{
			Original: `variable "service_host" {
  default = "blah"
}

module "app" {
  source = "./sub"
  service_listeners = [
    {
      hosts    = [var.service_host]
      listener = ""
    }
  ]
}
`,
			Changes: Changes{
				&testChange{
					text: "\n",
					rng: &Range{
						Start: Pos{Line: 8, Column: 18},
						End:   Pos{Line: 8, Column: 18},
					},
				},
				&testChange{
					text: "      ",
					rng: &Range{
						Start: Pos{Line: 9, Column: 0},
						End:   Pos{Line: 9, Column: 0},
					},
				},
				&testChange{
					text: "  ",
					rng: &Range{
						Start: Pos{Line: 9, Column: 6},
						End:   Pos{Line: 9, Column: 6},
					},
				},
			},
			Expect: `variable "service_host" {
  default = "blah"
}

module "app" {
  source = "./sub"
  service_listeners = [
    {
      hosts    = [
        var.service_host]
      listener = ""
    }
  ]
}
`,
		},
	}

	for _, tc := range testCases {
		given, err := ApplyChanges([]byte(tc.Original), tc.Changes)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(tc.Expect, string(given)); diff != "" {
			t.Fatalf("content mismatch: %s", diff)
		}
	}
}

type testChange struct {
	text string
	rng  *Range
}

func (fc *testChange) Text() string {
	return fc.text
}

func (fc *testChange) Range() *Range {
	return fc.rng
}
