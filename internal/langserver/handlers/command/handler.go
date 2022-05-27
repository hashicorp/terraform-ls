package command

import (
	"log"

	"github.com/hashicorp/terraform-ls/internal/state"
)

type CmdHandler struct {
	StateStore *state.StateStore
	Logger     *log.Logger
}
