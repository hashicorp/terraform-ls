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

// semVer is an instance of version.Version. This has the secondary
// benefit of verifying during tests and init time that our version is a
// proper semantic version, which should always be the case.
var semVer *goversion.Version

func init() {
	var err error
	semVer, err = goversion.NewVersion(version)
	if err != nil {
		panic(err.Error())
	}
}

// VersionString returns the complete version string, including prerelease
func VersionString() string {
	if prerelease != "" {
		return fmt.Sprintf("%s-%s", version, prerelease)
	}
	return version
}
