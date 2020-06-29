package rootmodule

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var (
	discardLogger = log.New(ioutil.Discard, "", 0)

	// skipDirNames represent directory names which would never contain
	// plugin/module cache, so it's safe to skip them during the walk
	skipDirNames = map[string]bool{
		".git":                true,
		".idea":               true,
		".vscode":             true,
		"terraform.tfstate.d": true,
	}
)

type Walker struct {
	logger *log.Logger
}

func NewWalker() *Walker {
	return &Walker{
		logger: discardLogger,
	}
}

func (w *Walker) SetLogger(logger *log.Logger) {
	w.logger = logger
}

type WalkFunc func(rootModulePath string) error

func (w *Walker) WalkInitializedRootModules(path string, wf WalkFunc) error {
	w.logger.Printf("walking through %s", path)
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			w.logger.Printf("unable to access %s: %s", path, err.Error())
			return nil
		}

		if info.Name() == ".terraform" {
			rootDir, err := filepath.Abs(filepath.Dir(path))
			if err != nil {
				return err
			}

			w.logger.Printf("found root module %s", rootDir)
			return wf(rootDir)
		}

		if !info.IsDir() {
			// All files are skipped, we only care about dirs
			return nil
		}

		if isSkippableDir(info.Name()) {
			w.logger.Printf("skipping %s", path)
			return filepath.SkipDir
		}

		return nil
	})
}

func isSkippableDir(dirName string) bool {
	_, ok := skipDirNames[dirName]
	return ok
}
