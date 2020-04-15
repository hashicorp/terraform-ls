package source

import (
	"bufio"

	"github.com/hashicorp/hcl/v2"
)

type sourceLine struct {
	content []byte
	rng     hcl.Range
}

// IsAllASCII returns true if the receiver is provably all ASCII, which allows
// for some fast paths where we can treat columns and bytes as equivalent.
func (l sourceLine) IsAllASCII() bool {
	// If we have the same number of columns as bytes then our content is
	// all ASCII, since it clearly contains no multi-byte grapheme clusters.
	bytes := l.rng.End.Byte - l.rng.Start.Byte
	columns := l.rng.End.Column - l.rng.Start.Column
	return bytes == columns
}

func (l sourceLine) Range() hcl.Range {
	return l.rng
}

func (l sourceLine) Bytes() []byte {
	return l.content
}

func MakeSourceLines(filename string, s []byte) []Line {
	var ret []Line
	if len(s) == 0 {
		return ret
	}

	sc := hcl.NewRangeScanner(s, filename, bufio.ScanLines)
	for sc.Scan() {
		ret = append(ret, sourceLine{
			content: sc.Bytes(),
			rng:     sc.Range(),
		})
	}

	return ret
}
