package filesystem

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/hashicorp/terraform-ls/internal/source"
	"github.com/spf13/afero"
)

type fsystem struct {
	memFs afero.Fs
	osFs  afero.Fs

	docMeta   map[string]*documentMetadata
	docMetaMu *sync.RWMutex

	logger *log.Logger
}

func NewFilesystem() *fsystem {
	return &fsystem{
		memFs:     afero.NewMemMapFs(),
		osFs:      afero.NewReadOnlyFs(afero.NewOsFs()),
		docMeta:   make(map[string]*documentMetadata, 0),
		docMetaMu: &sync.RWMutex{},
		logger:    log.New(ioutil.Discard, "", 0),
	}
}

func (fs *fsystem) SetLogger(logger *log.Logger) {
	fs.logger = logger
}

func (fs *fsystem) CreateDocument(dh DocumentHandler, langId string, text []byte) error {
	_, err := fs.memFs.Stat(dh.Dir())
	if err != nil {
		if os.IsNotExist(err) {
			err := fs.memFs.MkdirAll(dh.Dir(), 0755)
			if err != nil {
				return fmt.Errorf("failed to create parent dir: %w", err)
			}
		} else {
			return err
		}
	}

	f, err := fs.memFs.Create(dh.FullPath())
	if err != nil {
		return err
	}
	_, err = f.Write(text)
	if err != nil {
		return err
	}

	return fs.createDocumentMetadata(dh, langId, text)
}

func (fs *fsystem) CreateAndOpenDocument(dh DocumentHandler, langId string, text []byte) error {
	err := fs.CreateDocument(dh, langId, text)
	if err != nil {
		return err
	}

	return fs.markDocumentAsOpen(dh)
}

func (fs *fsystem) ChangeDocument(dh VersionedDocumentHandler, changes DocumentChanges) error {
	if len(changes) == 0 {
		return nil
	}

	isOpen, err := fs.isDocumentOpen(dh)
	if err != nil {
		return err
	}

	if !isOpen {
		return &DocumentNotOpenErr{dh}
	}

	f, err := fs.memFs.OpenFile(dh.FullPath(), os.O_RDWR, 0700)
	if err != nil {
		return err
	}
	defer f.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(f)
	if err != nil {
		return err
	}

	for _, ch := range changes {
		err := fs.applyDocumentChange(&buf, ch)
		if err != nil {
			return err
		}
	}

	err = f.Truncate(0)
	if err != nil {
		return err
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}
	_, err = f.Write(buf.Bytes())
	if err != nil {
		return err
	}

	return fs.updateDocumentMetadataLines(dh, buf.Bytes())
}

func (fs *fsystem) applyDocumentChange(buf *bytes.Buffer, change DocumentChange) error {
	// if the range is nil, we assume it is full content change
	if change.Range() == nil {
		buf.Reset()
		_, err := buf.WriteString(change.Text())
		return err
	}

	lines := source.MakeSourceLines("", buf.Bytes())

	startByte, err := ByteOffsetForPos(lines, change.Range().Start)
	if err != nil {
		return err
	}
	endByte, err := ByteOffsetForPos(lines, change.Range().End)
	if err != nil {
		return err
	}

	diff := endByte - startByte
	if diff > 0 {
		buf.Grow(diff)
	}

	beforeChange := make([]byte, startByte, startByte)
	copy(beforeChange, buf.Bytes())
	afterBytes := buf.Bytes()[endByte:]
	afterChange := make([]byte, len(afterBytes), len(afterBytes))
	copy(afterChange, afterBytes)

	buf.Reset()

	_, err = buf.Write(beforeChange)
	if err != nil {
		return err
	}
	_, err = buf.WriteString(change.Text())
	if err != nil {
		return err
	}
	_, err = buf.Write(afterChange)
	if err != nil {
		return err
	}

	return nil
}

func (fs *fsystem) CloseAndRemoveDocument(dh DocumentHandler) error {
	isOpen, err := fs.isDocumentOpen(dh)
	if err != nil {
		return err
	}

	if !isOpen {
		return &DocumentNotOpenErr{dh}
	}

	err = fs.memFs.Remove(dh.FullPath())
	if err != nil {
		return err
	}

	return fs.removeDocumentMetadata(dh)
}

func (fs *fsystem) GetDocument(dh DocumentHandler) (Document, error) {
	dm, err := fs.getDocumentMetadata(dh)
	if err != nil {
		return nil, err
	}

	return &document{
		meta: dm,
		fs:   fs.memFs,
	}, nil
}

func (fs *fsystem) ReadFile(name string) ([]byte, error) {
	b, err := afero.ReadFile(fs.memFs, name)
	if err != nil && os.IsNotExist(err) {
		return afero.ReadFile(fs.osFs, name)
	}

	return b, err
}

func (fsys *fsystem) ReadDir(name string) ([]fs.DirEntry, error) {
	memList, err := afero.NewIOFS(fsys.memFs).ReadDir(name)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("memory FS: %w", err)
	}
	osList, err := afero.NewIOFS(fsys.osFs).ReadDir(name)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("OS FS: %w", err)
	}

	list := memList
	for _, osEntry := range osList {
		if fileIsInList(list, osEntry) {
			continue
		}
		list = append(list, osEntry)
	}

	return list, nil
}

func fileIsInList(list []fs.DirEntry, entry fs.DirEntry) bool {
	for _, di := range list {
		if di.Name() == entry.Name() {
			return true
		}
	}
	return false
}

func (fs *fsystem) Open(name string) (fs.File, error) {
	f, err := fs.memFs.Open(name)
	if err != nil && os.IsNotExist(err) {
		return fs.osFs.Open(name)
	}

	return f, err
}

func (fs *fsystem) Stat(name string) (os.FileInfo, error) {
	fi, err := fs.memFs.Stat(name)
	if err != nil && os.IsNotExist(err) {
		return fs.osFs.Stat(name)
	}

	return fi, err
}
