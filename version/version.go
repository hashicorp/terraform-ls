package version

import (
	"fmt"
	"errors"

	version "github.com/hashicorp/go-version"
)

// The main version number that is being run at the moment.
var Version = "_"

// A pre-release marker for the version. If this is "" (empty string)
// then it means that it is a final release. Otherwise, this is a pre-release
// such as "dev" (in development), "beta", "rc1", etc.
var Prerelease = "dev"

// SemVer is an instance of version.Version. This has the secondary
// benefit of verifying during tests and init time that our version is a
// proper semantic version, which should always be the case.
var SemVer *version.Version

func init() {
	var err error
	SemVer, err = version.NewVersion(Version)
	if err != nil {
		panic(errors.New("Please use 'make build' to compile and install"))
	}
}

// ServerName is the name used to send to clients as a way
// to identify itself in the LSP
const ServerName = "hashicorp/terraform-ls"

// String returns the complete version string, including prerelease
func String() string {
	if Prerelease != "" {
		return fmt.Sprintf("%s-%s", Version, Prerelease)
	}
	return Version
}
