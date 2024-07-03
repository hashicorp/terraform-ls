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
)

func sendModuleTelemetry(features *Features, telemetrySender telemetry.Sender) notifier.Hook {
	return func(ctx context.Context, changes state.Changes) error {
		if changes.IsRemoval {
			// we ignore removed modules for now
			return nil
		}

		hasChanged := changes.CoreRequirements || changes.Backend || changes.ProviderRequirements ||
			changes.TerraformVersion || changes.InstalledProviders

		if !hasChanged {
			return nil
		}

		path, err := notifier.RecordPathFromContext(ctx)
		if err != nil {
			return err
		}

		// Query and merge telemetry from all modules
		// We assume there are no conflicting property keys
		properties := features.Modules.Telemetry(path)
		rootTelemetry := features.RootModules.Telemetry(path)
		for property, value := range rootTelemetry {
			properties[property] = value
		}

		telemetrySender.SendEvent(ctx, "moduleData", properties)

		return nil
	}
}

func updateDiagnostics(features *Features, dNotifier *diagnostics.Notifier) notifier.Hook {
	return func(ctx context.Context, changes state.Changes) error {
		if changes.Diagnostics {
			path, err := notifier.RecordPathFromContext(ctx)
			if err != nil {
				return err
			}

			diags := diagnostics.NewDiagnostics()
			diags.EmptyRootDiagnostic()

			diags.Extend(features.Modules.Diagnostics(path))
			diags.Extend(features.Variables.Diagnostics(path))
			diags.Extend(features.Stacks.Diagnostics(path))

			dNotifier.PublishHCLDiags(ctx, path, diags)
		}
		return nil
	}
}

func callRefreshClientCommand(clientRequester session.ClientCaller, commandId string) notifier.Hook {
	return func(ctx context.Context, changes state.Changes) error {
		// TODO: avoid triggering if module calls/providers did not change
		isOpen, err := notifier.RecordIsOpen(ctx)
		if err != nil {
			return err
		}

		if isOpen {
			path, err := notifier.RecordPathFromContext(ctx)
			if err != nil {
				return err
			}

			_, err = clientRequester.Callback(ctx, commandId, nil)
			if err != nil {
				return fmt.Errorf("error calling %s for %s: %s", commandId, path, err)
			}
		}

		return nil
	}
}

func refreshCodeLens(clientRequester session.ClientCaller) notifier.Hook {
	return func(ctx context.Context, changes state.Changes) error {
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
	return func(ctx context.Context, changes state.Changes) error {
		isOpen, err := notifier.RecordIsOpen(ctx)
		if err != nil {
			return err
		}

		localChanges := isOpen && (changes.TerraformVersion || changes.CoreRequirements ||
			changes.InstalledProviders || changes.ProviderRequirements)

		if localChanges || changes.ReferenceOrigins || changes.ReferenceTargets {
			path, err := notifier.RecordPathFromContext(ctx)
			if err != nil {
				return err
			}

			_, err = clientRequester.Callback(ctx, "workspace/semanticTokens/refresh", nil)
			if err != nil {
				return fmt.Errorf("error refreshing %s: %s", path, err)
			}
		}

		return nil
	}
}
