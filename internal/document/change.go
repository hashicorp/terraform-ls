// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package document

import (
	"bytes"

	"github.com/hashicorp/terraform-ls/internal/source"
)

type Change interface {
	Text() string
	Range() *Range
}

type Changes []Change

func ApplyChanges(original []byte, changes Changes) ([]byte, error) {
	if len(changes) == 0 {
		return original, nil
	}

	var buf bytes.Buffer
	_, err := buf.Write(original)
	if err != nil {
		return nil, err
	}

	for _, ch := range changes {
		err := applyDocumentChange(&buf, ch)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func applyDocumentChange(buf *bytes.Buffer, change Change) error {
	// if the range is nil, we assume it is full content change
	if change.Range() == nil {
		buf.Reset()
		_, err := buf.WriteString(change.Text())
		return err
	}

	lines := source.MakeSourceLines("", buf.Bytes())

	startByte, err := ByteOffsetForPos(lines, change.Range().Start)
	if err != nil {
		return err
	}
	endByte, err := ByteOffsetForPos(lines, change.Range().End)
	if err != nil {
		return err
	}

	diff := endByte - startByte
	if diff > 0 {
		buf.Grow(diff)
	}

	beforeChange := make([]byte, startByte, startByte)
	copy(beforeChange, buf.Bytes())
	afterBytes := buf.Bytes()[endByte:]
	afterChange := make([]byte, len(afterBytes), len(afterBytes))
	copy(afterChange, afterBytes)

	buf.Reset()

	_, err = buf.Write(beforeChange)
	if err != nil {
		return err
	}
	_, err = buf.WriteString(change.Text())
	if err != nil {
		return err
	}
	_, err = buf.Write(afterChange)
	if err != nil {
		return err
	}

	return nil
}
