package filesystem

import (
	"io/ioutil"
	"log"
	"sync"
)

type fsystem struct {
	mu sync.RWMutex

	logger *log.Logger
	dirs   map[string]*dir
}

func NewFilesystem() *fsystem {
	return &fsystem{
		dirs:   make(map[string]*dir),
		logger: log.New(ioutil.Discard, "", 0),
	}
}

func (fs *fsystem) SetLogger(logger *log.Logger) {
	fs.logger = logger
}

func (fs *fsystem) Open(file File) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	d, ok := fs.dirs[file.Dir()]
	if !ok {
		d = newDir()
		fs.dirs[file.Dir()] = d
	}

	f, ok := d.files[file.Filename()]
	if !ok {
		f = NewFile(file.FullPath(), file.Text())
	}
	f.open = true
	f.version = file.Version()
	d.files[file.Filename()] = f
	return nil
}

func (fs *fsystem) Change(fh VersionedFileHandler, changes FileChanges) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	f := fs.file(fh)
	if f == nil || !f.open {
		return &FileNotOpenErr{fh}
	}
	for _, change := range changes {
		f.applyChange(change)
	}

	f.IncrementVersion()
	return nil
}

func (fs *fsystem) Close(fh FileHandler) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	f := fs.file(fh)
	if f == nil || !f.open {
		return &FileNotOpenErr{fh}
	}

	delete(fs.dirs[fh.Dir()].files, fh.Filename())

	return nil
}

func (fs *fsystem) GetFile(fh FileHandler) (File, error) {
	f := fs.file(fh)
	if f == nil || !f.open {
		return nil, &FileNotOpenErr{fh}
	}

	return f, nil
}

func (fs *fsystem) file(fh FileHandler) *file {
	d, ok := fs.dirs[fh.Dir()]
	if !ok {
		return nil
	}
	return d.files[fh.Filename()]
}
