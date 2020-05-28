package hcl

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	hcllib "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
)

type File interface {
	BlockTokensAtPosition(filesystem.FilePosition) (hclsyntax.Tokens, hcllib.Pos, error)
}

type file struct {
	filename string
	content  []byte
	pf       *parsedFile
}

type parsedFile struct {
	Body   hcllib.Body
	Tokens hclsyntax.Tokens
}

func NewFile(f filesystem.File) File {
	return &file{
		filename: f.Filename(),
		content:  []byte(f.Text()),
	}
}

func (f *file) parse() (*parsedFile, error) {
	if f.pf != nil {
		return f.pf, nil
	}

	var parseDiags hcllib.Diagnostics

	tokens, diags := hclsyntax.LexConfig(f.content, f.filename, hcllib.InitialPos)
	if diags.HasErrors() {
		parseDiags = append(parseDiags, diags...)
	}

	body, diags := hclsyntax.ParseBodyFromTokens(tokens, hclsyntax.TokenEOF)
	if diags.HasErrors() {
		parseDiags = append(parseDiags, diags...)
	}

	f.pf = &parsedFile{
		Tokens: tokens,
		Body:   body,
	}

	if parseDiags.HasErrors() {
		return f.pf, parseDiags
	}
	return f.pf, nil
}

func (f *file) BlockTokensAtPosition(filePos filesystem.FilePosition) (hclsyntax.Tokens, hcllib.Pos, error) {
	pos := filePos.Position()

	b, err := f.blockAtPosition(pos)
	if err != nil {
		return hclsyntax.Tokens{}, pos, err
	}

	return b, pos, nil
}

func (f *file) blockAtPosition(pos hcllib.Pos) (hclsyntax.Tokens, error) {
	pf, _ := f.parse()

	body, ok := pf.Body.(*hclsyntax.Body)
	if !ok {
		return hclsyntax.Tokens{}, fmt.Errorf("unexpected body type (%T)", body)
	}
	if body.SrcRange.Empty() && pos != hcllib.InitialPos {
		return hclsyntax.Tokens{}, &InvalidHclPosErr{pos, body.SrcRange}
	}
	if !body.SrcRange.Empty() {
		if posIsEqual(body.SrcRange.End, pos) {
			return hclsyntax.Tokens{}, &NoBlockFoundErr{pos}
		}
		if !body.SrcRange.ContainsPos(pos) {
			return hclsyntax.Tokens{}, &InvalidHclPosErr{pos, body.SrcRange}
		}
	}

	for _, block := range body.Blocks {
		wholeRange := hcllib.RangeBetween(block.TypeRange, block.CloseBraceRange)
		if wholeRange.ContainsPos(pos) {
			return definitionTokens(tokensInRange(f.pf.Tokens, block.Range())), nil
		}
	}

	return nil, &NoBlockFoundErr{pos}
}

func tokensInRange(tokens hclsyntax.Tokens, rng hcllib.Range) hclsyntax.Tokens {
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
		if lastToken.Type != hclsyntax.TokenEOF &&
			lastToken.Type != hclsyntax.TokenNewline {
			tRng := lastToken.Range

			// if not we attach a newline
			trailingNewLine := hclsyntax.Token{
				Type:  hclsyntax.TokenNewline,
				Bytes: []byte("\n"),
				Range: hcl.Range{
					Filename: tRng.Filename,
					Start:    tRng.End,
					End: hcl.Pos{
						Byte:   tRng.End.Byte + 1,
						Column: 1,
						Line:   tRng.End.Line + 1,
					},
				},
			}

			tokens = append(tokens, trailingNewLine)
		}
	}
	return tokens
}

func posIsEqual(a, b hcllib.Pos) bool {
	return a.Byte == b.Byte &&
		a.Column == b.Column &&
		a.Line == b.Line
}
