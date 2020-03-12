package schema

import (
	"fmt"
)

type SchemaUnavailableErr struct {
	BlockType string
	FullName  string
}

func (e *SchemaUnavailableErr) Error() string {
	return fmt.Sprintf("schema unavailable for %s %q", e.BlockType, e.FullName)
}
