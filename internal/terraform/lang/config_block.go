package lang

import (
	"fmt"
	"log"
	"sort"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
)

type configBlockFactory interface {
	New(hclsyntax.Tokens) (ConfigBlock, error)
	LabelSchema() LabelSchema
	Documentation() MarkupContent
}

type labelCandidates map[string][]CompletionCandidate

type completableLabels struct {
	logger *log.Logger
	block  Block
	tokens hclsyntax.Tokens
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
	prefix := wordBeforePos(cl.tokens, pos)
	for _, c := range candidates {
		if !strings.HasPrefix(c.Label(), prefix) {
			continue
		}
		c.SetPrefix(prefix)
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
	tokens hclsyntax.Tokens
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

	prefix := wordBeforePos(cb.tokens, pos)
	for name, attr := range b.Attributes() {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if attr.IsComputedOnly() || attr.IsDeclared() {
			continue
		}
		list.candidates = append(list.candidates, &attributeCandidate{
			Name:   name,
			Attr:   attr,
			Pos:    pos,
			Prefix: prefix,
		})
	}

	for name, block := range b.BlockTypes() {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if block.ReachedMaxItems() {
			continue
		}
		list.candidates = append(list.candidates, &nestedBlockCandidate{
			Name:      name,
			BlockType: block,
			Pos:       pos,
			Prefix:    prefix,
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
	pos           hcl.Pos
	prefix        string
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

func (c *labelCandidate) Snippet() string {
	return c.label
}

func (c *labelCandidate) SetPrefix(prefix string) {
	c.prefix = prefix
}

func (c *labelCandidate) PrefixRange() hcl.Range {
	return hcl.Range{
		Start: hcl.Pos{
			Line:   c.pos.Line,
			Column: c.pos.Column - len(c.prefix),
		},
		End: c.pos,
	}
}

type attributeCandidate struct {
	Name   string
	Attr   *Attribute
	Pos    hcl.Pos
	Prefix string
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

func (c *attributeCandidate) Snippet() string {
	return fmt.Sprintf("%s = %s", c.Name, snippetForAttrType(0, c.Attr.Schema().AttributeType))
}

func (c *attributeCandidate) SetPrefix(prefix string) {
	c.Prefix = prefix
}

func (c *attributeCandidate) PrefixRange() hcl.Range {
	return hcl.Range{
		Start: hcl.Pos{
			Line:   c.Pos.Line,
			Column: c.Pos.Column - len(c.Prefix),
		},
		End: c.Pos,
	}
}

type nestedBlockCandidate struct {
	Name      string
	BlockType *BlockType
	Pos       hcl.Pos
	Prefix    string
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

func (c *nestedBlockCandidate) Snippet() string {
	return snippetForNestedBlock(c.Name)
}

func (c *nestedBlockCandidate) SetPrefix(prefix string) {
	c.Prefix = prefix
}

func (c *nestedBlockCandidate) PrefixRange() hcl.Range {
	return hcl.Range{
		Start: hcl.Pos{
			Line:   c.Pos.Line,
			Column: c.Pos.Column - len(c.Prefix),
		},
		End: c.Pos,
	}
}
