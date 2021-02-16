package main

import (
	"fmt"

	goversion "github.com/hashicorp/go-version"
)

// The main version number that is being run at the moment.
var version = "0.0.0"

// A pre-release marker for the version. If this is "" (empty string)
// then it means that it is a final release. Otherwise, this is a pre-release
// such as "dev" (in development), "beta", "rc1", etc.
var prerelease = "dev"

var (
	buildGoVersion = ""
	buildGoOS      = ""
	buildGoArch    = ""
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
	if prerelease != "" {
		return buildInfo(fmt.Sprintf("%s-%s", version, prerelease))
	}

	return buildInfo(version)
}
func buildInfo(version string) string {
	if buildGoVersion != "" && buildGoOS != "" && buildGoArch != "" {
		return fmt.Sprintf("%s\ngo%s %s/%s", version, buildGoVersion, buildGoOS, buildGoArch)
	}
	return version
}
