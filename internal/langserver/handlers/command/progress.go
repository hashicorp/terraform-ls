package command

import (
	"context"

	"github.com/creachadair/jrpc2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func progressBegin(ctx context.Context, title string) error {
	token, ok := lsctx.ProgressToken(ctx)
	if !ok {
		return nil
	}

	return jrpc2.PushNotify(ctx, "$/progress", lsp.ProgressParams{
		Token: token,
		Value: lsp.WorkDoneProgressBegin{
			Kind:  "begin",
			Title: title,
		},
	})
}

func progressReport(ctx context.Context, message string) error {
	token, ok := lsctx.ProgressToken(ctx)
	if !ok {
		return nil
	}

	return jrpc2.PushNotify(ctx, "$/progress", lsp.ProgressParams{
		Token: token,
		Value: lsp.WorkDoneProgressReport{
			Kind:    "report",
			Message: message,
		},
	})
}

func progressEnd(ctx context.Context, message string) error {
	token, ok := lsctx.ProgressToken(ctx)
	if !ok {
		return nil
	}

	return jrpc2.PushNotify(ctx, "$/progress", lsp.ProgressParams{
		Token: token,
		Value: lsp.WorkDoneProgressEnd{
			Kind:    "end",
			Message: message,
		},
	})
}
