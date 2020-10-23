package filesystem

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"sync"

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

func (fs *fsystem) CreateDocument(dh DocumentHandler, text []byte) error {
	f, err := fs.memFs.Create(dh.FullPath())
	if err != nil {
		return err
	}
	_, err = f.Write(text)
	if err != nil {
		return err
	}

	return fs.createDocumentMetadata(dh, text)
}

func (fs *fsystem) CreateAndOpenDocument(dh DocumentHandler, text []byte) error {
	err := fs.CreateDocument(dh, text)
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
	if rangeIsNil(change.Range()) {
		buf.Reset()
		_, err := buf.WriteString(change.Text())
		return err
	}

	// apply partial change
	diff := diffLen(change)
	if diff > 0 {
		buf.Grow(diff)
	}

	startByte, endByte := change.Range().Start.Byte, change.Range().End.Byte
	beforeChange := make([]byte, startByte, startByte)
	copy(beforeChange, buf.Bytes())
	afterBytes := buf.Bytes()[endByte:]
	afterChange := make([]byte, len(afterBytes), len(afterBytes))
	copy(afterChange, afterBytes)

	buf.Reset()

	_, err := buf.Write(beforeChange)
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

func diffLen(change DocumentChange) int {
	rangeLen := change.Range().End.Byte - change.Range().Start.Byte
	return len(change.Text()) - rangeLen
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

func (fs *fsystem) ReadDir(name string) ([]os.FileInfo, error) {
	memList, err := afero.ReadDir(fs.memFs, name)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	osList, err := afero.ReadDir(fs.osFs, name)
	if err != nil {
		return nil, err
	}

	list := memList
	for _, osFi := range osList {
		if fileIsInList(list, osFi) {
			continue
		}
		list = append(list, osFi)
	}

	return list, nil
}

func fileIsInList(list []os.FileInfo, file os.FileInfo) bool {
	for _, fi := range list {
		if fi.Name() == file.Name() {
			return true
		}
	}
	return false
}

func (fs *fsystem) Open(name string) (File, error) {
	f, err := fs.memFs.Open(name)
	if err != nil && os.IsNotExist(err) {
		return fs.osFs.Open(name)
	}

	return f, err
}
