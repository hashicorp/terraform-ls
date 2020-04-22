package source

import (
	"testing"
)

func TestMakeSourceLines_empty(t *testing.T) {
	lines := MakeSourceLines("/test.tf", []byte{})
	if len(lines) != 0 {
		t.Fatalf("Expected no lines from empty file, %d parsed:\n%#v",
			len(lines), lines)
	}
}

func TestMakeSourceLines_success(t *testing.T) {
	lines := MakeSourceLines("/test.tf", []byte("\n\n\n\n"))
	expectedLines := 4
	if len(lines) != expectedLines {
		t.Fatalf("Expected exactly %d lines, %d parsed",
			expectedLines, len(lines))
	}
}
