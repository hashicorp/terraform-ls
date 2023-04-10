// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	"github.com/hashicorp/terraform-ls/internal/langserver/notifier"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/telemetry"
	"github.com/hashicorp/terraform-schema/backend"
)

func sendModuleTelemetry(store *state.StateStore, telemetrySender telemetry.Sender) notifier.Hook {
	return func(ctx context.Context, changes state.ModuleChanges) error {
		if changes.IsRemoval {
			// we ignore removed modules for now
			return nil
		}

		mod, err := notifier.ModuleFromContext(ctx)
		if err != nil {
			return err
		}

		properties, hasChanged := moduleTelemetryData(mod, changes, store)
		if hasChanged {
			telemetrySender.SendEvent(ctx, "moduleData", properties)
		}
		return nil
	}
}

func moduleTelemetryData(mod *state.Module, ch state.ModuleChanges, store *state.StateStore) (map[string]interface{}, bool) {
	properties := make(map[string]interface{})
	hasChanged := ch.CoreRequirements || ch.Backend || ch.ProviderRequirements ||
		ch.TerraformVersion || ch.InstalledProviders

	if !hasChanged {
		return properties, false
	}

	if len(mod.Meta.CoreRequirements) > 0 {
		properties["tfRequirements"] = mod.Meta.CoreRequirements.String()
	}
	if mod.Meta.Cloud != nil {
		properties["cloud"] = true

		hostname := mod.Meta.Cloud.Hostname

		// https://developer.hashicorp.com/terraform/language/settings/terraform-cloud#usage-example
		// Required for Terraform Enterprise;
		// Defaults to app.terraform.io for Terraform Cloud
		if hostname == "" {
			hostname = "app.terraform.io"
		}

		// anonymize any non-default hostnames
		if hostname != "app.terraform.io" {
			hostname = "custom-hostname"
		}

		properties["cloud.hostname"] = hostname
	}
	if mod.Meta.Backend != nil {
		properties["backend"] = mod.Meta.Backend.Type
		if data, ok := mod.Meta.Backend.Data.(*backend.Remote); ok {
			hostname := data.Hostname

			// https://developer.hashicorp.com/terraform/language/settings/backends/remote#hostname
			// Defaults to app.terraform.io for Terraform Cloud
			if hostname == "" {
				hostname = "app.terraform.io"
			}

			// anonymize any non-default hostnames
			if hostname != "app.terraform.io" {
				hostname = "custom-hostname"
			}

			properties["backend.remote.hostname"] = hostname
		}
	}
	if len(mod.Meta.ProviderRequirements) > 0 {
		reqs := make(map[string]string, 0)
		for pAddr, cons := range mod.Meta.ProviderRequirements {
			if telemetry.IsPublicProvider(pAddr) {
				reqs[pAddr.String()] = cons.String()
				continue
			}

			// anonymize any unknown providers or the ones not publicly listed
			id, err := store.GetProviderID(pAddr)
			if err != nil {
				continue
			}
			addr := fmt.Sprintf("unlisted/%s", id)
			reqs[addr] = cons.String()
		}
		properties["providerRequirements"] = reqs
	}
	if mod.TerraformVersion != nil {
		properties["tfVersion"] = mod.TerraformVersion.String()
	}
	if len(mod.InstalledProviders) > 0 {
		installedProviders := make(map[string]string, 0)
		for pAddr, pv := range mod.InstalledProviders {
			if telemetry.IsPublicProvider(pAddr) {
				versionString := ""
				if pv != nil {
					versionString = pv.String()
				}
				installedProviders[pAddr.String()] = versionString
				continue
			}

			// anonymize any unknown providers or the ones not publicly listed
			id, err := store.GetProviderID(pAddr)
			if err != nil {
				continue
			}
			addr := fmt.Sprintf("unlisted/%s", id)
			installedProviders[addr] = ""
		}
		properties["installedProviders"] = installedProviders
	}

	if !hasChanged {
		return nil, false
	}

	modId, err := store.GetModuleID(mod.Path)
	if err != nil {
		return nil, false
	}
	properties["moduleId"] = modId

	return properties, true
}

func updateDiagnostics(dNotifier *diagnostics.Notifier) notifier.Hook {
	return func(ctx context.Context, changes state.ModuleChanges) error {
		if changes.Diagnostics {
			mod, err := notifier.ModuleFromContext(ctx)
			if err != nil {
				return err
			}

			diags := diagnostics.NewDiagnostics()
			diags.EmptyRootDiagnostic()

			defer dNotifier.PublishHCLDiags(ctx, mod.Path, diags)

			if mod != nil {
				diags.Append("HCL", mod.ModuleDiagnostics.AutoloadedOnly().AsMap())
				diags.Append("HCL", mod.VarsDiagnostics.AutoloadedOnly().AsMap())
			}
		}
		return nil
	}
}

func callRefreshClientCommand(clientRequester session.ClientCaller, commandId string) notifier.Hook {
	return func(ctx context.Context, changes state.ModuleChanges) error {
		// TODO: avoid triggering if module calls/providers did not change
		isOpen, err := notifier.ModuleIsOpen(ctx)
		if err != nil {
			return err
		}

		if isOpen {
			mod, err := notifier.ModuleFromContext(ctx)
			if err != nil {
				return err
			}

			_, err = clientRequester.Callback(ctx, commandId, nil)
			if err != nil {
				return fmt.Errorf("Error calling %s for %s: %s", commandId, mod.Path, err)
			}
		}

		return nil
	}
}

func refreshCodeLens(clientRequester session.ClientCaller) notifier.Hook {
	return func(ctx context.Context, changes state.ModuleChanges) error {
		// TODO: avoid triggering for new targets outside of open module
		if changes.ReferenceOrigins || changes.ReferenceTargets {
			_, err := clientRequester.Callback(ctx, "workspace/codeLens/refresh", nil)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func refreshSemanticTokens(clientRequester session.ClientCaller) notifier.Hook {
	return func(ctx context.Context, changes state.ModuleChanges) error {
		isOpen, err := notifier.ModuleIsOpen(ctx)
		if err != nil {
			return err
		}

		localChanges := isOpen && (changes.TerraformVersion || changes.CoreRequirements ||
			changes.InstalledProviders || changes.ProviderRequirements)

		if localChanges || changes.ReferenceOrigins || changes.ReferenceTargets {
			mod, err := notifier.ModuleFromContext(ctx)
			if err != nil {
				return err
			}

			_, err = clientRequester.Callback(ctx, "workspace/semanticTokens/refresh", nil)
			if err != nil {
				return fmt.Errorf("Error refreshing %s: %s", mod.Path, err)
			}
		}

		return nil
	}
}
