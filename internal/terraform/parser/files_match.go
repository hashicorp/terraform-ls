// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package parser

import (
	"bytes"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
)

// FilesMatch reports whether the files in dir that include accepts, read
// through fs, are exactly the files in parsed, with the content they were
// parsed from. It returns false if a file was added, removed or changed since
// it was parsed, or if any of them cannot be read.
func FilesMatch[K interface {
	comparable
	String() string
}](fs FS, dir string, include func(name string) bool, parsed map[K]*hcl.File) bool {
	parsedSrc := make(map[string][]byte, len(parsed))
	for name, file := range parsed {
		if file != nil {
			parsedSrc[name.String()] = file.Bytes
		}
	}

	entries, err := fs.ReadDir(dir)
	if err != nil {
		return false
	}

	matched := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !include(name) {
			continue
		}

		src, ok := parsedSrc[name]
		if !ok {
			return false
		}
		current, err := fs.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return false
		}
		if !bytes.Equal(current, src) {
			return false
		}
		matched++
	}

	return matched == len(parsedSrc)
}
