// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/langserver/progress"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

const terraformVersionRequestVersion = 0

type terraformInfoResponse struct {
	FormatVersion     int    `json:"v"`
	RequiredVersion   string `json:"required_version,omitempty"`
	DiscoveredVersion string `json:"discovered_version,omitempty"`
}

func (h *CmdHandler) TerraformVersionRequestHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	progress.Begin(ctx, "Initializing")
	defer func() {
		progress.End(ctx, "Finished")
	}()

	response := terraformInfoResponse{
		FormatVersion: terraformVersionRequestVersion,
	}

	progress.Report(ctx, "Finding current module info ...")
	modUri, ok := args.GetString("uri")
	if !ok || modUri == "" {
		return response, fmt.Errorf("%w: expected module uri argument to be set", jrpc2.InvalidParams.Err())
	}

	if !uri.IsURIValid(modUri) {
		return response, fmt.Errorf("URI %q is not valid", modUri)
	}

	modPath, err := uri.PathFromURI(modUri)
	if err != nil {
		return response, err
	}

	progress.Report(ctx, "Recording terraform version info ...")

	terraformVersion := h.RootModulesFeature.TerraformVersion(modPath)
	if terraformVersion != nil {
		response.DiscoveredVersion = terraformVersion.String()
	}

	coreRequirements, err := h.ModulesFeature.CoreRequirements(modPath)
	if err != nil {
		return response, err
	}
	if coreRequirements != nil {
		response.RequiredVersion = coreRequirements.String()
	}

	progress.Report(ctx, "Sending response ...")

	return response, nil
}
