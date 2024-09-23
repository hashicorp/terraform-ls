// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/hcl-lang/validator"
)

var validators = []validator.Validator{
	validator.BlockLabelsLength{},
	validator.DeprecatedAttribute{},
	validator.DeprecatedBlock{},
	validator.MaxBlocks{},
	validator.MinBlocks{},
	validator.MissingRequiredAttribute{},
	validator.UnexpectedAttribute{},
	validator.UnexpectedBlock{},
}
