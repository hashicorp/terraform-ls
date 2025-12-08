// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/modules/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/registry"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfregistry "github.com/hashicorp/terraform-schema/registry"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

const tracerName = "github.com/hashicorp/terraform-ls/internal/terraform/module"

// PreloadEmbeddedSchema loads provider schemas based on
// provider requirements parsed earlier via [LoadModuleMetadata].
// This is the cheapest way of getting provider schemas in terms
// of resources, time and complexity/UX.
func PreloadEmbeddedSchema(ctx context.Context, logger *log.Logger, fs fs.ReadDirFS, modStore *state.ModuleStore, schemaStore *globalState.ProviderSchemaStore, modPath string) error {
	mod, err := modStore.ModuleRecordByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid preloading schema if it is already in progress or already known
	if mod.PreloadEmbeddedSchemaState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetPreloadEmbeddedSchemaState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}
	defer modStore.SetPreloadEmbeddedSchemaState(modPath, op.OpStateLoaded)

	pReqs, err := modStore.ProviderRequirementsForModule(modPath)
	if err != nil {
		return err
	}

	missingReqs, err := schemaStore.MissingSchemas(pReqs)
	if err != nil {
		return err
	}
	if len(missingReqs) == 0 {
		// avoid preloading any schemas if we already have all
		return nil
	}

	for _, pAddr := range missingReqs {
		err := globalState.PreloadSchemaForProviderAddr(ctx, pAddr, fs, schemaStore, logger)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetModuleDataFromRegistry obtains data about any modules (inputs & outputs)
// from the Registry API based on module calls which were previously parsed
// via [LoadModuleMetadata]. The same data could be obtained via [ParseModuleManifest]
// but getting it from the API comes with little expectations,
// specifically the modules do not need to be installed on disk and we don't
// need to parse and decode all files.
func GetModuleDataFromRegistry(ctx context.Context, regClient registry.Client, modStore *state.ModuleStore, modRegStore *globalState.RegistryModuleStore, modPath string) error {
	// loop over module calls
	calls, err := modStore.DeclaredModuleCalls(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs (parsing, meta) reported no changes

	var errs *multierror.Error

	for _, declaredModule := range calls {
		sourceAddr, ok := declaredModule.SourceAddr.(tfaddr.Module)
		if !ok {
			// skip any modules which do not come from the Registry
			continue
		}

		// check if that address was already cached
		// if there was an error finding in cache, so cache again
		exists, err := modRegStore.Exists(sourceAddr, declaredModule.Version)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		if exists {
			// entry in cache, no need to look up
			continue
		}

		// get module data from Terraform Registry
		metaData, err := regClient.GetModuleData(ctx, sourceAddr, declaredModule.Version)
		if err != nil {
			errs = multierror.Append(errs, err)

			clientError := registry.ClientError{}
			if errors.As(err, &clientError) &&
				((clientError.StatusCode >= 400 && clientError.StatusCode < 408) ||
					(clientError.StatusCode > 408 && clientError.StatusCode < 429)) {
				// Still cache the module
				err = modRegStore.CacheError(sourceAddr)
				if err != nil {
					errs = multierror.Append(errs, err)
				}
			}

			continue
		}

		registryInputs := metaData.Root.Inputs
		registryOutputs := metaData.Root.Outputs

		// Check if the source address contains a submodule
		// If we can find the submodule in the API response, we will use its inputs and outputs instead
		if sourceAddr.Subdir != "" {
			for _, mod := range metaData.Submodules {
				if mod.Path == sourceAddr.Subdir {
					registryInputs = mod.Inputs
					registryOutputs = mod.Outputs

					break
				}
			}
		}

		inputs := make([]tfregistry.Input, len(registryInputs))
		for i, input := range registryInputs {
			isRequired := isRegistryModuleInputRequired(metaData.PublishedAt, input)
			inputs[i] = tfregistry.Input{
				Name:        input.Name,
				Description: lang.Markdown(input.Description),
				Required:    isRequired,
			}

			inputType := cty.DynamicPseudoType
			if input.Type != "" {
				// Registry API unfortunately doesn't marshal types using
				// cty marshalers, making it lossy, so we just try to decode
				// on best-effort basis.
				rawType := []byte(fmt.Sprintf("%q", input.Type))
				typ, err := ctyjson.UnmarshalType(rawType)
				if err == nil {
					inputType = typ
				}
			}
			inputs[i].Type = inputType

			if input.Default != "" {
				// Registry API unfortunately doesn't marshal values using
				// cty marshalers, making it lossy, so we just try to decode
				// on best-effort basis.
				val, err := ctyjson.Unmarshal([]byte(input.Default), inputType)
				if err == nil {
					inputs[i].Default = val
				}
			}
		}
		outputs := make([]tfregistry.Output, len(registryOutputs))
		for i, output := range registryOutputs {
			outputs[i] = tfregistry.Output{
				Name:        output.Name,
				Description: lang.Markdown(output.Description),
			}
		}

		modVersion, err := version.NewVersion(metaData.Version)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		// if not, cache it
		err = modRegStore.Cache(sourceAddr, modVersion, inputs, outputs)
		if err != nil {
			// A different job which ran in parallel for a different module block
			// with the same source may have already cached the same module.
			existsError := &globalState.AlreadyExistsError{}
			if errors.As(err, &existsError) {
				continue
			}

			errs = multierror.Append(errs, err)
			continue
		}
	}

	return errs.ErrorOrNil()
}

// isRegistryModuleInputRequired checks whether the module input is required.
// It reflects the fact that modules ingested into the Registry
// may have used `default = null` (implying optional variable) which
// the Registry wasn't able to recognise until ~ 19th August 2022.
func isRegistryModuleInputRequired(publishTime time.Time, input registry.Input) bool {
	fixTime := time.Date(2022, time.August, 20, 0, 0, 0, 0, time.UTC)
	// Modules published after the date have "nullable" inputs
	// (default = null) ingested as Required=false and Default="null".
	//
	// The same inputs ingested prior to the date make it impossible
	// to distinguish variable with `default = null` and missing default.
	if input.Required && input.Default == "" && publishTime.Before(fixTime) {
		// To avoid false diagnostics, we safely assume the input is optional
		return false
	}
	return input.Required
}
