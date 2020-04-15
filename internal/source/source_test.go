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

func TestSourceLine_isAllASCII(t *testing.T) {
	lines := MakeSourceLines("/test.tf", []byte(`plaintext
smiley 🙃 here`))
	if len(lines) != 2 {
		t.Fatal("Expected exactly 2 lines")
	}

	if !lines[0].IsAllASCII() {
		t.Fatalf("Expected first line (%q) to be reported as ASCII",
			string(lines[0].Bytes()))
	}
	if lines[1].IsAllASCII() {
		t.Fatalf("Expected second line (%q) NOT to be reported as ASCII",
			string(lines[1].Bytes()))
	}
}
