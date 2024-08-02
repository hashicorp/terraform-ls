// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/hcl-lang/validator"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/decoder/validations"
)

var stackValidators = []validator.Validator{
	validator.BlockLabelsLength{},
	validator.DeprecatedAttribute{},
	validator.DeprecatedBlock{},
	validator.MaxBlocks{},
	validator.MinBlocks{},
	validator.UnexpectedAttribute{},
	validator.UnexpectedBlock{},
	validations.MissingRequiredAttribute{},
	validations.StackBlockValidName{},
}
