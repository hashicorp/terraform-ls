package document

import "fmt"

// Range represents LSP-style range between two positions
// Positions are zero-indexed
type Range struct {
	Start, End Pos
}

// Pos represents LSP-style position (zero-indexed)
type Pos struct {
	Line, Column int
}

func (p Pos) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}
