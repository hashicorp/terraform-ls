// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/document"
)

func TestStateNotChangedErr_Is(t *testing.T) {
	fooErr := StateNotChangedErr{Dir: document.DirHandleFromPath("/foo")}
	barErr := StateNotChangedErr{Dir: document.DirHandleFromPath("/bar")}

	testCases := []struct {
		name        string
		err         error
		target      error
		expectMatch bool
	}{
		{
			name:        "matches zero value regardless of directory",
			err:         fooErr,
			target:      StateNotChangedErr{},
			expectMatch: true,
		},
		{
			name:        "matches sentinel for a different directory",
			err:         fooErr,
			target:      barErr,
			expectMatch: true,
		},
		{
			name:        "matches sentinel for the same directory",
			err:         fooErr,
			target:      fooErr,
			expectMatch: true,
		},
		{
			name:        "matches when wrapped",
			err:         fmt.Errorf("loading metadata: %w", fooErr),
			target:      StateNotChangedErr{},
			expectMatch: true,
		},
		{
			name:        "does not match an unrelated error",
			err:         errors.New("something actually went wrong"),
			target:      StateNotChangedErr{},
			expectMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if match := errors.Is(tc.err, tc.target); match != tc.expectMatch {
				t.Fatalf("expected errors.Is to return %t, given err=%q target=%q",
					tc.expectMatch, tc.err, tc.target)
			}
		})
	}
}
