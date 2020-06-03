package lang

import (
	"fmt"
	"log"
	"sort"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	tfjson "github.com/hashicorp/terraform-json"
	ihcl "github.com/hashicorp/terraform-ls/internal/hcl"
)

type configBlockFactory interface {
	New(ihcl.TokenizedBlock) (ConfigBlock, error)
	LabelSchema() LabelSchema
	Documentation() MarkupContent
}

type labelCandidates map[string][]*labelCandidate

type completableLabels struct {
	logger        *log.Logger
	maxCandidates int
	parsedLabels  []*ParsedLabel
	tBlock        ihcl.TokenizedBlock
	labels        labelCandidates
}

func (cl *completableLabels) maxCompletionCandidates() int {
	if cl.maxCandidates > 0 {
		return cl.maxCandidates
	}
	return defaultMaxCompletionCandidates
}

func (cl *completableLabels) completionCandidatesAtPos(pos hcl.Pos) (CompletionCandidates, error) {
	list := &candidateList{
		candidates: make([]CompletionCandidate, 0),
	}
	l, ok := LabelAtPos(cl.parsedLabels, pos)
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

	prefix, prefixRng := prefixAtPos(cl.tBlock, pos)

	for _, c := range candidates {
		if len(list.candidates) >= cl.maxCompletionCandidates() {
			list.isIncomplete = true
			break
		}
		if !strings.HasPrefix(c.Label(), prefix) {
			continue
		}
		c.prefixRng = prefixRng
		list.candidates = append(list.candidates, c)
	}
	list.Sort()

	return list, nil
}

// completableBlock provides common completion functionality
// for any Block implementation
type completableBlock struct {
	logger        *log.Logger
	maxCandidates int
	parsedLabels  []*ParsedLabel
	tBlock        ihcl.TokenizedBlock
	schema        *tfjson.SchemaBlock
}

func (cl *completableBlock) maxCompletionCandidates() int {
	if cl.maxCandidates > 0 {
		return cl.maxCandidates
	}
	return defaultMaxCompletionCandidates
}

func (cb *completableBlock) completionCandidatesAtPos(pos hcl.Pos) (CompletionCandidates, error) {
	list := &candidateList{
		candidates: make([]CompletionCandidate, 0),
	}

	block := ParseBlock(cb.tBlock, cb.schema)

	if !block.PosInBody(pos) {
		// TODO: Allow this (requires access to the parser/all block types here)
		cb.logger.Println("avoiding completion outside of block body")
		return nil, nil
	}

	// Completing the body (attributes and nested blocks)
	b, ok := block.BlockAtPos(pos)
	if !ok {
		// This should never happen as the completion
		// should only be called on a block the "pos" points to
		cb.logger.Printf("block type not found at %#v", pos)
		return nil, nil
	}

	prefix, prefixRng := prefixAtPos(cb.tBlock, pos)
	cb.logger.Printf("completing block: %#v, %#v", prefix, prefixRng)

	for name, attr := range b.Attributes() {
		if len(list.candidates) >= cb.maxCompletionCandidates() {
			list.isIncomplete = true
			break
		}
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if attr.IsComputedOnly() || attr.IsDeclared() {
			continue
		}
		list.candidates = append(list.candidates, &attributeCandidate{
			Name:        name,
			Attr:        attr,
			PrefixRange: prefixRng,
		})
	}

	for name, block := range b.BlockTypes() {
		if len(list.candidates) >= cb.maxCompletionCandidates() {
			list.isIncomplete = true
			break
		}
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if block.ReachedMaxItems() {
			continue
		}

		nbc := &nestedBlockCandidate{
			Name:      name,
			BlockType: block,
		}
		if prefixRng != nil {
			nbc.PrefixRange = prefixRng
		}
		list.candidates = append(list.candidates, nbc)
	}

	list.Sort()

	return list, nil
}

type candidateList struct {
	candidates   []CompletionCandidate
	isIncomplete bool
}

func (l *candidateList) Sort() {
	less := func(i, j int) bool {
		return l.candidates[i].Label() < l.candidates[j].Label()
	}
	sort.Slice(l.candidates, less)
}

func (l *candidateList) List() []CompletionCandidate {
	return l.candidates
}

func (l *candidateList) Len() int {
	return len(l.candidates)
}

func (l *candidateList) IsComplete() bool {
	return !l.isIncomplete
}

type labelCandidate struct {
	label         string
	detail        string
	documentation MarkupContent
	prefixRng     *hcl.Range
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

func (c *labelCandidate) Snippet() TextEdit {
	return c.PlainText()
}

func (c *labelCandidate) PlainText() TextEdit {
	return &textEdit{
		newText: c.label,
		rng:     c.prefixRng,
	}
}

type attributeCandidate struct {
	Name        string
	Attr        *Attribute
	PrefixRange *hcl.Range
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

func (c *attributeCandidate) Snippet() TextEdit {
	return &textEdit{
		newText: fmt.Sprintf("%s = %s", c.Name, snippetForAttrType(0, c.Attr.Schema().AttributeType)),
		rng:     c.PrefixRange,
	}
}

func (c *attributeCandidate) PlainText() TextEdit {
	return &textEdit{
		newText: c.Name,
		rng:     c.PrefixRange,
	}
}

type nestedBlockCandidate struct {
	Name        string
	BlockType   *BlockType
	PrefixRange *hcl.Range
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

func (c *nestedBlockCandidate) Snippet() TextEdit {
	return &textEdit{
		newText: snippetForNestedBlock(c.Name),
		rng:     c.PrefixRange,
	}
}

func (c *nestedBlockCandidate) PlainText() TextEdit {
	return &textEdit{
		newText: c.Name,
		rng:     c.PrefixRange,
	}
}
