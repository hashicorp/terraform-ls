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
