// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package filesystem

import (
	"io/fs"
	"os"
)

type osFs struct{}

func (osfs osFs) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (osfs osFs) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (osfs osFs) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func (osfs osFs) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}
