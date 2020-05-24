package lang

import (
	"fmt"
	"log"
	"sort"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
)

type configBlockFactory interface {
	New(*hclsyntax.Block) (ConfigBlock, error)
	LabelSchema() LabelSchema
	Documentation() MarkupContent
}

type labelCandidates map[string][]CompletionCandidate

type completableLabels struct {
	logger *log.Logger
	block  Block
	labels labelCandidates
}

func (cl *completableLabels) completionCandidatesAtPos(pos hcl.Pos) (CompletionCandidates, error) {
	list := &completeList{
		candidates: make([]CompletionCandidate, 0),
	}
	l, ok := cl.block.LabelAtPos(pos)
	if !ok {
		cl.logger.Printf("label not found at %#v", pos)
		return list, nil
	}
	candidates, ok := cl.labels[l.Name]
	if !ok {
		cl.logger.Printf("label %q doesn't have completion candidates", l.Name)
		return list, nil
	}

	cl.logger.Printf("completing label %q ...", l.Name)
	for _, c := range candidates {
		list.candidates = append(list.candidates, c)
	}
	list.Sort()

	return list, nil
}

// completableBlock provides common completion functionality
// for any Block implementation
type completableBlock struct {
	logger *log.Logger
	block  Block
}

func (cb *completableBlock) completionCandidatesAtPos(pos hcl.Pos) (CompletionCandidates, error) {
	list := &completeList{
		candidates: make([]CompletionCandidate, 0),
	}

	if !cb.block.PosInBody(pos) {
		cb.logger.Println("avoiding completion outside of block body")
		return nil, nil
	}

	if cb.block.PosInAttribute(pos) {
		cb.logger.Println("avoiding completion in the middle of existing attribute")
		return nil, nil
	}

	// Completing the body (attributes and nested blocks)
	b, ok := cb.block.BlockAtPos(pos)
	if !ok {
		// This should never happen as the completion
		// should only be called on a block the "pos" points to
		cb.logger.Printf("block type not found at %#v", pos)
		return nil, nil
	}

	for name, attr := range b.Attributes() {
		if attr.IsComputedOnly() || attr.IsDeclared() {
			continue
		}
		list.candidates = append(list.candidates, &attributeCandidate{
			Name: name,
			Attr: attr,
			Pos:  pos,
		})
	}

	for name, block := range b.BlockTypes() {
		if block.ReachedMaxItems() {
			continue
		}
		list.candidates = append(list.candidates, &nestedBlockCandidate{
			Name:      name,
			BlockType: block,
			Pos:       pos,
		})
	}

	list.Sort()

	return list, nil
}

type completeList struct {
	candidates []CompletionCandidate
}

func (l *completeList) Sort() {
	less := func(i, j int) bool {
		return l.candidates[i].Label() < l.candidates[j].Label()
	}
	sort.Slice(l.candidates, less)

}

func (l *completeList) List() []CompletionCandidate {
	return l.candidates
}

func (l *completeList) Len() int {
	return len(l.candidates)
}

func (l *completeList) IsComplete() bool {
	return true
}

type labelCandidate struct {
	label         string
	detail        string
	documentation MarkupContent
}

func (c *labelCandidate) Label() string {
	return c.label
}

func (c *labelCandidate) Detail() string {
	return c.detail
}

func (c *labelCandidate) Documentation() MarkupContent {
	return c.documentation
}

func (c *labelCandidate) Snippet(pos hcl.Pos) (hcl.Pos, string) {
	return pos, c.label
}

type attributeCandidate struct {
	Name string
	Attr *Attribute
	Pos  hcl.Pos
}

func (c *attributeCandidate) Label() string {
	return c.Name
}

func (c *attributeCandidate) Detail() string {
	if c.Attr == nil {
		return ""
	}
	return schemaAttributeDetail(c.Attr.Schema())
}

func (c *attributeCandidate) Documentation() MarkupContent {
	if c.Attr == nil {
		return PlainText("")
	}
	if schema := c.Attr.Schema(); schema != nil {
		if schema.DescriptionKind == tfjson.SchemaDescriptionKindMarkdown {
			return Markdown(schema.Description)
		}
		return PlainText(schema.Description)
	}
	return PlainText("")
}

func (c *attributeCandidate) Snippet(pos hcl.Pos) (hcl.Pos, string) {
	return pos, fmt.Sprintf("%s = %s", c.Name, snippetForAttrType(0, c.Attr.Schema().AttributeType))
}

type nestedBlockCandidate struct {
	Name      string
	BlockType *BlockType
	Pos       hcl.Pos
}

func (c *nestedBlockCandidate) Label() string {
	return c.Name
}

func (c *nestedBlockCandidate) Detail() string {
	return schemaBlockDetail(c.BlockType)
}

func (c *nestedBlockCandidate) Documentation() MarkupContent {
	if c.BlockType == nil || c.BlockType.Schema() == nil || c.BlockType.Schema().Block == nil {
		return PlainText("")
	}
	if c.BlockType.Schema().Block.DescriptionKind == tfjson.SchemaDescriptionKindMarkdown {
		return Markdown(c.BlockType.Schema().Block.Description)
	}
	return PlainText(c.BlockType.Schema().Block.Description)
}

func (c *nestedBlockCandidate) Snippet(pos hcl.Pos) (hcl.Pos, string) {
	return pos, snippetForNestedBlock(c.Name)
}
