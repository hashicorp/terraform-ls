package schema

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

type Reader interface {
	ProviderConfigSchema(name string) (*tfjson.Schema, error)
}

type Writer interface {
	ObtainSchemasForDir(*exec.Executor, string) error
	ProviderConfigSchema(name string) (*tfjson.Schema, error)
}

type storage struct {
	ps *tfjson.ProviderSchemas

	logger *log.Logger
}

func NewStorage() *storage {
	return &storage{
		logger: log.New(ioutil.Discard, "", 0),
	}
}

func (s *storage) SetLogger(logger *log.Logger) {
	s.logger = logger
}

func MockStorage(ps *tfjson.ProviderSchemas) *storage {
	s := NewStorage()
	s.ps = ps
	return s
}

func (c *storage) ObtainSchemasForDir(tf *exec.Executor, dir string) error {
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

	c.logger.Printf("Obtaining schemas for %q ...", dir)
	start := time.Now()
	ps, err := tf.ProviderSchemas()
	if err != nil {
		return fmt.Errorf("unable to get schemas: %s", err)
	}
	c.ps = ps
	c.logger.Printf("Schemas retrieved in %s", time.Since(start))

	return nil
}

func (c *storage) ProviderConfigSchema(name string) (*tfjson.Schema, error) {
	schema, ok := c.ps.Schemas[name]
	if !ok {
		return nil, &SchemaUnavailableErr{"provider", name}
	}

	if schema.ConfigSchema == nil {
		return nil, &SchemaUnavailableErr{"provider", name}
	}

	return schema.ConfigSchema, nil
}
