package exec

import (
	"errors"
	"testing"
	"time"
)

func TestExec_timeout(t *testing.T) {
	e := MockExecutor(&Mock{
		Args:          []string{"version"},
		SleepDuration: 100 * time.Millisecond,
		Stdout:        "Terraform v0.12.0\n",
	})
	e.timeout = 1 * time.Millisecond

	expectedErr := ExecTimeoutError([]string{"terraform", "version"}, e.timeout)

	_, err := e.Version()
	if err != nil {
		if errors.Is(err, expectedErr) {
			return
		}

		t.Fatalf("errors don't match.\nexpected: %#v\ngiven:    %#v\n",
			expectedErr, err)
	}

	t.Fatalf("expected timeout error: %#v", expectedErr)
}

func TestExec_Version(t *testing.T) {
	e := MockExecutor(&Mock{
		Args:     []string{"version"},
		Stdout:   "Terraform v0.12.0\n",
		ExitCode: 0,
	})
	v, err := e.Version()
	if err != nil {
		t.Fatal(err)
	}
	if v != "0.12.0" {
		t.Fatalf("output does not match: %#v", v)
	}
}

func TestExec_ProviderSchemas(t *testing.T) {
	e := MockExecutor(&Mock{
		Args:     []string{"providers", "schema", "-json"},
		Stdout:   `{"format_version": "0.1"}`,
		ExitCode: 0,
	})

	ps, err := e.ProviderSchemas()
	if err != nil {
		t.Fatal(err)
	}

	expectedVersion := "0.1"
	if ps.FormatVersion != expectedVersion {
		t.Fatalf("format version doesn't match.\nexpected: %q\ngiven: %q\n",
			expectedVersion, ps.FormatVersion)
	}
}
