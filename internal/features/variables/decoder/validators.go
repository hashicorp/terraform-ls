// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/hcl-lang/validator"
)

var varsValidators = []validator.Validator{
	validator.UnexpectedAttribute{},
	validator.UnexpectedBlock{},
}
