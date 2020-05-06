package context

import "fmt"

type MissingContextErr struct {
	CtxKey *contextKey
}

func (e *MissingContextErr) Error() string {
	return fmt.Sprintf("missing context: %s", e.CtxKey)
}
