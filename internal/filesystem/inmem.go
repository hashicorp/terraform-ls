// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package filesystem

import (
	"bytes"
	"io/fs"
	"time"
)

type inMemFile struct {
	bytes []byte
	info  fs.FileInfo
}

func (f inMemFile) Read(b []byte) (int, error) {
	return bytes.NewBuffer(f.bytes).Read(b)
}

func (f inMemFile) Stat() (fs.FileInfo, error) {
	return f.info, nil
}

func (f inMemFile) Close() error {
	return nil
}

type inMemFileInfo struct {
	name    string
	size    int
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (fi inMemFileInfo) Name() string {
	return fi.name
}

func (fi inMemFileInfo) Size() int64 {
	return int64(fi.size)
}

func (fi inMemFileInfo) Mode() fs.FileMode {
	return fi.mode
}

func (fi inMemFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi inMemFileInfo) IsDir() bool {
	return fi.isDir
}

func (fi inMemFileInfo) Sys() interface{} {
	return nil
}

type inMemDirEntry struct {
	name  string
	isDir bool
	typ   fs.FileMode
	info  fs.FileInfo
}

func (de inMemDirEntry) Name() string {
	return de.name
}

func (de inMemDirEntry) IsDir() bool {
	return de.isDir
}

func (de inMemDirEntry) Type() fs.FileMode {
	return de.typ
}

func (de inMemDirEntry) Info() (fs.FileInfo, error) {
	return de.info, nil
}
