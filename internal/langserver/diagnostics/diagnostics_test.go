package diagnostics

import (
	"context"
	"io/ioutil"
	"log"
	"testing"

	"github.com/hashicorp/hcl/v2"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

var discardLogger = log.New(ioutil.Discard, "", 0)

func TestDiags_Closes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	n := NewNotifier(ctx, discardLogger)

	cancel()
	n.PublishHCLDiags(context.Background(), "", map[string]hcl.Diagnostics{
		"test": {
			{
				Severity: hcl.DiagError,
			},
		},
	}, "test")

	if _, open := <-n.diags; open {
		t.Fatal("channel should be closed")
	}
}

func TestPublish_DoesNotSendAfterClose(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Fatal(err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	n := NewNotifier(ctx, discardLogger)

	cancel()
	n.PublishHCLDiags(context.Background(), "", map[string]hcl.Diagnostics{
		"test": {
			{
				Severity: hcl.DiagError,
			},
		},
	}, "test")
}

func TestMergeDiags_CachesMultipleSourcesPerURI(t *testing.T) {
	uri := "test.tf"

	n := NewNotifier(context.Background(), discardLogger)

	all := n.mergeDiags(uri, "source1", []lsp.Diagnostic{
		{
			Severity: lsp.SeverityError,
			Message:  "diag1",
		},
	})
	if len(all) != 1 {
		t.Fatalf("returns diags is incorrect length: expected %d, got %d", 1, len(all))
	}

	all = n.mergeDiags(uri, "source2", []lsp.Diagnostic{
		{
			Severity: lsp.SeverityError,
			Message:  "diag2",
		},
	})
	if len(all) != 2 {
		t.Fatalf("returns diags is incorrect length: expected %d, got %d", 2, len(all))
	}
}

func TestMergeDiags_OverwritesSource_EvenWithEmptySlice(t *testing.T) {
	uri := "test.tf"

	n := NewNotifier(context.Background(), discardLogger)

	all := n.mergeDiags(uri, "source1", []lsp.Diagnostic{
		{
			Severity: lsp.SeverityError,
			Message:  "diag1",
		},
	})
	if len(all) != 1 {
		t.Fatalf("returns diags is incorrect length: expected %d, got %d", 1, len(all))
	}

	all = n.mergeDiags(uri, "source1", []lsp.Diagnostic{
		{
			Severity: lsp.SeverityError,
			Message:  "diagOverwritten",
		},
	})
	if len(all) != 1 {
		t.Fatalf("returns diags is incorrect length: expected %d, got %d", 1, len(all))
	}
	if all[0].Message != "diagOverwritten" {
		t.Fatalf("diag has incorrect message: expected %s, got %s", "diagOverwritten", all[0].Message)
	}

	all = n.mergeDiags(uri, "source2", []lsp.Diagnostic{
		{
			Severity: lsp.SeverityError,
			Message:  "diag2",
		},
	})
	if len(all) != 2 {
		t.Fatalf("returns diags is incorrect length: expected %d, got %d", 2, len(all))
	}

	all = n.mergeDiags(uri, "source2", []lsp.Diagnostic{})
	if len(all) != 1 {
		t.Fatalf("returns diags is incorrect length: expected %d, got %d", 1, len(all))
	}
}
