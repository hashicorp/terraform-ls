package hcl

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	hcllib "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
)

type File interface {
	BlockAtPosition(filesystem.FilePosition) (*hcllib.Block, hcllib.Pos, error)
	Diff([]byte) (filesystem.FileChanges, error)
}

type file struct {
	filename string
	content  []byte
	f        *hcllib.File
}

func NewFile(f filesystem.File) File {
	return &file{
		filename: f.Filename(),
		content:  []byte(f.Text()),
	}
}

func (f *file) ast() (*hcllib.File, error) {
	if f.f != nil {
		return f.f, nil
	}

	hf, err := hclsyntax.ParseConfig(f.content, f.filename, hcllib.InitialPos)
	f.f = hf

	return f.f, err
}

func (f *file) BlockAtPosition(filePos filesystem.FilePosition) (*hcllib.Block, hcllib.Pos, error) {
	pos := filePos.Position()

	b, err := f.blockAtPosition(pos)
	if err != nil {
		return nil, pos, err
	}

	return b, pos, nil
}

func (f *file) Diff(target []byte) (filesystem.FileChanges, error) {
	var changes filesystem.FileChanges

	ast, _ := f.ast()
	body, ok := ast.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("invalid configuration format: %T", ast.Body)
	}

	changes = append(changes, &fileChange{
		newText: target,
		rng:     body.Range(),
	})

	return changes, nil
}

type fileChange struct {
	newText []byte
	rng     hcl.Range
}

func (ch *fileChange) Text() string {
	return string(ch.newText)
}

func (ch *fileChange) Range() hcl.Range {
	return ch.rng
}

func (f *file) blockAtPosition(pos hcllib.Pos) (*hcllib.Block, error) {
	ast, _ := f.ast()

	if body, ok := ast.Body.(*hclsyntax.Body); ok {
		if body.SrcRange.Empty() && pos != hcllib.InitialPos {
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
