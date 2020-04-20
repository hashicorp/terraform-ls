package lang

import (
	"log"

	hcl "github.com/hashicorp/hcl/v2"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
	lsp "github.com/sourcegraph/go-lsp"
)

// Parser implements a parser which can turn raw HCL block
// into ConfigBlock with the help of a schema reader
type Parser interface {
	SetLogger(*log.Logger)
	SetCapabilities(lsp.TextDocumentClientCapabilities)
	SetSchemaReader(schema.Reader)
	ParseBlockFromHCL(*hcl.Block) (ConfigBlock, error)
}

// ConfigBlock implements an abstraction above HCL block
// which provides any LSP capabilities (e.g. completion)
type ConfigBlock interface {
	CompletionItemsAtPos(pos hcl.Pos) (lsp.CompletionList, error)
	Name() string
	BlockType() string
	Labels() []*Label
}

// Block represents a decoded HCL block (by a Parser)
// which keeps track of the related schema
type Block interface {
	BlockAtPos(pos hcl.Pos) (Block, bool)
	Range() hcl.Range
	PosInBody(pos hcl.Pos) bool
	PosInAttribute(pos hcl.Pos) bool
	Attributes() map[string]*Attribute
	BlockTypes() map[string]*BlockType
}

type LabelSchema []string

type Label struct {
	Name  string
	Value string
}

type BlockType struct {
	BlockList []Block
	schema    *tfjson.SchemaBlockType
}

type Attribute struct {
	schema       *tfjson.SchemaAttribute
	hclAttribute *hcl.Attribute
}
