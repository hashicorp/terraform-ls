// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package document

import (
	"fmt"
	"runtime"
	"testing"
)

func TestHandleFromURI(t *testing.T) {
	testCases := []struct {
		RawURI         string
		ExpectedHandle Handle
	}{
		{
			RawURI: "file:///random/path/to/config.tf",
			ExpectedHandle: Handle{
				Dir:      DirHandle{URI: "file:///random/path/to"},
				Filename: "config.tf",
			},
		},
		{
			RawURI: "file:///C:/random/path/to/config.tf",
			ExpectedHandle: Handle{
				Dir:      DirHandle{URI: "file:///C:/random/path/to"},
				Filename: "config.tf",
			},
		},
		{
			RawURI: "file:///C%3A/random/path/to/config.tf",
			ExpectedHandle: Handle{
				Dir:      DirHandle{URI: "file:///C:/random/path/to"},
				Filename: "config.tf",
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			h := HandleFromURI(tc.RawURI)
			if h != tc.ExpectedHandle {
				t.Fatalf("expected handle: %#v, given: %#v", tc.ExpectedHandle, h)
			}
		})
	}
}

func TestHandleFromPath(t *testing.T) {
	type testCase struct {
		RawURI         string
		ExpectedHandle Handle
	}

	testCases := []testCase{}
	if runtime.GOOS == "windows" {
		testCases = []testCase{
			{
				RawURI: `C:\random\path\to\config.tf`,
				ExpectedHandle: Handle{
					Dir:      DirHandle{URI: "file:///C:/random/path/to"},
					Filename: "config.tf",
				},
			},
		}
	} else {
		testCases = []testCase{
			{
				RawURI: "/random/path/to/config.tf",
				ExpectedHandle: Handle{
					Dir:      DirHandle{URI: "file:///random/path/to"},
					Filename: "config.tf",
				},
			},
		}
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			h := HandleFromPath(tc.RawURI)
			if h != tc.ExpectedHandle {
				t.Fatalf("expected handle: %#v, given: %#v", tc.ExpectedHandle, h)
			}
		})
	}
}

func TestDirHandleFromPath(t *testing.T) {
	type testCase struct {
		RawURI         string
		ExpectedHandle DirHandle
	}

	testCases := []testCase{}
	if runtime.GOOS == "windows" {
		testCases = []testCase{
			{
				RawURI:         `C:\random\path\to`,
				ExpectedHandle: DirHandle{URI: "file:///C:/random/path/to"},
			},
			{
				RawURI:         `C:\random\path\to\`,
				ExpectedHandle: DirHandle{URI: "file:///C:/random/path/to"},
			},
		}
	} else {
		testCases = []testCase{
			{
				RawURI:         "/random/path/to",
				ExpectedHandle: DirHandle{URI: "file:///random/path/to"},
			},
			{
				RawURI:         "/random/path/to/",
				ExpectedHandle: DirHandle{URI: "file:///random/path/to"},
			},
		}
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			h := DirHandleFromPath(tc.RawURI)
			if h != tc.ExpectedHandle {
				t.Fatalf("expected handle: %#v, given: %#v", tc.ExpectedHandle, h)
			}
		})
	}
}

func TestHandle_FullURI(t *testing.T) {
	type testCase struct {
		Handle      Handle
		ExpectedURI string
	}

	testCases := []testCase{
		{
			Handle: Handle{
				Dir:      DirHandle{URI: "file:///C:/random/path/to"},
				Filename: "config.tf",
			},
			ExpectedURI: "file:///C:/random/path/to/config.tf",
		},
		{
			Handle: Handle{
				Dir:      DirHandle{URI: "file:///random/path/to"},
				Filename: "config.tf",
			},
			ExpectedURI: "file:///random/path/to/config.tf",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if tc.ExpectedURI != tc.Handle.FullURI() {
				t.Fatalf("expected URI: %#v, given: %#v", tc.ExpectedURI, tc.Handle.FullURI())
			}
		})
	}
}

func TestHandle_FullPath(t *testing.T) {
	type testCase struct {
		Handle       Handle
		ExpectedPath string
	}

	testCases := []testCase{}
	if runtime.GOOS == "windows" {
		testCases = []testCase{
			{
				Handle: Handle{
					Dir:      DirHandle{URI: "file:///C:/random/path/to"},
					Filename: "config.tf",
				},
				ExpectedPath: `C:\random\path\to\config.tf`,
			},
		}
	} else {
		testCases = []testCase{
			{
				Handle: Handle{
					Dir:      DirHandle{URI: "file:///random/path/to"},
					Filename: "config.tf",
				},
				ExpectedPath: "/random/path/to/config.tf",
			},
		}
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if tc.ExpectedPath != tc.Handle.FullPath() {
				t.Fatalf("expected path: %#v, given: %#v", tc.ExpectedPath, tc.Handle.FullPath())
			}
		})
	}
}
