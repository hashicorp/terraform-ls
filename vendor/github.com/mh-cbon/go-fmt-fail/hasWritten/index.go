package hasWritten

import (
	"io"
)

type HasWritten struct {
	w       io.Writer
	Written bool
}

func New(w io.Writer) *HasWritten {
	h := HasWritten{w: w, Written: false}
	return &h
}

func (h *HasWritten) Write(p []byte) (n int, err error) {
	h.Written = true
	return h.w.Write(p)
}
