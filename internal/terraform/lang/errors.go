package lang

import "fmt"

type InvalidLabelsErr struct {
	BlockType string
	Labels    []string
}

func (e *InvalidLabelsErr) Error() string {
	return fmt.Sprintf("invalid labels for %s block: %q", e.BlockType, e.Labels)
}

type emptyConfigErr struct {
}

func (e *emptyConfigErr) Error() string {
	return fmt.Sprintf("empty config")
}

func EmptyConfigErr() *emptyConfigErr {
	return &emptyConfigErr{}
}

type SchemaUnavailableErr struct {
	BlockType string
	FullName  string
}

func (e *SchemaUnavailableErr) Error() string {
	return fmt.Sprintf("schema unavailable for %s %q", e.BlockType, e.FullName)
}
