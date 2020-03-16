package errors

import (
	"fmt"

	"github.com/hashicorp/go-version"
)

type UnsupportedTerraformVersion struct {
	Component   string
	Version     string
	Constraints version.Constraints
}

func (utv *UnsupportedTerraformVersion) Error() string {
	msg := "terraform version is not supported"
	if utv.Version != "" {
		msg = fmt.Sprintf("terraform version %s is not supported", utv.Version)
	}

	if utv.Component != "" {
		msg += fmt.Sprintf(" in %s", utv.Component)
	}

	if utv.Constraints != nil {
		msg += fmt.Sprintf(" (supported: %s)", utv.Constraints.String())
	}

	return msg
}
