package hcl

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/source"
	"github.com/pmezard/go-difflib/difflib"
)

type fileChange struct {
	newText string
	rng     hcl.Range
	opCode  difflib.OpCode
}

func (ch *fileChange) Text() string {
	return ch.newText
}

func (ch *fileChange) Range() hcl.Range {
	return ch.rng
}

const (
	OpReplace = 'r'
	OpDelete  = 'd'
	OpInsert  = 'i'
	OpEqual   = 'e'
)

// Diff calculates difference between File's content
// and after byte sequence and returns it as filesystem.FileChanges
func Diff(f filesystem.File, after []byte) filesystem.FileChanges {
	return diffLines(f.Filename(),
		source.MakeSourceLines(f.Filename(), f.Text()),
		source.MakeSourceLines(f.Filename(), after))
}

// diffLines calculates difference between two source.Lines
// and returns them as filesystem.FileChanges
func diffLines(filename string, beforeLines, afterLines source.Lines) filesystem.FileChanges {
	context := 3

	m := difflib.NewMatcher(
		source.StringLines(beforeLines),
		source.StringLines(afterLines))

	changes := make(filesystem.FileChanges, 0)

	for _, group := range m.GetGroupedOpCodes(context) {
		for _, c := range group {
			beforeStart, beforeEnd := c.I1, c.I2
			afterStart, afterEnd := c.J1, c.J2

			if c.Tag == OpEqual {
				continue
			}

			if c.Tag == OpReplace {
				var rng hcl.Range
				var newBytes []byte

				for i, line := range beforeLines[beforeStart:beforeEnd] {
					if i == 0 {
						rng = line.Range()
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
				var deleteRng hcl.Range
				for i, line := range beforeLines[beforeStart:beforeEnd] {
					if i == 0 {
						deleteRng = line.Range()
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
				var insertRng hcl.Range
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
