package hcl

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
)

type file struct {
	filename string
	content  []byte
	pf       *parsedFile
}

type parsedFile struct {
	Body   hcl.Body
	Tokens hclsyntax.Tokens
}

type parsedBlock struct {
	tokens hclsyntax.Tokens
}

func (pb *parsedBlock) Tokens() hclsyntax.Tokens {
	return pb.tokens
}

func (pb *parsedBlock) TokenAtPosition(pos hcl.Pos) (hclsyntax.Token, error) {
	for _, t := range pb.tokens {
		if rangeContainsOffset(t.Range, pos.Byte) {
			return t, nil
		}
	}

	return hclsyntax.Token{}, &NoTokenFoundErr{pos}
}

func NewFile(dh filesystem.DocumentHandler, content []byte) TokenizedFile {
	return &file{
		filename: dh.Filename(),
		content:  content,
	}
}

func NewTestFile(b []byte) TokenizedFile {
	return &file{
		filename: "/test.tf",
		content:  b,
	}
}

func NewTestBlock(b []byte) (TokenizedBlock, error) {
	f := NewTestFile(b)
	return f.BlockAtPosition(hcl.InitialPos)
}

func (f *file) parse() (*parsedFile, error) {
	if f.pf != nil {
		return f.pf, nil
	}

	tokens, diags := hclsyntax.LexConfig(f.content, f.filename, hcl.InitialPos)
	if diags.HasErrors() {
		// The hclsyntax parser assumes all tokens are valid
		// so we return early here
		// TODO: Avoid ignoring TokenQuotedNewline to provide completion in unclosed string
		return nil, diags
	}

	body, _ := hclsyntax.ParseBodyFromTokens(tokens, hclsyntax.TokenEOF)

	f.pf = &parsedFile{
		Tokens: tokens,
		Body:   body,
	}

	return f.pf, nil
}

func (f *file) PosInBlock(pos hcl.Pos) bool {
	_, err := f.BlockAtPosition(pos)
	if IsNoBlockFoundErr(err) {
		return false
	}

	return true
}

func (f *file) BlockAtPosition(pos hcl.Pos) (TokenizedBlock, error) {
	pf, err := f.parse()
	if err != nil {
		return nil, err
	}

	body, ok := pf.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unexpected body type (%T)", body)
	}
	if body.SrcRange.Empty() && pos != hcl.InitialPos {
		return nil, &InvalidHclPosErr{pos, body.SrcRange}
	}
	if !body.SrcRange.Empty() {
		if posIsEqual(body.SrcRange.End, pos) {
			return nil, &NoBlockFoundErr{pos}
		}
		if !body.SrcRange.ContainsPos(pos) {
			return nil, &InvalidHclPosErr{pos, body.SrcRange}
		}
	}

	for _, block := range body.Blocks {
		if block.Range().ContainsPos(pos) {
			dt := definitionTokens(tokensInRange(pf.Tokens, block.Range()))
			return &parsedBlock{dt}, nil
		}
	}

	return nil, &NoBlockFoundErr{pos}
}

func (f *file) Blocks() ([]TokenizedBlock, error) {
	var blocks []TokenizedBlock

	pf, err := f.parse()
	if err != nil {
		return blocks, err
	}

	body, ok := pf.Body.(*hclsyntax.Body)
	if !ok {
		return blocks, fmt.Errorf("unexpected body type (%T)", body)
	}

	for _, block := range body.Blocks {
		dt := definitionTokens(tokensInRange(pf.Tokens, block.Range()))
		blocks = append(blocks, &parsedBlock{dt})
	}

	return blocks, nil
}

func (f *file) TokenAtPosition(pos hcl.Pos) (hclsyntax.Token, error) {
	pf, _ := f.parse()

	for _, t := range pf.Tokens {
		if rangeContainsOffset(t.Range, pos.Byte) {
			return t, nil
		}
	}

	return hclsyntax.Token{}, &NoTokenFoundErr{pos}
}

func tokensInRange(tokens hclsyntax.Tokens, rng hcl.Range) hclsyntax.Tokens {
	var ts hclsyntax.Tokens

	for _, t := range tokens {
		if rangeContainsRange(rng, t.Range) {
			ts = append(ts, t)
		}
	}

	return ts
}

func rangeContainsRange(a, b hcl.Range) bool {
	switch {
	case a.Filename != b.Filename:
		// If the ranges are in different files then they can't possibly contain each other
		return false
	case a.Empty() || b.Empty():
		// Empty ranges can will never be contained in each other
		return false
	case rangeContainsOffset(a, b.Start.Byte) && rangeContainsOffset(a, b.End.Byte):
		return true
	case rangeContainsOffset(b, a.Start.Byte) && rangeContainsOffset(b, a.End.Byte):
		return true
	default:
		return false
	}
}

// rangeContainsOffset is a reimplementation of hcl.Range.ContainsOffset
// which treats offset matching the end of a range as contained
func rangeContainsOffset(rng hcl.Range, offset int) bool {
	return offset >= rng.Start.Byte && offset <= rng.End.Byte
}

// definitionTokens turns any non-empty sequence of tokens into one that
// satisfies HCL's loose definition of a valid block or attribute
// as represented by tokens
func definitionTokens(tokens hclsyntax.Tokens) hclsyntax.Tokens {
	if len(tokens) > 0 {
		// Check if seqence has a terminating token
		lastToken := tokens[len(tokens)-1]
		if lastToken.Type != hclsyntax.TokenEOF {
			tRng := lastToken.Range

			// if not we attach EOF
			eofToken := hclsyntax.Token{
				Type:  hclsyntax.TokenEOF,
				Bytes: []byte{},
				Range: hcl.Range{
					Filename: tRng.Filename,
					Start:    tRng.End,
					End:      tRng.End,
				},
			}

			tokens = append(tokens, eofToken)
		}
	}
	return tokens
}

func posIsEqual(a, b hcl.Pos) bool {
	return a.Byte == b.Byte &&
		a.Column == b.Column &&
		a.Line == b.Line
}
