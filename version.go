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
	fullVersion string

	version, versionPrerelease, _ = strings.Cut(fullVersion, "-")

	// https://semver.org/#spec-item-10
	versionMetadata = ""
)

func init() {
	// Verify that the version is proper semantic version, which should always be the case.
	_, err := goversion.NewVersion(version)
	if err != nil {
		panic(err.Error())
	}
}

// VersionString returns the complete version string, including prerelease
func VersionString() string {
	v := version
	if versionPrerelease != "" {
		v += "-" + versionPrerelease
	}

	if versionMetadata != "" {
		v += "+" + versionMetadata
	}

	return v
}
