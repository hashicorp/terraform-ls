package lang

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/hashicorp/go-version"
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	ihcl "github.com/hashicorp/terraform-ls/internal/hcl"
	"github.com/hashicorp/terraform-ls/internal/terraform/addrs"
	"github.com/hashicorp/terraform-ls/internal/terraform/errors"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
)

// defaultMaxCompletionCandidates is the maximum number of candidates
// to send in one completion response (with isIncomplete = true)
var defaultMaxCompletionCandidates = 100

// 0.12.0 first introduced HCL2 which provides
// more convenient/cleaner parsing
//
// We set no upper bound for now as there is only schema-related
// logic and schema format itself is version-checked elsewhere
//
// We may become more pessimistic as the parser begins to support
// language features which may differ between versions
// (e.g. meta-parameters)
const parserVersionConstraint = ">= 0.12.0"

type parser struct {
	logger *log.Logger

	maxCandidates int
	schemaReader  schema.Reader
	providerRefs  addrs.ProviderReferences
}

func parserSupportsTerraform(v string) error {
	rawVer, err := version.NewVersion(v)
	if err != nil {
		return err
	}

	// Assume that alpha/beta/rc prereleases have the same compatibility
	segments := rawVer.Segments64()
	segmentsOnly := fmt.Sprintf("%d.%d.%d", segments[0], segments[1], segments[2])
	tfVersion, err := version.NewVersion(segmentsOnly)
	if err != nil {
		return fmt.Errorf("failed to parse stripped version: %w", err)
	}

	c, err := version.NewConstraint(parserVersionConstraint)
	if err != nil {
		return err
	}

	if !c.Check(tfVersion) {
		return &errors.UnsupportedTerraformVersion{
			Component:   "parser",
			Version:     v,
			Constraints: c,
		}
	}

	return nil
}

// FindCompatibleParser finds a parser that is compatible with
// given Terraform version, so that it parses config accuretly
func FindCompatibleParser(v string) (Parser, error) {
	err := parserSupportsTerraform(v)
	if err != nil {
		return nil, err
	}

	return newParser(), nil
}

func newParser() *parser {
	return &parser{
		logger:        log.New(ioutil.Discard, "", 0),
		maxCandidates: defaultMaxCompletionCandidates,
		providerRefs:  make(addrs.ProviderReferences, 0),
	}
}

func (p *parser) SetLogger(logger *log.Logger) {
	p.logger = logger
}

func (p *parser) SetSchemaReader(sr schema.Reader) {
	// TODO: mutex
	p.schemaReader = sr
}

func (p *parser) SetProviderReferences(refs addrs.ProviderReferences) {
	// TODO: mutex
	p.providerRefs = refs
}

func (p *parser) blockTypes() map[string]configBlockFactory {
	return map[string]configBlockFactory{
		"provider": &providerBlockFactory{
			logger:       p.logger,
			schemaReader: p.schemaReader,
			providerRefs: p.providerRefs,
		},
		"resource": &resourceBlockFactory{
			logger:       p.logger,
			schemaReader: p.schemaReader,
			providerRefs: p.providerRefs,
		},
		"data": &datasourceBlockFactory{
			logger:       p.logger,
			schemaReader: p.schemaReader,
			providerRefs: p.providerRefs,
		},
	}
}

func (p *parser) CompletionCandidatesAtPos(file ihcl.TokenizedFile, pos hcl.Pos) (CompletionCandidates, error) {
	if !file.PosInBlock(pos) {
		return p.BlockTypeCandidates(file, pos), nil
	}

	block, err := file.BlockAtPosition(pos)
	if err != nil {
		return nil, fmt.Errorf("finding HCL block failed: %#v", err)
	}

	cfgBlock, err := p.ParseBlockFromTokens(block)
	if err != nil {
		return nil, fmt.Errorf("finding config block failed: %w", err)
	}

	return cfgBlock.CompletionCandidatesAtPos(pos)
}

func (p *parser) BlockTypeCandidates(file ihcl.TokenizedFile, pos hcl.Pos) CompletionCandidates {
	bTypes := p.blockTypes()

	list := &candidateList{
		candidates: make([]CompletionCandidate, 0),
	}

	prefix, prefixRng := prefixAtPos(file, pos)
	for name, t := range bTypes {
		if len(list.candidates) >= p.maxCandidates {
			list.isIncomplete = true
			break
		}
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		list.candidates = append(list.candidates, &completableBlockType{
			TypeName:      name,
			LabelSchema:   t.LabelSchema(),
			documentation: t.Documentation(),
			prefix:        prefix,
			prefixRng:     prefixRng,
		})
	}

	return list
}

func (p *parser) Blocks(file ihcl.TokenizedFile) ([]ConfigBlock, error) {
	blocks, err := file.Blocks()
	if err != nil {
		return nil, err
	}

	var cfgBlocks []ConfigBlock

	for _, block := range blocks {
		cfgBlock, err := p.ParseBlockFromTokens(block)
		if err != nil {
			// do not error out if parsing failed, continue to parse
			// blocks that are supported
			p.logger.Printf("parsing config block failed: %s", err)
			continue
		}
		cfgBlocks = append(cfgBlocks, cfgBlock)
	}

	return cfgBlocks, nil
}

type completableBlockType struct {
	TypeName      string
	LabelSchema   LabelSchema
	documentation MarkupContent
	prefix        string
	prefixRng     *hcl.Range
}

func (bt *completableBlockType) Label() string {
	return bt.TypeName
}

func (bt *completableBlockType) PlainText() TextEdit {
	return &textEdit{
		newText: bt.TypeName,
		rng:     bt.prefixRng,
	}
}

func (bt *completableBlockType) Snippet() TextEdit {
	return &textEdit{
		newText: snippetForBlock(bt.TypeName, bt.LabelSchema),
		rng:     bt.prefixRng,
	}
}

func (bt *completableBlockType) Detail() string {
	return ""
}

func (bt *completableBlockType) Documentation() MarkupContent {
	return bt.documentation
}

func (p *parser) ParseBlockFromTokens(tBlock ihcl.TokenizedBlock) (ConfigBlock, error) {
	// It is probably excessive to be parsing the whole block just for type
	// but there is no avoiding it without refactoring the upstream HCL parser
	// and it should not hurt the performance too much
	//
	// We ignore diags as we assume incomplete (invalid) configuration
	block, _ := hclsyntax.ParseBlockFromTokens(tBlock.Tokens())

	p.logger.Printf("Parsed block type: %q", block.Type)

	f, ok := p.blockTypes()[block.Type]
	if !ok {
		return nil, &unknownBlockTypeErr{block.Type}
	}

	cfgBlock, err := f.New(tBlock)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", block.Type, err)
	}

	return cfgBlock, nil
}

func discardLog() *log.Logger {
	return log.New(ioutil.Discard, "", 0)
}
