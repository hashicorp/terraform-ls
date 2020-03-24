package jrpc2

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/creachadair/jrpc2/metrics"
)

// ServerMetrics returns the server metrics collector associated with the given
// context, or nil if ctx does not have a collector attached.  The context
// passed to a handler by *jrpc2.Server will include this value.
func ServerMetrics(ctx context.Context) *metrics.M {
	return ctx.Value(serverKey{}).(*Server).metrics
}

// InboundRequest returns the inbound request associated with the given
// context, or nil if ctx does not have an inbound request. The context passed
// to the handler by *jrpc2.Server will include this value.
//
// This is mainly useful to wrapped server methods that do not have the request
// as an explicit parameter; for direct implementations of Handler.Handle the
// request value returned by InboundRequest will be the same value as was
// passed explicitly.
func InboundRequest(ctx context.Context) *Request {
	if v := ctx.Value(inboundRequestKey{}); v != nil {
		return v.(*Request)
	}
	return nil
}

type inboundRequestKey struct{}

// ServerPush posts a server notification to the client. If ctx does not
// contain a server notifier, this reports ErrNotifyUnsupported. The context
// passed to the handler by *jrpc2.Server will support notifications if the
// server was constructed with the AllowPush option set true.
func ServerPush(ctx context.Context, method string, params interface{}) error {
	s := ctx.Value(serverKey{}).(*Server)
	if !s.allowP {
		return ErrNotifyUnsupported
	}
	return s.Push(ctx, method, params)
}

// CancelRequest requests the cancellation of the pending or in-flight request
// with the specified ID.  If no request exists with that ID, this is a no-op
// without error.
func CancelRequest(ctx context.Context, id string) {
	s := ctx.Value(serverKey{}).(*Server)
	s.cancelRequests(ctx, []json.RawMessage{json.RawMessage(id)})
}

type serverKey struct{}

// ErrNotifyUnsupported is returned by ServerPush if server notifications are
// not enabled in the specified context.
var ErrNotifyUnsupported = errors.New("server notifications are not enabled")
