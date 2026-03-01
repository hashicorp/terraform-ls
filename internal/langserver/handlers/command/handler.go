// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"log"

	"github.com/hashicorp/hcl-lang/decoder"
	fmodules "github.com/hashicorp/terraform-ls/internal/features/modules"
	frootmodules "github.com/hashicorp/terraform-ls/internal/features/rootmodules"
	"github.com/hashicorp/terraform-ls/internal/state"
)

type CmdHandler struct {
	StateStore *state.StateStore
	Logger     *log.Logger
	// TODO? Can features contribute commands, so we don't have to import
	// the features here?
	ModulesFeature     *fmodules.ModulesFeature
	RootModulesFeature *frootmodules.RootModulesFeature
	Decoder            *decoder.Decoder
}
