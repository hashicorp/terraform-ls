package schema

import (
	"fmt"
)

type NoSchemaAvailableErr struct{}

func (e *NoSchemaAvailableErr) Error() string {
	return "no schema available"
}

type SchemaUnavailableErr struct {
	BlockType string
	FullName  string
}

func (e *SchemaUnavailableErr) Error() string {
	return fmt.Sprintf("schema unavailable for %s %q", e.BlockType, e.FullName)
}
