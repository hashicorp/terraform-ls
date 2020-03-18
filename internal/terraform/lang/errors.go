package lang

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2"
)

type emptyCfgErr struct {
}

func (e *emptyCfgErr) Error() string {
	return fmt.Sprintf("empty config")
}

var EmptyConfigErr = &emptyCfgErr{}

type invalidLabelsErr struct {
	BlockType string
	Labels    []string
}

func (e *invalidLabelsErr) Is(target error) bool {
	return reflect.DeepEqual(e, target)
}

func (e *invalidLabelsErr) Error() string {
	return fmt.Sprintf("invalid labels for %s block: %q", e.BlockType, e.Labels)
}

type unknownBlockTypeErr struct {
	BlockType string
}

func (e *unknownBlockTypeErr) Is(target error) bool {
	return reflect.DeepEqual(e, target)
}

func (e *unknownBlockTypeErr) Error() string {
	return fmt.Sprintf("unknown block type: %q", e.BlockType)
}

type unsupportedConfigTypeErr struct {
	Body hcl.Body
}

func (e *unsupportedConfigTypeErr) Error() string {
	return fmt.Sprintf("unsupported body type: %T", e.Body)
}

type noSchemaReaderErr struct {
	BlockType string
}

func (e *noSchemaReaderErr) Error() string {
	msg := "no schema reader available"
	if e.BlockType != "" {
		msg += fmt.Sprintf(" for %q", e.BlockType)
	}

	return msg
}
