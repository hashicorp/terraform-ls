package diagnostics

import (
	"context"
	"io/ioutil"
	"log"
	"testing"

	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

var discardLogger = log.New(ioutil.Discard, "", 0)

func TestDiags_Closes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	n := NewNotifier(ctx, discardLogger)

	cancel()
	n.Publish(context.Background(), "", map[string][]lsp.Diagnostic{
		"test": {
			{
				Severity: lsp.SeverityError,
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
	n.Publish(context.Background(), "", map[string][]lsp.Diagnostic{
		"test": {
			{
				Severity: lsp.SeverityError,
			},
		},
	}, "test")
}
