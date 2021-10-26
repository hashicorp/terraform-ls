package handlers

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/telemetry"
	"github.com/hashicorp/terraform-schema/backend"
)

func sendModuleTelemetry(ctx context.Context, store *state.StateStore, telemetrySender telemetry.Sender) state.ModuleChangeHook {
	return func(_, newMod *state.Module) {
		modId, err := store.GetModuleID(newMod.Path)
		if err != nil {
			return
		}

		properties := map[string]interface{}{
			"moduleId": modId,
		}

		if len(newMod.Meta.CoreRequirements) > 0 {
			properties["tfRequirements"] = newMod.Meta.CoreRequirements.String()
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

		if newMod.TerraformVersion != nil {
			properties["tfVersion"] = newMod.TerraformVersion.String()
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

		telemetrySender.SendEvent(ctx, "moduleData", properties)
	}
}
