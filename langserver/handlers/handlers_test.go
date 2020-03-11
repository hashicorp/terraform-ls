package handlers

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

func TestMain(m *testing.M) {
	if v := os.Getenv("TF_LS_MOCK"); v != "" {
		os.Exit(exec.ExecuteMock(v))
		return
	}

	os.Exit(m.Run())
}
