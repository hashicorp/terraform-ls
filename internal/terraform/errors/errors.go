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

func (utv *UnsupportedTerraformVersion) Is(err error) bool {
	te, ok := err.(*UnsupportedTerraformVersion)
	if !ok {
		return false
	}

	return te.Version == utv.Version
}

type NotInitializedErr struct {
	Dir string
}

func (e *NotInitializedErr) Is(err error) bool {
	_, ok := err.(*NotInitializedErr)
	if !ok {
		return false
	}

	return true
}

func (e *NotInitializedErr) Error() string {
	if e.Dir != "" {
		return fmt.Sprintf("workspace %s not initialized", e.Dir)
	}

	return fmt.Sprintf("workspace not initialized")
}
