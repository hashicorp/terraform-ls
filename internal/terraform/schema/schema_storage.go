package schema

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/errors"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"golang.org/x/sync/semaphore"
)

type Reader interface {
	ProviderConfigSchema(name string) (*tfjson.Schema, error)
	ResourceSchema(rType string) (*tfjson.Schema, error)
	DataSourceSchema(dsType string) (*tfjson.Schema, error)
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

	// sem ensures atomic reading and obtaining of schemas
	// as the process of obtaining it may not be thread-safe
	sem *semaphore.Weighted

	// sync makes operations synchronous which makes testing easier
	sync bool
}

var defaultLogger = log.New(ioutil.Discard, "", 0)

func NewStorage() *Storage {
	return &Storage{
		logger: defaultLogger,
		sem:    semaphore.NewWeighted(1),
	}
}

func SchemaSupportsTerraform(v string) error {
	c, err := version.NewConstraint(
		">= 0.12.0", // Version 0.12 first introduced machine-readable schemas
	)
	if err != nil {
		return fmt.Errorf("failed to parse constraint: %w", err)
	}

	ver, err := version.NewVersion(v)
	if err != nil {
		return fmt.Errorf("failed to parse verison: %w", err)
	}

	supported := c.Check(ver)
	if !supported {
		return &errors.UnsupportedTerraformVersion{
			Component:   "schema storage",
			Version:     v,
			Constraints: c,
		}
	}

	return watcherSupportsTerraform(ver)
}

func (s *Storage) SetLogger(logger *log.Logger) {
	s.logger = logger
}

func (s *Storage) SetSynchronous() {
	s.sync = true
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
	s.logger.Printf("Acquiring semaphore before retrieving schema for %q ...", dir)
	err := s.sem.Acquire(context.Background(), 1)
	if err != nil {
		return fmt.Errorf("failed to acquire semaphore: %w", err)
	}
	defer s.sem.Release(1)

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
	s.logger.Printf("Acquiring semaphore before reading %q provider schema", name)
	acquired := s.sem.TryAcquire(1)
	if !acquired {
		return nil, fmt.Errorf("schema unavailable temporarily")
	}
	defer s.sem.Release(1)

	s.logger.Printf("Reading %q provider schema", name)

	if s.ps == nil {
		return nil, &SchemaUnavailableErr{"provider", name}
	}

	schema, ok := s.ps.Schemas[name]
	if !ok {
		return nil, &SchemaUnavailableErr{"provider", name}
	}

	if schema.ConfigSchema == nil {
		return nil, &SchemaUnavailableErr{"provider", name}
	}

	return schema.ConfigSchema, nil
}

func (s *Storage) ResourceSchema(rType string) (*tfjson.Schema, error) {
	s.logger.Printf("Acquiring semaphore before reading %q resource schema", rType)
	acquired := s.sem.TryAcquire(1)
	if !acquired {
		return nil, fmt.Errorf("schema unavailable temporarily")
	}
	defer s.sem.Release(1)

	s.logger.Printf("Reading %q resource schema", rType)

	if s.ps == nil {
		return nil, &SchemaUnavailableErr{"resource", rType}
	}

	// Vast majority of resources should follow naming convention
	// of <provider>_resource_name, but this is not enforced
	// in any way so we have to check all providers
	for _, schema := range s.ps.Schemas {
		rSchema, ok := schema.ResourceSchemas[rType]
		if ok {
			return rSchema, nil
		}
	}

	return nil, &SchemaUnavailableErr{"resource", rType}
}

func (s *Storage) DataSourceSchema(dsType string) (*tfjson.Schema, error) {
	s.logger.Printf("Acquiring semaphore before reading %q datasource schema", dsType)
	acquired := s.sem.TryAcquire(1)
	if !acquired {
		return nil, fmt.Errorf("schema unavailable temporarily")
	}
	defer s.sem.Release(1)

	s.logger.Printf("Reading %q datasource schema", dsType)

	if s.ps == nil {
		return nil, &SchemaUnavailableErr{"data", dsType}
	}

	// Vast majority of Datasources should follow naming convention
	// of <provider>_datasource_name, but this is not enforced
	// in any way so we have to check all providers
	for _, schema := range s.ps.Schemas {
		rSchema, ok := schema.DataSourceSchemas[dsType]
		if ok {
			return rSchema, nil
		}
	}

	return nil, &SchemaUnavailableErr{"data", dsType}
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
		s.logger.Println("watching already in progress")
		return nil
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
