package filesystem

import (
	"unicode/utf16"
	"unicode/utf8"

	"github.com/apparentlymart/go-textseg/textseg"
	"github.com/hashicorp/terraform-ls/internal/source"
)

func ByteOffsetForPos(lines source.Lines, pos Pos) (int, error) {
	if pos.Line+1 > len(lines) {
		return 0, &InvalidPosErr{Pos: pos}
	}

	return byteOffsetForLSPColumn(lines[pos.Line], pos.Column), nil
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
func byteOffsetForLSPColumn(l source.Line, lspCol int) int {
	if lspCol < 0 {
		return l.Range().Start.Byte
	}

	// Normally ASCII-only lines could be short-circuited here
	// but it's not as easy to tell whether a line is ASCII-only
	// based on column/byte differences as we also scan newlines
	// and a single line range technically spans 2 lines.

	// If there are non-ASCII characters then we need to edge carefully
	// along the line while counting UTF-16 code units in our UTF-8 buffer,
	// since LSP columns are a count of UTF-16 units.
	byteCt := 0
	utf16Ct := 0
	colIdx := 1
	remain := l.Bytes()
	for {
		if len(remain) == 0 { // ran out of characters on the line, so given column is invalid
			return l.Range().End.Byte
		}
		if utf16Ct >= lspCol { // we've found it
			return l.Range().Start.Byte + byteCt
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
