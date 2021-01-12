package module

import (
	"fmt"
	"os"
)

func findFile(paths []string) (File, error) {
	var lf File
	var err error

	for _, path := range paths {
		lf, err = newFile(path)
		if err == nil {
			return lf, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return nil, err
}

type file struct {
	path string
}

func (f *file) Path() string {
	return f.path
}

func newFile(path string) (File, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("expected %s to be a file, not a dir", path)
	}

	return &file{path: path}, nil
}
