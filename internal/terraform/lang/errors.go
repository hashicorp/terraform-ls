package lang

import (
	"fmt"
	"reflect"
)

type emptyCfgErr struct {
}

func (e *emptyCfgErr) Error() string {
	return "empty config"
}

var EmptyConfigErr = &emptyCfgErr{}

type unknownBlockTypeErr struct {
	BlockType string
}

func (e *unknownBlockTypeErr) Is(target error) bool {
	return reflect.DeepEqual(e, target)
}

func (e *unknownBlockTypeErr) Error() string {
	return fmt.Sprintf("unknown block type: %q", e.BlockType)
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

type UnknownProviderErr struct{}

func (e *UnknownProviderErr) Error() string {
	return "unknown provider"
}
