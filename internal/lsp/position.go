package lsp

import (
	"unicode/utf16"
	"unicode/utf8"

	"github.com/apparentlymart/go-textseg/textseg"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/source"
	lsp "github.com/sourcegraph/go-lsp"
)

type filePosition struct {
	fh  FileHandler
	pos hcl.Pos
}

func (p *filePosition) Position() hcl.Pos {
	return p.pos
}

func (p *filePosition) DocumentURI() string {
	return p.fh.DocumentURI()
}

func (p *filePosition) FullPath() string {
	return p.fh.FullPath()
}

func (p *filePosition) Dir() string {
	return p.fh.Dir()
}

func (p *filePosition) Filename() string {
	return p.fh.Filename()
}

func FilePositionFromDocumentPosition(params lsp.TextDocumentPositionParams, f File) (*filePosition, error) {
	pos, err := lspPositionToHCL(f.Lines(), params.Position)
	if err != nil {
		return nil, err
	}

	return &filePosition{
		fh:  FileHandler(params.TextDocument.URI),
		pos: pos,
	}, nil
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
func byteForLSPColumn(l source.Line, lspCol int) int {
	if lspCol < 0 {
		return l.Range().Start.Byte
	}

	// Easy path: if the entire line is ASCII then column counts are equivalent
	// in LSP vs. HCL aside from zero- vs. one-based counting.
	if l.IsAllASCII() {
		return l.Range().Start.Byte + lspCol
	}

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

func lspPositionToHCL(lines []source.Line, pos lsp.Position) (hcl.Pos, error) {
	if len(lines) == 0 {
		if pos.Character != 0 || pos.Line != 0 {
			return hcl.Pos{}, &InvalidLspPosErr{pos}
		}
		return hcl.Pos{Line: 1, Column: 1, Byte: 0}, nil
	}

	for i, srcLine := range lines {
		if i == pos.Line {
			return hcl.Pos{
				// LSP indexing is zero-based, HCL's is one-based
				Line:   i + 1,
				Column: pos.Character + 1,
				Byte:   byteForLSPColumn(srcLine, pos.Character),
			}, nil
		}
	}

	return hcl.Pos{}, &InvalidLspPosErr{pos}
}
