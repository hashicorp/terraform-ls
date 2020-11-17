package hcl

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/source"
	"github.com/pmezard/go-difflib/difflib"
)

type fileChange struct {
	newText string
	rng     *hcl.Range
	opCode  difflib.OpCode
}

func (ch *fileChange) Text() string {
	return ch.newText
}

func (ch *fileChange) Range() *filesystem.Range {
	if ch.rng == nil {
		return nil
	}

	return &filesystem.Range{
		Start: filesystem.Pos{
			Line:   ch.rng.Start.Line - 1,
			Column: ch.rng.Start.Column - 1,
		},
		End: filesystem.Pos{
			Line:   ch.rng.End.Line - 1,
			Column: ch.rng.End.Column - 1,
		},
	}
}

const (
	OpReplace = 'r'
	OpDelete  = 'd'
	OpInsert  = 'i'
	OpEqual   = 'e'
)

// Diff calculates difference between Document's content
// and after byte sequence and returns it as filesystem.DocumentChanges
func Diff(f filesystem.DocumentHandler, before, after []byte) filesystem.DocumentChanges {
	return diffLines(f.Filename(),
		source.MakeSourceLines(f.Filename(), before),
		source.MakeSourceLines(f.Filename(), after))
}

// diffLines calculates difference between two source.Lines
// and returns them as filesystem.DocumentChanges
func diffLines(filename string, beforeLines, afterLines source.Lines) filesystem.DocumentChanges {
	context := 3

	m := difflib.NewMatcher(
		source.StringLines(beforeLines),
		source.StringLines(afterLines))

	changes := make(filesystem.DocumentChanges, 0)

	for _, group := range m.GetGroupedOpCodes(context) {
		for _, c := range group {
			beforeStart, beforeEnd := c.I1, c.I2
			afterStart, afterEnd := c.J1, c.J2

			if c.Tag == OpEqual {
				continue
			}

			if c.Tag == OpReplace {
				var rng *hcl.Range
				var newBytes []byte

				for i, line := range beforeLines[beforeStart:beforeEnd] {
					if i == 0 {
						lr := line.Range()
						rng = &lr
						continue
					}
					rng.End = line.Range().End
				}

				for _, line := range afterLines[afterStart:afterEnd] {
					newBytes = append(newBytes, line.Bytes()...)
				}

				changes = append(changes, &fileChange{
					newText: string(newBytes),
					rng:     rng,
				})
				continue
			}

			if c.Tag == OpDelete {
				var deleteRng *hcl.Range
				for i, line := range beforeLines[beforeStart:beforeEnd] {
					if i == 0 {
						lr := line.Range()
						deleteRng = &lr
						continue
					}
					deleteRng.End = line.Range().End
				}
				changes = append(changes, &fileChange{
					newText: "",
					rng:     deleteRng,
					opCode:  c,
				})
				continue
			}

			if c.Tag == OpInsert {
				insertRng := &hcl.Range{}
				insertRng.Start = beforeLines[beforeStart-1].Range().End
				insertRng.End = beforeLines[beforeStart-1].Range().End
				var newBytes []byte

				for _, line := range afterLines[afterStart:afterEnd] {
					newBytes = append(newBytes, line.Bytes()...)
				}

				changes = append(changes, &fileChange{
					newText: string(newBytes),
					rng:     insertRng,
				})
				continue
			}

		}
	}

	return changes
}
