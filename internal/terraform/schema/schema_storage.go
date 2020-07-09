package schema

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/addrs"
	tferr "github.com/hashicorp/terraform-ls/internal/terraform/errors"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"golang.org/x/sync/semaphore"
)

type Reader interface {
	ProviderConfigSchema(name addrs.Provider) (*tfjson.Schema, error)
	Providers() ([]addrs.Provider, error)
	ResourceSchema(rType string) (*tfjson.Schema, error)
	Resources() ([]Resource, error)
	DataSourceSchema(dsType string) (*tfjson.Schema, error)
	DataSources() ([]DataSource, error)
}

type Writer interface {
	ObtainSchemasForModule(context.Context, *exec.Executor, string) error
}

type Resource struct {
	Name            string
	Provider        addrs.Provider
	Description     string
	DescriptionKind tfjson.SchemaDescriptionKind
}

type DataSource struct {
	Name            string
	Provider        addrs.Provider
	Description     string
	DescriptionKind tfjson.SchemaDescriptionKind
}

type StorageFactory func(v string) (*Storage, error)

type Storage struct {
	ps        *tfjson.ProviderSchemas
	logger    *log.Logger
	tfVersion *version.Version

	// sem ensures atomic reading and obtaining of schemas
	// as the process of obtaining it may not be thread-safe
	sem *semaphore.Weighted
}

var defaultLogger = log.New(ioutil.Discard, "", 0)

func NewStorageForVersion(tfVersion string) (*Storage, error) {
	c, err := version.NewConstraint(
		">= 0.12.0", // Version 0.12 first introduced machine-readable schemas
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse constraint: %w", err)
	}

	ver, err := parseVersion(tfVersion)
	if err != nil {
		return nil, err
	}

	supported := c.Check(ver)
	if !supported {
		return nil, &tferr.UnsupportedTerraformVersion{
			Component:   "schema storage",
			Version:     tfVersion,
			Constraints: c,
		}
	}

	return &Storage{
		logger:    defaultLogger,
		tfVersion: ver,
		sem:       semaphore.NewWeighted(1),
	}, nil
}

func parseVersion(rawVersion string) (*version.Version, error) {
	ver, err := version.NewVersion(rawVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}

	// Assume that alpha/beta/rc prereleases have the same compatibility
	segments := ver.Segments64()
	segmentsOnly := fmt.Sprintf("%d.%d.%d", segments[0], segments[1], segments[2])
	return version.NewVersion(segmentsOnly)
}

func (s *Storage) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// ObtainSchemasForModule will obtain schema via tf
// and store it for later consumption via Reader methods
func (s *Storage) ObtainSchemasForModule(ctx context.Context, tf *exec.Executor, dir string) error {
	return s.obtainSchemasForModule(ctx, tf, dir)
}

func (s *Storage) obtainSchemasForModule(ctx context.Context, tf *exec.Executor, dir string) error {
	s.logger.Printf("Acquiring semaphore before retrieving schema for %q ...", dir)
	err := s.sem.Acquire(context.Background(), 1)
	if err != nil {
		return fmt.Errorf("failed to acquire semaphore: %w", err)
	}
	defer s.sem.Release(1)

	tf.SetWorkdir(dir)

	s.logger.Printf("Retrieving schemas for %q ...", dir)
	start := time.Now()
	ps, err := tf.ProviderSchemas(ctx)
	if err != nil {
		return fmt.Errorf("Unable to retrieve schemas for %q: %w", dir, err)
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

func (s *Storage) ProviderConfigSchema(addr addrs.Provider) (*tfjson.Schema, error) {
	identity := s.providerIdentity(addr)

	s.logger.Printf("Reading %q provider schema", identity)

	ps, err := s.schema()
	if err != nil {
		return nil, err
	}

	schema, ok := ps.Schemas[identity]
	if !ok {
		return nil, &SchemaUnavailableErr{"provider", identity}
	}

	if schema.ConfigSchema == nil {
		return nil, &SchemaUnavailableErr{"provider", identity}
	}

	return schema.ConfigSchema, nil
}

func (s *Storage) providerIdentity(addr addrs.Provider) string {
	if s.tfVersion.LessThan(version.Must(version.NewVersion("0.13"))) {
		return addr.Type
	}
	return addr.String()
}

func (s *Storage) Providers() ([]addrs.Provider, error) {
	ps, err := s.schema()
	if err != nil {
		return nil, err
	}

	providers := make([]addrs.Provider, 0)
	for sourceString := range ps.Schemas {
		addr, err := addrs.ParseProviderSourceString(sourceString)
		if err != nil {
			return nil, err
		}
		providers = append(providers, addr)
	}

	return providers, nil
}

func (s *Storage) ResourceSchema(rType string) (*tfjson.Schema, error) {
	s.logger.Printf("Reading %q resource schema", rType)

	ps, err := s.schema()
	if err != nil {
		return nil, err
	}

	// TODO: Reflect provider alias associations here
	// (need to be parsed and made accessible first)
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
		providerAddr, err := addrs.ParseProviderSourceString(provider)
		if err != nil {
			return nil, err
		}

		for name, r := range schema.ResourceSchemas {
			resources = append(resources, Resource{
				Provider:    providerAddr,
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

	// TODO: Reflect provider alias associations here
	// (need to be parsed and made accessible first)
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
		providerAddr, err := addrs.ParseProviderSourceString(provider)
		if err != nil {
			return nil, err
		}

		for name, d := range schema.DataSourceSchemas {
			dataSources = append(dataSources, DataSource{
				Provider:    providerAddr,
				Name:        name,
				Description: d.Block.Description,
			})
		}
	}

	return dataSources, nil
}
