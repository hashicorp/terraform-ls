// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	_ "embed"
	"strings"

	goversion "github.com/hashicorp/go-version"
)

var (
	// The next version number that will be released. This will be updated after every release
	// Version must conform to the format expected by github.com/hashicorp/go-version
	// for tests to work.
	// A pre-release marker for the version can also be specified (e.g -dev). If this is omitted
	// then it means that it is a final release. Otherwise, this is a pre-release
	// such as "dev" (in development), "beta", "rc1", etc.
	//go:embed version/VERSION
	rawVersion string

	version = goversion.Must(goversion.NewVersion(strings.TrimSpace(rawVersion)))
)

// VersionString returns the complete version string, including prerelease
func VersionString() string {
	return version.String()
}
