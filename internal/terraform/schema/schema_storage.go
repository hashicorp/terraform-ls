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
	Providers() ([]string, error)
	ResourceSchema(rType string) (*tfjson.Schema, error)
	Resources() ([]Resource, error)
	DataSourceSchema(dsType string) (*tfjson.Schema, error)
	DataSources() ([]DataSource, error)
}

type Writer interface {
	ObtainSchemasForModule(*exec.Executor, string) error
}

type Resource struct {
	Name            string
	Provider        string
	Description     string
	DescriptionKind tfjson.SchemaDescriptionKind
}

type DataSource struct {
	Name            string
	Provider        string
	Description     string
	DescriptionKind tfjson.SchemaDescriptionKind
}

type StorageFactory func() *Storage

type Storage struct {
	ps *tfjson.ProviderSchemas

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

	rawVer, err := version.NewVersion(v)
	if err != nil {
		return fmt.Errorf("failed to parse version: %w", err)
	}

	// Assume that alpha/beta/rc prereleases have the same compatibility
	segments := rawVer.Segments64()
	segmentsOnly := fmt.Sprintf("%d.%d.%d", segments[0], segments[1], segments[2])
	ver, err := version.NewVersion(segmentsOnly)
	if err != nil {
		return fmt.Errorf("failed to parse stripped version: %w", err)
	}

	supported := c.Check(ver)
	if !supported {
		return &errors.UnsupportedTerraformVersion{
			Component:   "schema storage",
			Version:     v,
			Constraints: c,
		}
	}

	return nil
}

func (s *Storage) SetLogger(logger *log.Logger) {
	s.logger = logger
}

func (s *Storage) SetSynchronous() {
	s.sync = true
}

// ObtainSchemasForModule will (by default) asynchronously obtain schema via tf
// and store it for later consumption via Reader methods
func (s *Storage) ObtainSchemasForModule(tf *exec.Executor, dir string) error {
	if s.sync {
		return s.obtainSchemasForModule(tf, dir)
	}

	// This routine is not cancellable in itself
	// but the time-consuming part is done by exec.Executor
	// which is cancellable via its own context
	go func() {
		err := s.obtainSchemasForModule(tf, dir)
		if err != nil {
			s.logger.Printf("error obtaining schemas for %s: %s", dir, err)
		}
	}()

	return nil
}

func (s *Storage) obtainSchemasForModule(tf *exec.Executor, dir string) error {
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
		return fmt.Errorf("Unable to retrieve schemas for %q: %s", dir, err)
	}
	s.ps = ps
	s.logger.Printf("Schemas retrieved for %q in %s", dir, time.Since(start))
	return nil
}

func (s *Storage) schema() (*tfjson.ProviderSchemas, error) {
	s.logger.Println("Acquiring semaphore before reading schema")
	acquired := s.sem.TryAcquire(1)
	if !acquired {
		return nil, fmt.Errorf("schema temporarily unavailable")
	}
	defer s.sem.Release(1)

	if s.ps == nil {
		return nil, &NoSchemaAvailableErr{}
	}
	return s.ps, nil
}

func (s *Storage) ProviderConfigSchema(name string) (*tfjson.Schema, error) {
	s.logger.Printf("Reading %q provider schema", name)

	ps, err := s.schema()
	if err != nil {
		return nil, err
	}

	schema, ok := ps.Schemas[name]
	if !ok {
		return nil, &SchemaUnavailableErr{"provider", name}
	}

	if schema.ConfigSchema == nil {
		return nil, &SchemaUnavailableErr{"provider", name}
	}

	return schema.ConfigSchema, nil
}

func (s *Storage) Providers() ([]string, error) {
	ps, err := s.schema()
	if err != nil {
		return nil, err
	}

	providers := make([]string, 0)
	for name := range ps.Schemas {
		providers = append(providers, name)
	}

	return providers, nil
}

func (s *Storage) ResourceSchema(rType string) (*tfjson.Schema, error) {
	// TODO: this is going to need to use provider identities, especially in 0.13
	s.logger.Printf("Reading %q resource schema", rType)

	ps, err := s.schema()
	if err != nil {
		return nil, err
	}

	// Vast majority of resources should follow naming convention
	// of <provider>_resource_name, but this is not enforced
	// in any way so we have to check all providers
	for _, schema := range ps.Schemas {
		rSchema, ok := schema.ResourceSchemas[rType]
		if ok {
			return rSchema, nil
		}
	}

	return nil, &SchemaUnavailableErr{"resource", rType}
}

func (s *Storage) Resources() ([]Resource, error) {
	ps, err := s.schema()
	if err != nil {
		return nil, err
	}

	resources := make([]Resource, 0)
	for provider, schema := range ps.Schemas {
		for name, r := range schema.ResourceSchemas {
			resources = append(resources, Resource{
				Provider:    provider,
				Name:        name,
				Description: r.Block.Description,
			})
		}
	}

	return resources, nil
}

func (s *Storage) DataSourceSchema(dsType string) (*tfjson.Schema, error) {
	s.logger.Printf("Reading %q datasource schema", dsType)

	ps, err := s.schema()
	if err != nil {
		return nil, err
	}

	// Vast majority of Datasources should follow naming convention
	// of <provider>_datasource_name, but this is not enforced
	// in any way so we have to check all providers
	for _, schema := range ps.Schemas {
		rSchema, ok := schema.DataSourceSchemas[dsType]
		if ok {
			return rSchema, nil
		}
	}

	return nil, &SchemaUnavailableErr{"data", dsType}
}

func (s *Storage) DataSources() ([]DataSource, error) {
	ps, err := s.schema()
	if err != nil {
		return nil, err
	}

	dataSources := make([]DataSource, 0)
	for provider, schema := range ps.Schemas {
		for name, d := range schema.DataSourceSchemas {
			dataSources = append(dataSources, DataSource{
				Provider:    provider,
				Name:        name,
				Description: d.Block.Description,
			})
		}
	}

	return dataSources, nil
}
