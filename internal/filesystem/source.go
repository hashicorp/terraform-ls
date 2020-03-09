package filesystem

import (
	"bufio"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/apparentlymart/go-textseg/textseg"
	"github.com/hashicorp/hcl/v2"
	lsp "github.com/sourcegraph/go-lsp"
)

type sourceLine struct {
	content []byte
	rng     hcl.Range
}

// allASCII returns true if the receiver is provably all ASCII, which allows
// for some fast paths where we can treat columns and bytes as equivalent.
func (l sourceLine) allASCII() bool {
	// If we have the same number of columns as bytes then our content is
	// all ASCII, since it clearly contains no multi-byte grapheme clusters.
	bytes := l.rng.End.Byte - l.rng.Start.Byte
	columns := l.rng.End.Column - l.rng.Start.Column
	return bytes == columns
}

// byteForLSPColumn takes an lsp.Position.Character value for the receving line
// and finds the byte offset of the start of the UTF-8 sequence that represents
// it in the overall source buffer. This is different than the byte returned
// by posForLSPColumn because it can return offsets that are partway through
// a grapheme cluster, while HCL positions always round to the nearest
// grapheme cluster.
//
// Note that even this can't produce an exact result; if the column index
// refers to the second unit of a UTF-16 surrogate pair then it is rounded
// down the first unit because UTF-8 sequences are not divisible in the same
// way.
func (l sourceLine) byteForLSPColumn(lspCol int) int {
	if lspCol < 0 {
		return l.rng.Start.Byte
	}

	// Easy path: if the entire line is ASCII then column counts are equivalent
	// in LSP vs. HCL aside from zero- vs. one-based counting.
	if l.allASCII() {
		return l.rng.Start.Byte + lspCol
	}

	// If there are non-ASCII characters then we need to edge carefully
	// along the line while counting UTF-16 code units in our UTF-8 buffer,
	// since LSP columns are a count of UTF-16 units.
	byteCt := 0
	utf16Ct := 0
	colIdx := 1
	remain := l.content
	for {
		if len(remain) == 0 { // ran out of characters on the line, so given column is invalid
			return l.rng.End.Byte
		}
		if utf16Ct >= lspCol { // we've found it
			return l.rng.Start.Byte + byteCt
		}
		// Unlike our other conversion functions we're intentionally using
		// individual UTF-8 sequences here rather than grapheme clusters because
		// an LSP position might point into the middle of a grapheme cluster.

		adv, chBytes, _ := textseg.ScanUTF8Sequences(remain, true)
		remain = remain[adv:]
		byteCt += adv
		colIdx++
		for len(chBytes) > 0 {
			r, l := utf8.DecodeRune(chBytes)
			chBytes = chBytes[l:]
			c1, c2 := utf16.EncodeRune(r)
			if c1 == 0xfffd && c2 == 0xfffd {
				utf16Ct++ // codepoint fits in one 16-bit unit
			} else {
				utf16Ct += 2 // codepoint requires a surrogate pair
			}
		}
	}
}

type sourceLines []sourceLine

func makeSourceLines(filename string, s []byte) sourceLines {
	var ret sourceLines
	sc := hcl.NewRangeScanner(s, filename, bufio.ScanLines)
	for sc.Scan() {
		ret = append(ret, sourceLine{
			content: sc.Bytes(),
			rng:     sc.Range(),
		})
	}
	if len(ret) == 0 {
		ret = append(ret, sourceLine{
			content: nil,
			rng: hcl.Range{
				Filename: filename,
				Start:    hcl.Pos{Line: 1, Column: 1},
				End:      hcl.Pos{Line: 1, Column: 1},
			},
		})
	}
	return ret
}

func (ls sourceLines) lspPosToHclPos(pos lsp.Position) (hcl.Pos, error) {
	if len(ls) == 0 {
		if pos.Character != 0 || pos.Line != 0 {
			return hcl.Pos{}, &InvalidLspPosErr{pos}
		}
		return hcl.Pos{Line: 1, Column: 1, Byte: 0}, nil
	}

	for i, srcLine := range ls {
		if i == pos.Line {
			byte := srcLine.byteForLSPColumn(pos.Character)
			return hcl.Pos{
				// LSP indexing is zero-based, HCL's is one-based
				Line:   i + 1,
				Column: pos.Character + 1,
				Byte:   byte,
			}, nil
		}
	}

	return hcl.Pos{}, &InvalidLspPosErr{pos}
}
