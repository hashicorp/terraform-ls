package handlers

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/telemetry"
	"github.com/hashicorp/terraform-schema/backend"
)

func sendModuleTelemetry(ctx context.Context, store *state.StateStore, telemetrySender telemetry.Sender) state.ModuleChangeHook {
	return func(oldMod, newMod *state.Module) {
		if newMod == nil {
			// module is being removed
			// TODO: Track module removal as an event
			return
		}

		properties, hasChanged := moduleTelemetryData(oldMod, newMod, store)
		if !hasChanged {
			// avoid sending telemetry if nothing has changed
			return
		}

		telemetrySender.SendEvent(ctx, "moduleData", properties)
	}
}

func moduleTelemetryData(oldMod, newMod *state.Module, store *state.StateStore) (map[string]interface{}, bool) {
	properties := make(map[string]interface{})
	hasChanged := false

	if oldMod == nil || !oldMod.Meta.CoreRequirements.Equals(newMod.Meta.CoreRequirements) {
		hasChanged = true
	}
	if len(newMod.Meta.CoreRequirements) > 0 {
		properties["tfRequirements"] = newMod.Meta.CoreRequirements.String()
	}

	if oldMod == nil || !oldMod.Meta.Backend.Equals(newMod.Meta.Backend) {
		hasChanged = true
	}
	if newMod.Meta.Backend != nil {
		properties["backend"] = newMod.Meta.Backend.Type
		if data, ok := newMod.Meta.Backend.Data.(*backend.Remote); ok {
			hostname := data.Hostname

			// anonymize any non-default hostnames
			if hostname != "" && hostname != "app.terraform.io" {
				hostname = "custom-hostname"
			}

			properties["backend.remote.hostname"] = hostname
		}
	}

	if oldMod == nil || !oldMod.Meta.ProviderRequirements.Equals(newMod.Meta.ProviderRequirements) {
		hasChanged = true
	}
	if len(newMod.Meta.ProviderRequirements) > 0 {
		reqs := make(map[string]string, 0)
		for pAddr, cons := range newMod.Meta.ProviderRequirements {
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

	if oldMod == nil || !oldMod.TerraformVersion.Equal(newMod.TerraformVersion) {
		hasChanged = true
	}
	if newMod.TerraformVersion != nil {
		properties["tfVersion"] = newMod.TerraformVersion.String()
	}

	if oldMod == nil || !oldMod.InstalledProviders.Equals(newMod.InstalledProviders) {
		hasChanged = true
	}
	if len(newMod.InstalledProviders) > 0 {
		installedProviders := make(map[string]string, 0)
		for pAddr, pv := range newMod.InstalledProviders {
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

	modId, err := store.GetModuleID(newMod.Path)
	if err != nil {
		return nil, false
	}
	properties["moduleId"] = modId

	return properties, true
}

func updateDiagnostics(ctx context.Context, notifier *diagnostics.Notifier) state.ModuleChangeHook {
	return func(oldMod, newMod *state.Module) {
		oldDiags, newDiags := 0, 0
		if oldMod != nil {
			oldDiags = oldMod.ModuleDiagnostics.Count() + oldMod.VarsDiagnostics.Count()
		}
		if newMod != nil {
			newDiags = newMod.ModuleDiagnostics.Count() + newMod.VarsDiagnostics.Count()
		}

		if oldDiags == 0 && newDiags == 0 {
			return
		}

		diags := diagnostics.NewDiagnostics()
		diags.EmptyRootDiagnostic()

		defer notifier.PublishHCLDiags(ctx, newMod.Path, diags)

		if newMod != nil {
			diags.Append("HCL", newMod.ModuleDiagnostics.AsMap())
			diags.Append("HCL", newMod.VarsDiagnostics.AutoloadedOnly().AsMap())
		}
	}
}

func refreshCodeLens(ctx context.Context, clientRequester session.ClientCaller) state.ModuleChangeHook {
	return func(oldMod, newMod *state.Module) {
		oldOrigins, oldTargets := 0, 0
		if oldMod != nil {
			oldOrigins = len(oldMod.RefOrigins)
			oldTargets = len(oldMod.RefTargets)
		}
		newOrigins, newTargets := 0, 0
		if newMod != nil {
			newOrigins = len(newMod.RefOrigins)
			newTargets = len(newMod.RefTargets)
		}

		if oldOrigins != newOrigins || oldTargets != newTargets {
			clientRequester.Callback(ctx, "workspace/codeLens/refresh", nil)
		}
	}
}
