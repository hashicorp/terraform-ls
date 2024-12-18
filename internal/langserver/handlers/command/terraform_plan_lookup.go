// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

const terraformPlanLookupRequestVersion = 0

type vscodeRange struct {
	StartLine      int `json:"startLine"`
	StartCharacter int `json:"startCharacter"`
	EndLine        int `json:"endLine"`
	EndCharacter   int `json:"endCharacter"`
}

type terraformPlanLookupResponse struct {
	FormatVersion int         `json:"v"`
	Range         vscodeRange `json:"range"`
	FileUri       string      `json:"fileUri"`
}

func (h *CmdHandler) TerraformPlanLookupHandler(ctx context.Context, args cmd.CommandArgs) (interface{}, error) {
	response := terraformPlanLookupResponse{
		FormatVersion: terraformPlanLookupRequestVersion,
	}

	// NOTE: Do not return errors for this handler as this is
	// for hyperlinks in a terminal. We would spam message boxes
	// if a user hovered over things that were not valid

	// Get the module dir, if it exists
	modUri, ok := args.GetString("uri")
	if !ok || modUri == "" {
		return response, nil
	}

	// make sure this is a valid uri
	if !uri.IsURIValid(modUri) {
		return response, nil
	}

	// get the module path
	modPath, err := uri.PathFromURI(modUri)
	if err != nil {
		return response, nil
	}

	// find the mod state in memdb
	mod, _ := h.StateStore.Modules.ModuleByPath(modPath)
	if mod == nil {
		return response, nil
	}

	line, _ := args.GetString("line")

	// iterate through the ref targets and find the one that matches
	// the address given by the client
	for _, target := range mod.RefTargets {
		if target.Addr.String() == line {
			response.FileUri = fmt.Sprintf("%v/%v", modUri, target.RangePtr.Filename)

			// TODO: these lines are off because of the drift between
			// what terraform plan shows versus what is in the actual file
			response.Range.StartLine = target.RangePtr.Start.Line-1
			response.Range.StartCharacter = target.RangePtr.Start.Column-1
			response.Range.EndLine = target.RangePtr.End.Line-1
			response.Range.EndCharacter = target.RangePtr.End.Column-1

			return response, nil
		}
	}

	return response, nil
}
