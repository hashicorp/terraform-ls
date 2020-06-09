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
	list.SortAndSetSortText()

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

	if prefix == "" {
		allRequiredFieldCandidate := computeAllRequiredFieldCandidate(prefixRng, b)
		if !allRequiredFieldCandidate.Empty() {
			list.candidates = append(list.candidates, allRequiredFieldCandidate)
		}
	}

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
		list.candidates = append(list.candidates, &nestedBlockCandidate{
			Name:        name,
			BlockType:   block,
			PrefixRange: prefixRng,
		})
	}
	list.SortAndSetSortText()

	return list, nil
}

func computeAllRequiredFieldCandidate(prefixRng *hcl.Range, b Block) *AllRequiredFieldCandidate {
	allRequiredFieldCandidate := &AllRequiredFieldCandidate{
		PrefixRange: prefixRng,
	}
	for name, attr := range b.Attributes() {
		if attr.Schema().Required && !attr.IsDeclared() {
			allRequiredFieldCandidate.AttrCandidates = append(allRequiredFieldCandidate.AttrCandidates, &attributeCandidate{
				Name:        name,
				Attr:        attr,
				PrefixRange: prefixRng,
			})
		}
	}

	for name, block := range b.BlockTypes() {
		nbc := &nestedBlockCandidate{
			Name:        name,
			BlockType:   block,
			PrefixRange: prefixRng,
		}
		for i := 1; i <= block.MissedMinItems(); i++ {
			allRequiredFieldCandidate.NestedBlockCandidates = append(allRequiredFieldCandidate.NestedBlockCandidates, nbc)
		}
	}
	return allRequiredFieldCandidate
}

type candidateList struct {
	candidates   []CompletionCandidate
	isIncomplete bool
}

func (l *candidateList) SortAndSetSortText() {
	less := func(i, j int) bool {
		sortText1 := candidateSortTextPolicy(l.candidates[i])
		sortText2 := candidateSortTextPolicy(l.candidates[j])
		return sortText1 < sortText2 || (sortText1 == sortText2 && l.candidates[i].Label() < l.candidates[j].Label())
	}
	sort.Slice(l.candidates, less)
	for i, can := range l.candidates {
		setCandidateSortText(can, i+1)
	}
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
	sortText      string
}

func (c *labelCandidate) SortText() string {
	return c.sortText
}

func (c *labelCandidate) Label() string {
	return c.label
}

func (c *labelCandidate) CompletionItemKind() int {
	return 5 //lsp.CIKField
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
	sortText    string
}

func (c *attributeCandidate) SortText() string {
	return c.sortText
}

func (c *attributeCandidate) CompletionItemKind() int {
	return 5 // lsp.CIKField
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
		newText: fmt.Sprintf("%s = %s", c.Name, snippetForAttrType(c.Attr.Schema().AttributeType)),
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
	sortText    string
}

func (c *nestedBlockCandidate) SortText() string {
	return c.sortText
}

func (c *nestedBlockCandidate) Label() string {
	return c.Name
}

func (c *nestedBlockCandidate) CompletionItemKind() int {
	return 5 // lsp.CIKField
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

type AllRequiredFieldCandidate struct {
	AttrCandidates        []*attributeCandidate
	NestedBlockCandidates []*nestedBlockCandidate
	PrefixRange           *hcl.Range
	sortText              string
}

func (c *AllRequiredFieldCandidate) SortText() string {
	return c.sortText
}

func (c *AllRequiredFieldCandidate) Label() string {
	return "Fill Required Fields..."
}

func (c *AllRequiredFieldCandidate) CompletionItemKind() int {
	return 4 // lsp.CIKConstructor
}

func (c *AllRequiredFieldCandidate) Detail() string {
	return ""
}

func (c *AllRequiredFieldCandidate) Empty() bool {
	return len(c.AttrCandidates)+len(c.NestedBlockCandidates) == 0
}

func (c *AllRequiredFieldCandidate) Documentation() MarkupContent {
	text := c.PlainText().NewText()
	text = strings.Join(strings.Split(text, "\n"), "\n\t")
	return PlainText(fmt.Sprintf("Auto-generated object literal (required fields)\n{\n\t%s\n}", text))
}

func (c *AllRequiredFieldCandidate) Snippet() TextEdit {
	var content []string
	placeHolder := 1
	for _, attr := range c.AttrCandidates {
		text, nextPlaceHolder := snippetForAttrTypeWithPlaceholder(placeHolder, attr.Attr.Schema().AttributeType)
		content = append(content, fmt.Sprintf("%s = %s", attr.Name, text))
		placeHolder = nextPlaceHolder
	}
	for _, nestedBlock := range c.NestedBlockCandidates {
		content = append(content, snippetForNestedBlockWithPlaceholder(placeHolder, nestedBlock.Name)+"\n")
		placeHolder++
	}
	return &textEdit{
		newText: strings.Join(content, "\n"),
		rng:     c.PrefixRange,
	}
}

func (c *AllRequiredFieldCandidate) PlainText() TextEdit {
	var content []string
	for _, attr := range c.AttrCandidates {
		content = append(content, fmt.Sprintf("%s = %s", attr.Name, plainTextForAttrType(attr.Attr.Schema().AttributeType)))
	}
	for _, nestedBlock := range c.NestedBlockCandidates {
		content = append(content, "\n"+plainTextForNestedBlock(nestedBlock.Name))
	}
	return &textEdit{
		newText: strings.Join(content, "\n"),
		rng:     c.PrefixRange,
	}
}
