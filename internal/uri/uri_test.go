// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package uri

import (
	"fmt"
	"runtime"
	"testing"
)

func TestIsURIValid_invalid(t *testing.T) {
	uri := "output:extension-output-%232"
	if IsURIValid(uri) {
		t.Fatalf("Expected %q to be invalid", uri)
	}
}

func TestFromPath(t *testing.T) {
	type testCase struct {
		RawPath     string
		ExpectedURI string
	}
	testCases := []testCase{}

	if runtime.GOOS == "windows" {
		// windows
		testCases = []testCase{
			{
				RawPath:     `C:\Users\With Space\file.tf`,
				ExpectedURI: "file:///C:/Users/With%20Space/file.tf",
			},
			{
				RawPath:     `C:\Users\WithoutSpace\file.tf`,
				ExpectedURI: "file:///C:/Users/WithoutSpace/file.tf",
			},
			{
				RawPath:     `C:\Users\TrailingSeparator\`,
				ExpectedURI: "file:///C:/Users/TrailingSeparator",
			},
			// Ensure any-cased drive letter (which may come from VS Code) is uppercased
			{
				RawPath:     `c:\test`,
				ExpectedURI: "file:///C:/test",
			},
		}
	} else {
		// unix
		testCases = []testCase{
			{
				RawPath:     "/random/path/with space",
				ExpectedURI: "file:///random/path/with%20space",
			},
			{
				RawPath:     "/random/path",
				ExpectedURI: "file:///random/path",
			},
			{
				RawPath:     `/path/with/trailing-separator/`,
				ExpectedURI: "file:///path/with/trailing-separator",
			},
		}
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			uri := FromPath(tc.RawPath)
			if uri != tc.ExpectedURI {
				t.Fatalf("URI doesn't match.\nExpected: %q\nGiven: %q",
					tc.ExpectedURI, uri)
			}
		})
	}
}

func TestPathFromURI(t *testing.T) {
	type testCase struct {
		URI          string
		ExpectedPath string
	}
	testCases := []testCase{}

	if runtime.GOOS == "windows" {
		// windows
		testCases = []testCase{
			{
				URI:          "file:///C:/Users/With%20Space/tf-test/file.tf",
				ExpectedPath: `C:\Users\With Space\tf-test\file.tf`,
			},
			{
				URI:          "file:///C:/Users/With%20Space/tf-test",
				ExpectedPath: `C:\Users\With Space\tf-test`,
			},
			// Ensure URI with trailing slash is trimmed per RFC 3986 § 6.2.4
			{
				URI:          "file:///C:/Users/Test/tf-test/",
				ExpectedPath: `C:\Users\Test\tf-test`,
			},
			// Ensure over-escaped colon (which may come from VS Code) is normalized
			{
				URI:          "file:///C%3A/Users/With%20Space/tf-test",
				ExpectedPath: `C:\Users\With Space\tf-test`,
			},
			// Ensure any-cased drive letter (which may come from VS Code) is uppercased
			{
				URI:          "file:///c:/tf-test",
				ExpectedPath: `C:\tf-test`,
			},
		}
	} else {
		// unix
		testCases = []testCase{
			{
				URI:          "file:///valid/path/to/file.tf",
				ExpectedPath: "/valid/path/to/file.tf",
			},
			{
				URI:          "file:///valid/path/to",
				ExpectedPath: "/valid/path/to",
			},

			// Ensure URI with trailing slash is trimmed per RFC 3986 § 6.2.4
			{
				URI:          "file:///random/dir/",
				ExpectedPath: "/random/dir",
			},
		}
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if !IsURIValid(tc.URI) {
				t.Fatalf("Expected %q to be valid", tc.URI)
			}

			path, err := PathFromURI(tc.URI)
			if err != nil {
				t.Fatal(err)
			}
			if path != tc.ExpectedPath {
				t.Fatalf("Expected full path: %q, given: %q",
					tc.ExpectedPath, path)
			}
		})
	}
}

func TestMustParseURI(t *testing.T) {
	type testCase struct {
		RawURI      string
		ExpectedURI string
	}

	testCases := []testCase{
		{
			RawURI:      "file:///C:/Users/With Space/tf-test/file.tf",
			ExpectedURI: "file:///C:/Users/With%20Space/tf-test/file.tf",
		},
		{
			RawURI:      "file:///C:/Users/With%20Space/tf-test/file.tf",
			ExpectedURI: "file:///C:/Users/With%20Space/tf-test/file.tf",
		},
		// Ensure over-escaped colon (which may come from VS Code) is normalized
		{
			RawURI:      "file:///C%3A/Users/With%20Space/tf-test/file.tf",
			ExpectedURI: "file:///C:/Users/With%20Space/tf-test/file.tf",
		},
		// Ensure any-cased drive letter (which may come from VS Code) is uppercased
		{
			RawURI:      "file:///c:/tf-test",
			ExpectedURI: "file:///C:/tf-test",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			uri := MustParseURI(tc.RawURI)
			if tc.ExpectedURI != uri {
				t.Fatalf("Expected %q, given %q", tc.ExpectedURI, uri)
			}
		})
	}
}

func TestIsWSLURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want bool
	}{
		{
			name: "UNC WSL Path should return true",
			uri:  `file://wsl%24/Ubuntu/home/james/some/path`,
			want: true,
		},
		{
			name: "Regular file path should return false",
			uri:  `file://C:/foo/james/foo`,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWSLURI(tt.uri); got != tt.want {
				t.Errorf("IsWSLURI() = %v, want %v", got, tt.want)
			}
		})
	}
}
