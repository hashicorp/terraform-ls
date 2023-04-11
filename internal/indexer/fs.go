// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package indexer

import "io/fs"

type ReadOnlyFS interface {
	fs.FS
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
	Stat(name string) (fs.FileInfo, error)
}
