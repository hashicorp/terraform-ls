package filesystem

import (
	"unicode/utf16"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	lsp "github.com/sourcegraph/go-lsp"
	encunicode "golang.org/x/text/encoding/unicode"
)

var utf16encoding = encunicode.UTF16(encunicode.LittleEndian, encunicode.IgnoreBOM)
var utf16encoder = utf16encoding.NewEncoder()
var utf16decoder = utf16encoding.NewDecoder()

type file struct {
	fullPath string
	content  []byte
	open     bool

	ls   sourceLines
	errs bool
	ast  *hcl.File
}

func NewFile(fullPath string, content []byte) *file {
	return &file{fullPath: fullPath, content: content}
}

func (f *file) lines() sourceLines {
	if f.ls == nil {
		f.ls = makeSourceLines(f.fullPath, f.content)
	}
	return f.ls
}

func (f *file) HclBlockAtPos(pos hcl.Pos) (*hcl.Block, error) {
	ast, err := f.hclAST()
	if err != nil {
		return nil, err
	}

	if body, ok := ast.Body.(*hclsyntax.Body); ok {
		if body.SrcRange.Empty() && pos != hcl.InitialPos {
			return nil, &InvalidHclPosErr{pos, body.SrcRange}
		}
		if !body.SrcRange.Empty() && !body.SrcRange.ContainsPos(pos) {
			return nil, &InvalidHclPosErr{pos, body.SrcRange}
		}
	}

	block := ast.OutermostBlockAtPos(pos)
	if block == nil {
		return nil, &NoBlockFoundErr{pos}
	}

	return block, nil
}

func (f *file) LspPosToHCLPos(pos lsp.Position) (hcl.Pos, error) {
	return f.lines().lspPosToHclPos(pos)
}

func (f *file) applyChange(ch lsp.TextDocumentContentChangeEvent) {
	if ch.Range == nil {
		newBytes := []byte(ch.Text)
		f.change(newBytes)
		return
	}

	// Change positions/lengths are described in UTF-16 code units relative
	// to the start of a line, so to ensure we apply exactly what the client
	// is requesting (including weird conditions like typing into the middle
	// of a UTF-16 surrogate pair) we will transcode to UTF-16, apply the
	// edit, and transcode back. This can potentially cause a lot of churn
	// of large buffers, so we may wish to optimize this more in future, but
	// at least for now we'll limit the window of the buffer that we convert
	// to UTF-16.

	ls := f.lines()
	startLine := int(ch.Range.Start.Line)
	endLine := int(ch.Range.End.Line)
	if startLine < 0 {
		startLine = 0
	}
	if endLine >= len(ls) {
		endLine = len(ls) - 1
	}
	startChar := int(ch.Range.Start.Character)
	endChar := int(ch.Range.End.Character)

	startByte := ls[startLine].rng.Start.Byte
	endByte := ls[endLine].rng.End.Byte
	lastLineStartByte := ls[endLine].rng.Start.Byte
	// We take some care to avoid panics here but none of these situations
	// should actually arise for a well-behaved client.
	if lastLineStartByte > endByte {
		lastLineStartByte = endByte
	}
	if startByte > lastLineStartByte {
		startByte = lastLineStartByte
	}
	if startByte < 0 {
		startByte = 0
	}
	if endByte > len(f.content) {
		endByte = len(f.content) - 1
	}
	if lastLineStartByte > len(f.content) {
		lastLineStartByte = len(f.content) - 1
	}

	inU8buf := f.content[startByte:endByte]
	// We need to figure out now where in the UTF-16 buffer our lastLineStartByte
	// will end up, so we can properly slice using our end position's character.
	lastLineStartByteU16 := 0
	for b := inU8buf[:lastLineStartByte-startByte]; len(b) > 0; {
		r, l := utf8.DecodeRune(b)
		b = b[l:]
		if r1, r2 := utf16.EncodeRune(r); r1 == 0xfffd && r2 == 0xfffd {
			lastLineStartByteU16 += 2 // encoded as one 16-bit unit
		} else {
			lastLineStartByteU16 += 4 // encoded as two 16-bit units
		}
	}

	inU16buf, err := utf16encoder.Bytes(inU8buf)
	if err != nil {
		// Should never happen since errors are handled by inserting marker characters
		panic("utf16encoder failed")
	}

	replU16buf, err := utf16encoder.Bytes([]byte(ch.Text))
	if err != nil {
		panic("utf16encoder failed")
	}

	outU16BufLen := len(inU16buf) - (int(ch.RangeLength) * 2) + len(replU16buf)
	outU16Buf := make([]byte, 0, outU16BufLen)
	outU16Buf = append(outU16Buf, inU16buf[:startChar*2]...)
	outU16Buf = append(outU16Buf, replU16buf...)
	outU16Buf = append(outU16Buf, inU16buf[lastLineStartByteU16+endChar*2:]...)

	outU8Buf, err := utf16decoder.Bytes(outU16Buf)
	if err != nil {
		panic("utf16decoder failed")
	}

	var resultBuf []byte
	resultBuf = append(resultBuf, f.content[:startByte]...)
	resultBuf = append(resultBuf, outU8Buf...)
	resultBuf = append(resultBuf, f.content[endByte:]...)

	f.change(resultBuf)
}

func (f *file) makeTextEdits(new []byte) []lsp.TextEdit {
	oldLs := f.lines()
	newLs := makeSourceLines(f.fullPath, new)
	return makeTextEdits(oldLs, newLs, 0.15)
}

func (f *file) hclAST() (*hcl.File, error) {
	if f.ast != nil {
		return f.ast, nil
	}

	hf, diags := hclsyntax.ParseConfig(f.content, f.fullPath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}
	f.ast = hf

	return hf, nil
}

func (f *file) change(s []byte) {
	f.content = s
	f.ls = nil
	f.ast = nil
}
