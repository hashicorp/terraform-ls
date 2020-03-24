package jrpc2

import (
	"container/list"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/creachadair/jrpc2/channel"
	"github.com/creachadair/jrpc2/code"
	"github.com/creachadair/jrpc2/metrics"
	"golang.org/x/sync/semaphore"
)

type logger = func(string, ...interface{})

// A Server is a JSON-RPC 2.0 server. The server receives requests and sends
// responses on a channel.Channel provided by the caller, and dispatches
// requests to user-defined Handlers.
type Server struct {
	wg      sync.WaitGroup      // ready when workers are done at shutdown time
	mux     Assigner            // associates method names with handlers
	sem     *semaphore.Weighted // bounds concurrent execution (default 1)
	allow1  bool                // allow v1 requests with no version marker
	allowP  bool                // allow server notifications to the client
	log     logger              // write debug logs here
	rpcLog  RPCLogger           // log RPC requests and responses here
	dectx   decoder             // decode context from request
	ckreq   verifier            // request checking hook
	expctx  bool                // whether to expect request context
	metrics *metrics.M          // metrics collected during execution
	start   time.Time           // when Start was called
	builtin bool                // whether built-in rpc.* methods are enabled

	mu *sync.Mutex // protects the fields below

	err  error           // error from a previous operation
	work *sync.Cond      // for signaling message availability
	inq  *list.List      // inbound requests awaiting processing
	ch   channel.Channel // the channel to the client

	// For each request ID currently in-flight, this map carries a cancel
	// function attached to the context that was sent to the handler.
	used map[string]context.CancelFunc
}

// NewServer returns a new unstarted server that will dispatch incoming
// JSON-RPC requests according to mux. To start serving, call Start.
//
// N.B. It is only safe to modify mux after the server has been started if mux
// itself is safe for concurrent use by multiple goroutines.
//
// This function will panic if mux == nil.
func NewServer(mux Assigner, opts *ServerOptions) *Server {
	if mux == nil {
		panic("nil assigner")
	}
	dc, exp := opts.decodeContext()
	s := &Server{
		mux:     mux,
		sem:     semaphore.NewWeighted(opts.concurrency()),
		allow1:  opts.allowV1(),
		allowP:  opts.allowPush(),
		log:     opts.logger(),
		rpcLog:  opts.rpcLog(),
		dectx:   dc,
		ckreq:   opts.checkRequest(),
		expctx:  exp,
		mu:      new(sync.Mutex),
		metrics: opts.metrics(),
		start:   opts.startTime(),
		builtin: opts.allowBuiltin(),
		inq:     list.New(),
		used:    make(map[string]context.CancelFunc),
	}
	s.work = sync.NewCond(s.mu)
	return s
}

// Start enables processing of requests from c. This function will panic if the
// server is already running.
func (s *Server) Start(c channel.Channel) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ch != nil {
		panic("server is already running")
	}

	// Set up the queues and condition variable used by the workers.
	s.ch = c
	if s.start.IsZero() {
		s.start = time.Now().In(time.UTC)
	}

	// Reset all the I/O structures and start up the workers.
	s.err = nil

	// s.wg waits for the maintenance goroutines for receiving input and
	// processing the request queue. In addition, each request in flight adds a
	// goroutine to s.wg. At server shutdown, s.wg completes when the
	// maintenance goroutines and all pending requests are finished.
	s.wg.Add(2)

	// Accept requests from the client and enqueue them for processing.
	go func() { defer s.wg.Done(); s.read(c) }()

	// Remove requests from the queue and dispatch them to handlers.
	go func() { defer s.wg.Done(); s.serve() }()

	return s
}

// serve processes requests from the queue and dispatches them to handlers.
// The responses are written back by the handler goroutines.
//
// The flow of an inbound request is:
//
//   serve             -- main serving loop
//   * nextRequest     -- process the next request batch
//     * dispatch
//       * assign      -- assign handlers to requests
//       | ...
//       |
//       * invoke      -- invoke handlers
//       | \ handler   -- handle an individual request
//       |   ...
//       * deliver     -- send responses to the client
//
func (s *Server) serve() {
	for {
		next, err := s.nextRequest()
		if err != nil {
			s.log("Reading next request: %v", err)
			return
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			next()
		}()
	}
}

// nextRequest blocks until a request batch is available and returns a function
// that dispatches it to the appropriate handlers. The result is only an error
// if the connection failed; errors reported by the handler are reported to the
// caller and not returned here.
//
// The caller must invoke the returned function to complete the request.
func (s *Server) nextRequest() (func() error, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for s.ch != nil && s.inq.Len() == 0 {
		s.work.Wait()
	}
	if s.ch == nil && s.inq.Len() == 0 {
		return nil, s.err
	}
	ch := s.ch // capture

	next := s.inq.Remove(s.inq.Front()).(jrequests)
	s.log("Processing %d requests", len(next))

	// Construct a dispatcher to run the handlers outside the lock.
	return s.dispatch(next, ch), nil
}

// dispatch constructs a function that invokes each of the specified tasks.
// The caller must hold s.mu when calling dispatch, but the returned function
// should be executed outside the lock to wait for the handlers to return.
func (s *Server) dispatch(next jrequests, ch channel.Sender) func() error {
	// Resolve all the task handlers or record errors.
	start := time.Now()
	tasks := s.checkAndAssign(next)
	var wg sync.WaitGroup
	for _, t := range tasks {
		if t.err != nil {
			continue // nothing to do here; this task has already failed
		}
		t := t
		wg.Add(1)
		go func() {
			defer wg.Done()
			t.val, t.err = s.invoke(t.ctx, t.m, t.hreq)
		}()
	}

	// Wait for all the handlers to return, then deliver any responses.
	return func() error {
		wg.Wait()
		return s.deliver(tasks.responses(s.rpcLog), ch, time.Since(start))
	}
}

// deliver cleans up completed responses and arranges their replies (if any) to
// be sent back to the client.
func (s *Server) deliver(rsps jresponses, ch channel.Sender, elapsed time.Duration) error {
	if len(rsps) == 0 {
		return nil
	}
	s.log("Completed %d requests [%v elapsed]", len(rsps), elapsed)
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure all the inflight requests get their contexts cancelled.
	for _, rsp := range rsps {
		s.cancel(string(rsp.ID))
	}

	nw, err := encode(ch, rsps)
	s.metrics.CountAndSetMax("rpc.bytesWritten", int64(nw))
	return err
}

// checkAndAssign resolves all the task handlers for the given batch, or
// records errors for them as appropriate. The caller must hold s.mu.
func (s *Server) checkAndAssign(next jrequests) tasks {
	var ts tasks
	for _, req := range next {
		s.log("Checking request for %q: %s", req.M, string(req.P))
		fid := fixID(req.ID)
		t := &task{
			hreq:  &Request{id: fid, method: req.M, params: req.P},
			batch: req.batch,
		}
		if req.err != nil {
			t.err = req.err // deferred validation error
		} else if id := string(fid); id != "" && s.used[id] != nil {
			t.err = Errorf(code.InvalidRequest, "duplicate request id %q", id)
		} else if !s.versionOK(req.V) {
			t.err = ErrInvalidVersion
		} else if req.M == "" {
			t.err = Errorf(code.InvalidRequest, "empty method name")
		} else if s.setContext(t, id) {
			t.m = s.assign(t.ctx, req.M)
			if t.m == nil {
				t.err = Errorf(code.MethodNotFound, "no such method %q", req.M)
			}
		}

		if t.err != nil {
			s.log("Task error: %v", t.err)
			s.metrics.Count("rpc.errors", 1)
		}
		ts = append(ts, t)
	}
	return ts
}

// setContext constructs and attaches a request context to t, and reports
// whether this succeeded.
func (s *Server) setContext(t *task, id string) bool {
	base, params, err := s.dectx(context.Background(), t.hreq.method, t.hreq.params)
	t.hreq.params = params
	if err != nil {
		t.err = Errorf(code.InternalError, "invalid request context: %v", err)
		return false
	}

	// Check request.
	if err := s.ckreq(base, t.hreq); err != nil {
		t.err = err
		return false
	}

	t.ctx = context.WithValue(base, inboundRequestKey{}, t.hreq)

	// Store the cancellation for a request that needs a reply, so that we can
	// respond to rpc.cancel requests.
	if id != "" {
		ctx, cancel := context.WithCancel(t.ctx)
		s.used[id] = cancel
		t.ctx = ctx
	}
	return true
}

// invoke invokes the handler m for the specified request type, and marshals
// the return value into JSON if there is one.
func (s *Server) invoke(base context.Context, h Handler, req *Request) (json.RawMessage, error) {
	ctx := context.WithValue(base, serverKey{}, s)
	if err := s.sem.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer s.sem.Release(1)

	s.rpcLog.LogRequest(ctx, req)
	v, err := h.Handle(ctx, req)
	if err != nil {
		if req.IsNotification() {
			s.log("Discarding error from notification to %q: %v", req.Method(), err)
			return nil, nil // a notification
		}
		return nil, err // a call reporting an error
	}
	return json.Marshal(v)
}

// ServerInfo returns an atomic snapshot of the current server info for s.
func (s *Server) ServerInfo() *ServerInfo {
	info := &ServerInfo{
		Methods:     s.mux.Names(),
		UsesContext: s.expctx,
		StartTime:   s.start,
		Counter:     make(map[string]int64),
		MaxValue:    make(map[string]int64),
		Label:       make(map[string]string),
	}
	s.metrics.Snapshot(metrics.Snapshot{
		Counter:  info.Counter,
		MaxValue: info.MaxValue,
		Label:    info.Label,
	})
	return info
}

// Push posts a server-side notification to the client.  This is a non-standard
// extension of JSON-RPC, and may not be supported by all clients.  Unless s
// was constructed with the AllowPush option set true, this method will always
// report an error (ErrNotifyUnsupported) without sending anything.  If Push is
// called after the client connection is closed, it returns ErrConnClosed.
func (s *Server) Push(ctx context.Context, method string, params interface{}) error {
	if !s.allowP {
		return ErrNotifyUnsupported
	}
	var bits []byte
	if params != nil {
		v, err := json.Marshal(params)
		if err != nil {
			return err
		}
		bits = v
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ch == nil {
		return ErrConnClosed
	}
	s.log("Posting server notification %q %s", method, string(bits))
	nw, err := encode(s.ch, jresponses{{
		V: Version,
		M: method,
		P: bits,
	}})
	s.metrics.CountAndSetMax("rpc.bytesWritten", int64(nw))
	s.metrics.Count("rpc.notifications", 1)
	return err
}

// Stop shuts down the server. It is safe to call this method multiple times or
// from concurrent goroutines; it will only take effect once.
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop(errServerStopped)
}

// ServerStatus describes the status of a stopped server.
type ServerStatus struct {
	Err error // the error that caused the server to stop (nil on success)

	stopped bool // whether Stop was called
}

// Success reports whether the server exited without error.
func (s ServerStatus) Success() bool { return s.Err == nil }

// Stopped reports whether the server exited due to Stop being called.
func (s ServerStatus) Stopped() bool { return s.Err == nil && s.stopped }

// Closed reports whether the server exited due to a channel close.
func (s ServerStatus) Closed() bool { return s.Err == nil && !s.stopped }

// WaitStatus blocks until the server terminates, and returns the resulting
// status. After WaitStatus returns, whether or not there was an error, it is
// safe to call s.Start again to restart the server with a fresh channel.
func (s *Server) WaitStatus() ServerStatus {
	s.wg.Wait()
	// Sanity check.
	if s.inq.Len() != 0 {
		panic("s.inq is not empty at shutdown")
	}
	exitErr := s.err
	// Don't remark on a closed channel or EOF as a noteworthy failure.
	if s.err == io.EOF || channel.IsErrClosing(s.err) || s.err == errServerStopped {
		exitErr = nil
	}
	return ServerStatus{Err: exitErr, stopped: s.err == errServerStopped}
}

// Wait blocks until the server terminates and returns the resulting error.
// It is equivalent to s.WaitStatus().Err.
func (s *Server) Wait() error { return s.WaitStatus().Err }

// stop shuts down the connection and records err as its final state.  The
// caller must hold s.mu. If multiple callers invoke stop, only the first will
// successfully record its error status.
func (s *Server) stop(err error) {
	if s.ch == nil {
		return // nothing is running
	}
	s.log("Server signaled to stop with err=%v", err)
	s.ch.Close()

	// Remove any pending requests from the queue, but retain notifications.
	// The server will process pending notifications before giving up.
	for cur := s.inq.Front(); cur != nil; cur = cur.Next() {
		var keep jrequests
		for _, req := range cur.Value.(jrequests) {
			if req.ID == nil {
				keep = append(keep, req)
				s.log("Retaining notification %p", req)
			} else {
				s.cancel(string(req.ID))
			}
		}
		if len(keep) != 0 {
			s.inq.PushBack(keep)
		}
		s.inq.Remove(cur)
	}
	s.work.Broadcast()

	// Cancel any in-flight requests that made it out of the queue.
	for id, cancel := range s.used {
		cancel()
		delete(s.used, id)
	}

	// Sanity check.
	if len(s.used) != 0 {
		panic("s.used is not empty at shutdown")
	}

	s.err = err
	s.ch = nil
}

// read is the main receiver loop, decoding requests from the client and adding
// them to the queue. Decoding errors and message-format problems are handled
// and reported back to the client directly, so that any message that survives
// into the request queue is structurally valid.
func (s *Server) read(ch channel.Receiver) {
	for {
		// If the message is not sensible, report an error; otherwise enqueue it
		// for processing. Errors in individual requests are handled later.
		var in jrequests
		var derr error
		bits, err := ch.Recv()
		s.metrics.CountAndSetMax("rpc.bytesRead", int64(len(bits)))
		if err == nil || (err == io.EOF && len(bits) != 0) {
			err = nil
			derr = in.parseJSON(bits)
			s.metrics.Count("rpc.requests", int64(len(in)))
		}
		s.mu.Lock()
		if err != nil { // receive failure; shut down
			s.stop(err)
			s.mu.Unlock()
			return
		} else if derr != nil { // parse failure; report and continue
			s.pushError(derr)
		} else if len(in) == 0 {
			s.pushError(Errorf(code.InvalidRequest, "empty request batch"))
		} else {
			s.log("Received %d new requests", len(in))
			s.inq.PushBack(in)
			s.work.Broadcast()
		}
		s.mu.Unlock()
	}
}

// ServerInfo is the concrete type of responses from the rpc.serverInfo method.
type ServerInfo struct {
	// The list of method names exported by this server.
	Methods []string `json:"methods,omitempty"`

	// Whether this server understands context wrappers.
	UsesContext bool `json:"usesContext"`

	// Metric values defined by the evaluation of methods.
	Counter  map[string]int64  `json:"counters,omitempty"`
	MaxValue map[string]int64  `json:"maxValue,omitempty"`
	Label    map[string]string `json:"labels,omitempty"`

	// When the server started.
	StartTime time.Time `json:"startTime,omitempty"`
}

// assign returns a Handler to handle the specified name, or nil.
// The caller must hold s.mu.
func (s *Server) assign(ctx context.Context, name string) Handler {
	if s.builtin && strings.HasPrefix(name, "rpc.") {
		switch name {
		case rpcServerInfo:
			return methodFunc(s.handleRPCServerInfo)
		case rpcCancel:
			return methodFunc(s.handleRPCCancel)
		default:
			return nil // reserved
		}
	}
	return s.mux.Assign(ctx, name)
}

// pushError reports an error for the given request ID directly back to the
// client, bypassing the normal request handling mechanism.  The caller must
// hold s.mu when calling this method.
func (s *Server) pushError(err error) {
	s.log("Invalid request: %v", err)
	var jerr *Error
	if e, ok := err.(*Error); ok {
		jerr = e
	} else {
		jerr = &Error{code: code.FromError(err), message: err.Error()}
	}

	nw, err := encode(s.ch, jresponses{{
		V:  Version,
		ID: json.RawMessage("null"),
		E:  jerr,
	}})
	s.metrics.Count("rpc.errors", 1)
	s.metrics.CountAndSetMax("rpc.bytesWritten", int64(nw))
	if err != nil {
		s.log("Writing error response: %v", err)
	}
}

// cancel reports whether id is an active call.  If so, it also calls the
// cancellation function associated with id and removes it from the
// reservations. The caller must hold s.mu.
func (s *Server) cancel(id string) bool {
	cancel, ok := s.used[id]
	if ok {
		cancel()
		delete(s.used, id)
	}
	return ok
}

func (s *Server) versionOK(v string) bool {
	if v == "" {
		return s.allow1 // an empty version is OK if the server allows it
	}
	return v == Version // ... otherwise it must match the spec
}

// A task represents a pending method invocation received by the server.
type task struct {
	m Handler // the assigned handler (after assignment)

	ctx   context.Context // the context passed to the handler
	hreq  *Request        // the request passed to the handler
	batch bool            // whether the request was part of a batch

	val json.RawMessage // the result value (when complete)
	err error           // the error value (when complete)
}

type tasks []*task

func (ts tasks) responses(rpcLog RPCLogger) jresponses {
	var rsps jresponses
	for _, task := range ts {
		if task.hreq.id == nil {
			// Spec: "The Server MUST NOT reply to a Notification, including
			// those that are within a batch request.  Notifications are not
			// confirmable by definition, since they do not have a Response
			// object to be returned. As such, the Client would not be aware of
			// any errors."
			//
			// However, parse and validation errors must still be reported, with
			// an ID of null if the request ID was not resolvable.
			if c := code.FromError(task.err); c != code.ParseError && c != code.InvalidRequest {
				continue
			}
		}
		rsp := &jresponse{V: Version, ID: task.hreq.id, batch: task.batch}
		if rsp.ID == nil {
			rsp.ID = json.RawMessage("null")
		}
		if task.err == nil {
			rsp.R = task.val
		} else if e, ok := task.err.(*Error); ok {
			rsp.E = e
		} else if c := code.FromError(task.err); c != code.NoError {
			rsp.E = &Error{code: c, message: task.err.Error()}
		} else {
			rsp.E = &Error{code: code.InternalError, message: task.err.Error()}
		}
		rpcLog.LogResponse(task.ctx, &Response{
			id:     string(rsp.ID),
			err:    rsp.E,
			result: rsp.R,
		})
		rsps = append(rsps, rsp)
	}
	return rsps
}
