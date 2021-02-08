package errors

import (
	e "errors"

	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

func EnrichTfExecError(err error) error {
	if module.IsTerraformNotFound(err) {
		return e.New("Terraform (CLI) is required. " +
			"Please install Terraform or make it available in $PATH")
	}
	return err
}
