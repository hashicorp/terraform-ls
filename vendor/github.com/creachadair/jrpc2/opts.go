package jrpc2

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/creachadair/jrpc2/metrics"
)

// ServerOptions control the behaviour of a server created by NewServer.
// A nil *ServerOptions provides sensible defaults.
type ServerOptions struct {
	// If not nil, send debug logs here.
	Logger *log.Logger

	// If not nil, the methods of this value are called to log each request
	// received and each response or error returned.
	RPCLog RPCLogger

	// Instructs the server to tolerate requests that do not include the
	// required "jsonrpc" version marker.
	AllowV1 bool

	// Instructs the server to allow server notifications, a non-standard
	// extension to the JSON-RPC protocol. If AllowPush is false, the Push
	// method of the server will report an error when called.
	AllowPush bool

	// Instructs the server to disable the built-in rpc.* handler methods.
	//
	// By default, a server reserves all rpc.* methods, even if the given
	// assigner maps them. When this option is true, rpc.* methods are passed
	// along to the given assigner.
	DisableBuiltin bool

	// Allows up to the specified number of goroutines to execute concurrently
	// in request handlers. A value less than 1 uses runtime.NumCPU().
	Concurrency int

	// If set, this function is called with the method name and encoded request
	// parameters received from the client, before they are delivered to the
	// handler. Its return value replaces the context and argument values. This
	// allows the server to decode context metadata sent by the client.
	// If unset, ctx and params are used as given.
	DecodeContext func(context.Context, string, json.RawMessage) (context.Context, json.RawMessage, error)

	// If set, this function is called with the context and the client request
	// to be delivered to the handler. If CheckRequest reports a non-nil error,
	// the request fails with that error without invoking the handler.
	CheckRequest func(ctx context.Context, req *Request) error

	// If set, use this value to record server metrics. All servers created
	// from the same options will share the same metrics collector.  If none is
	// set, an empty collector will be created for each new server.
	Metrics *metrics.M

	// If nonzero this value as the server start time; otherwise, use the
	// current time when Start is called.
	StartTime time.Time
}

func (s *ServerOptions) logger() logger {
	if s == nil || s.Logger == nil {
		return func(string, ...interface{}) {}
	}
	logger := s.Logger
	return func(msg string, args ...interface{}) { logger.Output(2, fmt.Sprintf(msg, args...)) }
}

func (s *ServerOptions) allowV1() bool      { return s != nil && s.AllowV1 }
func (s *ServerOptions) allowPush() bool    { return s != nil && s.AllowPush }
func (s *ServerOptions) allowBuiltin() bool { return s == nil || !s.DisableBuiltin }

func (s *ServerOptions) concurrency() int64 {
	if s == nil || s.Concurrency < 1 {
		return int64(runtime.NumCPU())
	}
	return int64(s.Concurrency)
}

func (s *ServerOptions) startTime() time.Time {
	if s == nil {
		return time.Time{}
	}
	return s.StartTime
}

type decoder = func(context.Context, string, json.RawMessage) (context.Context, json.RawMessage, error)

func (s *ServerOptions) decodeContext() (decoder, bool) {
	if s == nil || s.DecodeContext == nil {
		return func(ctx context.Context, method string, params json.RawMessage) (context.Context, json.RawMessage, error) {
			return ctx, params, nil
		}, false
	}
	return s.DecodeContext, true
}

type verifier = func(context.Context, *Request) error

func (s *ServerOptions) checkRequest() verifier {
	if s == nil || s.CheckRequest == nil {
		return func(context.Context, *Request) error { return nil }
	}
	return s.CheckRequest
}

func (s *ServerOptions) metrics() *metrics.M {
	if s == nil || s.Metrics == nil {
		return metrics.New()
	}
	return s.Metrics
}

func (s *ServerOptions) rpcLog() RPCLogger {
	if s == nil || s.RPCLog == nil {
		return nullRPCLogger{}
	}
	return s.RPCLog
}

// ClientOptions control the behaviour of a client created by NewClient.
// A nil *ClientOptions provides sensible defaults.
type ClientOptions struct {
	// If not nil, send debug logs here.
	Logger *log.Logger

	// Instructs the client to tolerate responses that do not include the
	// required "jsonrpc" version marker.
	AllowV1 bool

	// Instructs the client not to send rpc.cancel notifications to the server
	// when the context for an in-flight request terminates.
	DisableCancel bool

	// If set, this function is called with the context, method name, and
	// encoded request parameters before the request is sent to the server.
	// Its return value replaces the request parameters. This allows the client
	// to send context metadata along with the request. If unset, the parameters
	// are unchanged.
	EncodeContext func(context.Context, string, json.RawMessage) (json.RawMessage, error)

	// If set, this function is called if a notification is received from the
	// server. If unset, server notifications are logged and discarded.  At
	// most one invocation of the callback will be active at a time.
	// Server notifications are a non-standard extension of JSON-RPC.
	OnNotify func(*Request)
}

func (c *ClientOptions) logger() logger {
	if c == nil || c.Logger == nil {
		return func(string, ...interface{}) {}
	}
	logger := c.Logger
	return func(msg string, args ...interface{}) { logger.Output(2, fmt.Sprintf(msg, args...)) }
}

func (c *ClientOptions) allowV1() bool     { return c != nil && c.AllowV1 }
func (c *ClientOptions) allowCancel() bool { return c == nil || !c.DisableCancel }

type encoder = func(context.Context, string, json.RawMessage) (json.RawMessage, error)

func (c *ClientOptions) encodeContext() encoder {
	if c == nil || c.EncodeContext == nil {
		return func(_ context.Context, methods string, params json.RawMessage) (json.RawMessage, error) {
			return params, nil
		}
	}
	return c.EncodeContext
}

func (c *ClientOptions) handleNotification() func(*jresponse) bool {
	if c == nil || c.OnNotify == nil {
		return func(*jresponse) bool { return false }
	}
	h := c.OnNotify
	return func(req *jresponse) bool {
		if req.isServerRequest() {
			h(&Request{method: req.M, params: req.P})
			return true
		}
		return false
	}
}

// An RPCLogger receives callbacks from a server to record the receipt of
// requests and the delivery of responses. These callbacks are invoked
// synchronously with the processing of the request.
type RPCLogger interface {
	// Called for each request received prior to invoking its handler.
	LogRequest(ctx context.Context, req *Request)

	// Called for each response produced by a handler, immediately prior to
	// sending it back to the client. The inbound request can be recovered from
	// the context using jrpc2.InboundRequest.
	LogResponse(ctx context.Context, rsp *Response)
}

type nullRPCLogger struct{}

func (nullRPCLogger) LogRequest(context.Context, *Request)   {}
func (nullRPCLogger) LogResponse(context.Context, *Response) {}
