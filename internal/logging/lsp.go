// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package logging

import (
	"context"
	"log"
	"strings"

	"github.com/creachadair/jrpc2"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func NewLspLogger(lspLog *LspLogger) *log.Logger {
	return log.New(lspLog, "", log.Lshortfile)
}

type LspLogger struct {
	Context context.Context
}

func (l LspLogger) Write(p []byte) (int, error) {
	logMessage := string(p)
	// there appears to be an extra newline that's helpful for stderr
	// but not for outputchannel
	logMessage = strings.TrimSuffix(logMessage, "\n")
	
	// TODO handle error here
	jrpc2.ServerFromContext(l.Context).Notify(l.Context, "window/logMessage", &lsp.LogMessageParams{
		Type: lsp.Log,
		Message: logMessage,
	})
	
	return 0, nil
}
