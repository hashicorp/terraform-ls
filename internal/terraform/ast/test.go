package ast

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
)

type TestFilename string

func (tf TestFilename) String() string {
	return string(tf)
}

func (tf TestFilename) IsJSON() bool {
	return strings.HasSuffix(string(tf), ".json")
}

func (tf TestFilename) IsIgnored() bool {
	return IsIgnoredFile(string(tf))
}

func IsTestFilename(name string) bool {
	return strings.HasSuffix(name, ".tftest.hcl") ||
		strings.HasSuffix(name, ".tftest.json")
}

type TestFiles map[TestFilename]*hcl.File

type TestDiags map[TestFilename]hcl.Diagnostics

type SourceTestDiags map[DiagnosticSource]TestDiags
