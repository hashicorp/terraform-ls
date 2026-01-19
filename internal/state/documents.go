// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"log"
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/source"
)

type DocumentStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger

	// TimeProvider provides current time (for mocking time.Now in tests)
	TimeProvider func() time.Time
}

func (s *DocumentStore) OpenDocument(dh document.Handle, langId string, version int, text []byte) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	// TODO: Introduce Exists method to Txn?
	obj, err := txn.First(s.tableName, "id", dh.Dir, dh.Filename)
	if err != nil {
		return err
	}
	if obj != nil {
		return &AlreadyExistsError{
			Idx: dh.FullURI(),
		}
	}

	doc := &document.Document{
		Dir:        dh.Dir,
		Filename:   dh.Filename,
		ModTime:    s.TimeProvider(),
		LanguageID: langId,
		Version:    version,
		Text:       text,
		Lines:      source.MakeSourceLines(dh.Filename, text),
	}

	err = txn.Insert(s.tableName, doc)
	if err != nil {
		return err
	}

	err = updateJobsDirOpenMark(txn, dh.Dir, true)
	if err != nil {
		return err
	}
	err = updateWalkerDirOpenMark(txn, dh.Dir, true)
	if err != nil {
		return err
	}
	err = updateModuleChangeDirOpenMark(txn, dh.Dir, true)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *DocumentStore) UpdateDocument(dh document.Handle, newText []byte, newVersion int) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	doc, err := copyDocument(txn, dh)
	if err != nil {
		return err
	}

	doc.Text = newText
	doc.Lines = source.MakeSourceLines(dh.Filename, newText)
	doc.Version = newVersion
	doc.ModTime = s.TimeProvider()

	err = txn.Insert(s.tableName, doc)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func copyDocument(txn *memdb.Txn, dh document.Handle) (*document.Document, error) {
	doc, err := getDocument(txn, dh)
	if err != nil {
		return nil, err
	}

	return doc.Copy(), nil
}

func (s *DocumentStore) GetDocument(dh document.Handle) (*document.Document, error) {
	txn := s.db.Txn(false)
	return getDocument(txn, dh)
}

func getDocument(txn *memdb.Txn, dh document.Handle) (*document.Document, error) {
	obj, err := txn.First(documentsTableName, "id", dh.Dir, dh.Filename)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, &document.DocumentNotFound{
			URI: dh.FullURI(),
		}
	}
	return obj.(*document.Document), nil
}

func (s *DocumentStore) CloseDocument(dh document.Handle) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	obj, err := txn.First(s.tableName, "id", dh.Dir, dh.Filename)
	if err != nil {
		return err
	}

	if obj == nil {
		// already removed
		return &document.DocumentNotFound{
			URI: dh.FullURI(),
		}
	}

	_, err = txn.DeleteAll(s.tableName, "id", dh.Dir, dh.Filename)
	if err != nil {
		return err
	}

	err = updateJobsDirOpenMark(txn, dh.Dir, false)
	if err != nil {
		return err
	}

	err = updateWalkerDirOpenMark(txn, dh.Dir, false)
	if err != nil {
		return err
	}

	err = updateModuleChangeDirOpenMark(txn, dh.Dir, false)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *DocumentStore) ListDocumentsInDir(dirHandle document.DirHandle) ([]*document.Document, error) {
	txn := s.db.Txn(false)
	it, err := txn.Get(s.tableName, "dir", dirHandle)
	if err != nil {
		return nil, err
	}

	docs := make([]*document.Document, 0)
	for item := it.Next(); item != nil; item = it.Next() {
		doc := item.(*document.Document)
		docs = append(docs, doc)
	}

	return docs, nil
}

func (s *DocumentStore) IsDocumentOpen(dh document.Handle) (bool, error) {
	txn := s.db.Txn(false)

	obj, err := txn.First(s.tableName, "id", dh.Dir, dh.Filename)
	if err != nil {
		return false, err
	}

	return obj != nil, nil
}

func (s *DocumentStore) HasOpenDocuments(dirHandle document.DirHandle) (bool, error) {
	txn := s.db.Txn(false)
	return DirHasOpenDocuments(txn, dirHandle)
}

func DirHasOpenDocuments(txn *memdb.Txn, dirHandle document.DirHandle) (bool, error) {
	obj, err := txn.First(documentsTableName, "dir", dirHandle)
	if err != nil {
		return false, err
	}

	return obj != nil, nil
}
