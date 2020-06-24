package rootmodule

import (
	"fmt"
)

type RootModuleNotFoundErr struct {
	Dir string
}

func (e *RootModuleNotFoundErr) Error() string {
	if e.Dir != "" {
		return fmt.Sprintf("root module not found for %s", e.Dir)
	}
	return "root module not found"
}

func IsRootModuleNotFound(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*RootModuleNotFoundErr)
	return ok
}
