// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package filesystem

import (
	"io/fs"

	"github.com/hashicorp/terraform-ls/internal/document"
)

func documentAsFile(doc *document.Document) fs.File {
	return inMemFile{
		bytes: doc.Text,
		info:  documentAsFileInfo(doc),
	}
}

func documentAsFileInfo(doc *document.Document) fs.FileInfo {
	return inMemFileInfo{
		name:    doc.Filename,
		size:    len(doc.Text),
		modTime: doc.ModTime,
		mode:    0o755,
		isDir:   false,
	}
}

func documentsAsDirEntries(docs []*document.Document) []fs.DirEntry {
	entries := make([]fs.DirEntry, len(docs))

	for i, doc := range docs {
		entries[i] = documentAsDirEntry(doc)
	}

	return entries
}

func documentAsDirEntry(doc *document.Document) fs.DirEntry {
	return inMemDirEntry{
		name:  doc.Filename,
		isDir: false,
		typ:   0,
		info:  documentAsFileInfo(doc),
	}
}
