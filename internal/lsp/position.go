package lsp

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

type filePosition struct {
	fh  *fileHandler
	pos hcl.Pos
}

func (p *filePosition) Position() hcl.Pos {
	return p.pos
}

func (p *filePosition) URI() string {
	return p.fh.URI()
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
	byteOffset, err := filesystem.ByteOffsetForPos(f.Lines(), lspPosToFsPos(params.Position))
	if err != nil {
		return nil, err
	}

	return &filePosition{
		fh: FileHandlerFromDocumentURI(params.TextDocument.URI),
		pos: hcl.Pos{
			Line:   int(params.Position.Line) + 1,
			Column: int(params.Position.Character) + 1,
			Byte:   byteOffset,
		},
	}, nil
}

func lspPosToFsPos(pos lsp.Position) filesystem.Pos {
	return filesystem.Pos{
		Line:   int(pos.Line),
		Column: int(pos.Character),
	}
}
