package diagnostics

import (
	"context"
	"io/ioutil"
	"log"
	"testing"

	"github.com/hashicorp/hcl/v2"
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
