package command

import "github.com/hashicorp/terraform-ls/internal/state"

type CmdHandler struct {
	StateStore *state.StateStore
}
