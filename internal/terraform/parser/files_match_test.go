// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package parser

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hashicorp/hcl/v2"
)

// testFilename stands in for the filename types the features key their
// parsed files by.
type testFilename string

func (n testFilename) String() string {
	return string(n)
}

func TestFilesMatch(t *testing.T) {
	isTf := func(name string) bool {
		return strings.HasSuffix(name, ".tf")
	}
	parsed := map[testFilename]*hcl.File{
		"main.tf":      {Bytes: []byte("main")},
		"variables.tf": {Bytes: []byte("variables")},
	}

	testCases := []struct {
		name  string
		files fstest.MapFS
		want  bool
	}{
		{
			name: "same files and content",
			files: fstest.MapFS{
				"mod/main.tf":      {Data: []byte("main")},
				"mod/variables.tf": {Data: []byte("variables")},
			},
			want: true,
		},
		{
			name: "files the filter does not accept are ignored",
			files: fstest.MapFS{
				"mod/main.tf":        {Data: []byte("main")},
				"mod/variables.tf":   {Data: []byte("variables")},
				"mod/README.md":      {Data: []byte("readme")},
				"mod/nested/main.tf": {Data: []byte("nested")},
			},
			want: true,
		},
		{
			name: "content changed",
			files: fstest.MapFS{
				"mod/main.tf":      {Data: []byte("main changed")},
				"mod/variables.tf": {Data: []byte("variables")},
			},
			want: false,
		},
		{
			name: "file added",
			files: fstest.MapFS{
				"mod/main.tf":      {Data: []byte("main")},
				"mod/variables.tf": {Data: []byte("variables")},
				"mod/outputs.tf":   {Data: []byte("outputs")},
			},
			want: false,
		},
		{
			name: "file removed",
			files: fstest.MapFS{
				"mod/main.tf": {Data: []byte("main")},
			},
			want: false,
		},
		{
			name:  "directory missing",
			files: fstest.MapFS{},
			want:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := FilesMatch(tc.files, "mod", isTf, parsed)
			if got != tc.want {
				t.Fatalf("FilesMatch() = %t, want %t", got, tc.want)
			}
		})
	}
}
