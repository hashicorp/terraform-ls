package source

import (
	"github.com/hashicorp/hcl/v2"
)

type Lines []Line

type Line interface {
	Range() hcl.Range
	Bytes() []byte
}
