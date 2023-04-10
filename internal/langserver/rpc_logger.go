// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package langserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/creachadair/jrpc2"
)

type rpcLogger struct {
	logger *log.Logger
}

func (rl *rpcLogger) LogRequest(ctx context.Context, req *jrpc2.Request) {
	idStr := ""
	if req.ID() != "" {
		idStr = fmt.Sprintf(" (ID %s)", req.ID())
	}
	reqType := "request"
	if req.IsNotification() {
		reqType = "notification"
	}

	var params json.RawMessage
	req.UnmarshalParams(&params)

	rl.logger.Printf("Incoming %s for %q%s: %s",
		reqType, req.Method(), idStr, params)
}

func (rl *rpcLogger) LogResponse(ctx context.Context, rsp *jrpc2.Response) {
	idStr := ""
	if rsp.ID() != "" {
		idStr = fmt.Sprintf(" (ID %s)", rsp.ID())
	}

	req := jrpc2.InboundRequest(ctx)
	if req.IsNotification() {
		idStr = " (notification)"
	}

	if rsp.Error() != nil {
		rl.logger.Printf("Error for %q%s: %s", req.Method(), idStr, rsp.Error())
		return
	}
	var body json.RawMessage
	rsp.UnmarshalResult(&body)
	rl.logger.Printf("Response to %q%s: %s", req.Method(), idStr, body)
}
