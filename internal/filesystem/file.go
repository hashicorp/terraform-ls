package filesystem

import (
	"fmt"

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

func (f *file) applyChange(ch lsp.TextDocumentContentChangeEvent) error {
	if ch.Range != nil {
		return fmt.Errorf("Partial updates are not supported (yet)")
	}

	newBytes := []byte(ch.Text)
	f.change(newBytes)

	return nil
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
