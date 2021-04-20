package state

import "fmt"

type AlreadyExistsError struct {
	Idx string
}

func (e *AlreadyExistsError) Error() string {
	if e.Idx != "" {
		return fmt.Sprintf("%s already exists", e.Idx)
	}
	return "already exists"
}

type NoSchemaError struct{}

func (e *NoSchemaError) Error() string {
	return "no schema found"
}

type ModuleNotFoundError struct {
	Path string
}

func (e *ModuleNotFoundError) Error() string {
	msg := "module not found"
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, msg)
	}

	return msg
}
