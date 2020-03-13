package schema

import (
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

type Reader interface {
	ProviderConfigSchema(name string) (*tfjson.Schema, error)
}

type Writer interface {
	ObtainSchemasForWorkspace(*exec.Executor, string) error
	AddWorkspaceForWatching(string) error
	StartWatching(*exec.Executor) error
}

type Storage struct {
	ps       *tfjson.ProviderSchemas
	w        watcher
	watching bool

	logger *log.Logger

	// mu ensures atomic reading and obtaining of schemas
	// as the process of obtaining it may not be thread-safe
	mu sync.RWMutex

	// sync makes operations synchronous which makes testing easier
	sync bool
}

var defaultLogger = log.New(ioutil.Discard, "", 0)

func NewStorage() *Storage {
	return &Storage{
		logger: defaultLogger,
	}
}

func (s *Storage) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// ObtainSchemasForWorkspace will (by default) asynchronously obtain schema via tf
// and store it for later consumption via Reader methods
func (s *Storage) ObtainSchemasForWorkspace(tf *exec.Executor, dir string) error {
	if s.sync {
		return s.obtainSchemasForWorkspace(tf, dir)
	}

	// This routine is not cancellable in itself
	// but the time-consuming part is done by exec.Executor
	// which is cancellable via its own context
	go func() {
		err := s.obtainSchemasForWorkspace(tf, dir)
		if err != nil {
			s.logger.Println("error obtaining schemas:", err)
		}
	}()

	return nil
}

func (s *Storage) obtainSchemasForWorkspace(tf *exec.Executor, dir string) error {
	s.logger.Printf("Obtaining lock before retrieving schema for %q ...", dir)
	s.mu.Lock()
	defer s.mu.Unlock()

	// Checking the version here may be excessive
	// TODO: Find a way to centralize this
	tfVersions, err := version.NewConstraint(">= 0.12.0")
	if err != nil {
		return err
	}
	err = tf.VersionIsSupported(tfVersions)
	if err != nil {
		return err
	}

	tf.SetWorkdir(dir)

	s.logger.Printf("Retrieving schemas for %q ...", dir)
	start := time.Now()
	ps, err := tf.ProviderSchemas()
	if err != nil {
		return fmt.Errorf("Unable to retrieve schemas: %s", err)
	}
	s.ps = ps
	s.logger.Printf("Schemas retrieved in %s", time.Since(start))
	return nil
}

func (s *Storage) ProviderConfigSchema(name string) (*tfjson.Schema, error) {
	s.logger.Printf("Obtaining lock before reading %q provider schema", name)
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.logger.Printf("Reading %q provider schema", name)
	schema, ok := s.ps.Schemas[name]
	if !ok {
		return nil, &SchemaUnavailableErr{"provider", name}
	}

	if schema.ConfigSchema == nil {
		return nil, &SchemaUnavailableErr{"provider", name}
	}

	return schema.ConfigSchema, nil
}

// watcher creates a new Watcher instance
// if one doesn't exist yet or returns an existing one
func (s *Storage) watcher() (watcher, error) {
	if s.w != nil {
		return s.w, nil
	}

	w, err := NewWatcher()
	if err != nil {
		return nil, err
	}
	w.SetLogger(s.logger)

	s.w = w
	return s.w, nil
}

// StartWatching starts to watch for plugin changes in dirs that were added
// via AddWorkspaceForWatching until StopWatching() is called
func (s *Storage) StartWatching(tf *exec.Executor) error {
	if s.watching {
		return fmt.Errorf("watching already in progress")
	}
	w, err := s.watcher()
	if err != nil {
		return err
	}

	go w.OnPluginChange(func(ww *watchedWorkspace) error {
		s.obtainSchemasForWorkspace(tf, ww.dir)
		return nil
	})
	s.watching = true

	s.logger.Printf("Watching for plugin changes ...")

	return nil
}

func (s *Storage) StopWatching() error {
	if s.w == nil {
		return nil
	}
	s.logger.Println("Stopping watcher ...")
	err := s.w.Close()
	if err == nil {
		s.watching = false
	}

	return err
}

func (s *Storage) AddWorkspaceForWatching(dir string) error {
	w, err := s.watcher()
	if err != nil {
		return err
	}

	s.logger.Printf("Adding workspace for watching: %q", dir)

	return w.AddWorkspace(dir)
}
