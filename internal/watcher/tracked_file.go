package watcher

import (
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
)

func trackedFileFromPath(path string) (TrackedFile, error) {
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}

	b, err := fileSha256Sum(path)
	if err != nil {
		return nil, err
	}

	return &trackedFile{
		path:      path,
		sha256sum: string(b),
	}, nil
}

type trackedFile struct {
	path      string
	sha256sum string
}

func (tf *trackedFile) Path() string {
	return tf.path
}

func (tf *trackedFile) Sha256Sum() string {
	return tf.sha256sum
}

func fileSha256Sum(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}
