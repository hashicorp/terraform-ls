// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/hcl-lang/validator"
)

var varsValidators = []validator.Validator{
	validator.UnexpectedAttribute{},
	validator.UnexpectedBlock{},
}
