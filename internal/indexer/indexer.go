// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package indexer

import (
	"io/ioutil"
	"log"

	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/registry"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

type Indexer struct {
	logger           *log.Logger
	fs               ReadOnlyFS
	modStore         *state.ModuleStore
	varsStore        *state.VarsStore
	schemaStore      *state.ProviderSchemaStore
	registryModStore *state.RegistryModuleStore
	jobStore         job.JobStore
	tfExecFactory    exec.ExecutorFactory
	registryClient   registry.Client
}

func NewIndexer(fs ReadOnlyFS, modStore *state.ModuleStore, varsStore *state.VarsStore, schemaStore *state.ProviderSchemaStore,
	registryModStore *state.RegistryModuleStore, jobStore job.JobStore,
	tfExec exec.ExecutorFactory, registryClient registry.Client) *Indexer {

	discardLogger := log.New(ioutil.Discard, "", 0)

	return &Indexer{
		fs:               fs,
		modStore:         modStore,
		varsStore:        varsStore,
		schemaStore:      schemaStore,
		registryModStore: registryModStore,
		jobStore:         jobStore,
		tfExecFactory:    tfExec,
		registryClient:   registryClient,
		logger:           discardLogger,
	}
}

func (idx *Indexer) SetLogger(logger *log.Logger) {
	idx.logger = logger
}

type Collector interface {
	CollectJobId(jobId job.ID)
}
