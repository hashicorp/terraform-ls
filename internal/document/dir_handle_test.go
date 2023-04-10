// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package document

import (
	"fmt"
	"testing"
)

func TestDirHandleFromURI(t *testing.T) {
	type testCase struct {
		RawURI         string
		ExpectedHandle DirHandle
	}

	testCases := []testCase{
		{
			RawURI: "file:///random/path",
			ExpectedHandle: DirHandle{
				URI: "file:///random/path",
			},
		},
		{
			RawURI: "file:///C:/random/path",
			ExpectedHandle: DirHandle{
				URI: "file:///C:/random/path",
			},
		},
		{
			RawURI: "file:///C%3A/random/path",
			ExpectedHandle: DirHandle{
				URI: "file:///C:/random/path",
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			h := DirHandleFromURI(tc.RawURI)
			if h != tc.ExpectedHandle {
				t.Fatalf("expected handle: %#v, given: %#v", tc.ExpectedHandle, h)
			}
		})
	}
}
