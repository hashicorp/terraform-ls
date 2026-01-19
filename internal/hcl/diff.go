// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package hcl

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/document"
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

func (ch *fileChange) Range() *document.Range {
	if ch.rng == nil {
		return nil
	}

	return &document.Range{
		Start: document.Pos{
			Line:   ch.rng.Start.Line - 1,
			Column: ch.rng.Start.Column - 1,
		},
		End: document.Pos{
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
// and after byte sequence and returns it as document.Changes
func Diff(f document.Handle, before, after []byte) document.Changes {
	return diffLines(f.Filename,
		source.MakeSourceLines(f.Filename, before),
		source.MakeSourceLines(f.Filename, after))
}

// diffLines calculates difference between two source.Lines
// and returns them as document.Changes
func diffLines(filename string, beforeLines, afterLines source.Lines) document.Changes {
	context := 3

	m := difflib.NewMatcher(
		source.StringLines(beforeLines),
		source.StringLines(afterLines))

	changes := make(document.Changes, 0)

	for _, group := range m.GetGroupedOpCodes(context) {
		for _, c := range group {
			if c.Tag == OpEqual {
				continue
			}

			// lines to pick from the original document (to delete/replace/insert to)
			beforeStart, beforeEnd := c.I1, c.I2
			// lines to pick from the new document (to replace ^ with)
			afterStart, afterEnd := c.J1, c.J2

			if c.Tag == OpReplace {
				var rng *hcl.Range
				var newBytes []byte

				for i, line := range beforeLines[beforeStart:beforeEnd] {
					if i == 0 {
						lr := line.Range
						rng = &lr
						continue
					}
					rng.End = line.Range.End
				}

				for _, line := range afterLines[afterStart:afterEnd] {
					newBytes = append(newBytes, line.Bytes...)
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
						lr := line.Range
						deleteRng = &lr
						continue
					}
					deleteRng.End = line.Range.End
				}
				changes = append(changes, &fileChange{
					newText: "",
					rng:     deleteRng,
					opCode:  c,
				})
				continue
			}

			if c.Tag == OpInsert {
				insertRng := &hcl.Range{
					Filename: filename,
					Start:    hcl.InitialPos,
					End:      hcl.InitialPos,
				}

				if beforeStart == beforeEnd {
					line := beforeLines[beforeStart]
					insertRng = line.Range.Ptr()

					// We're inserting to the beginning of the line
					// which we represent as 0-length range in HCL
					insertRng.End = insertRng.Start
				} else {
					for i, line := range beforeLines[beforeStart:beforeEnd] {
						if i == 0 {
							insertRng = line.Range.Ptr()
							continue
						}
						insertRng.End = line.Range.End
					}
				}

				var newBytes []byte

				for _, line := range afterLines[afterStart:afterEnd] {
					newBytes = append(newBytes, line.Bytes...)
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
