/*
Package jrpc2 implements a server and a client for the JSON-RPC 2.0 protocol
defined by http://www.jsonrpc.org/specification.

Servers

The *Server type implements a JSON-RPC server. A server communicates with a
client over a channel.Channel, and dispatches client requests to user-defined
method handlers.  Handlers satisfy the jrpc2.Handler interface by exporting a
Handle method with this signature:

   Handle(ctx Context.Context, req *jrpc2.Request) (interface{}, error)

The handler package helps adapt existing functions to this interface.
A server finds the handler for a request by looking up its method name in a
jrpc2.Assigner provided when the server is set up.

For example, suppose we have defined the following Add function, and would like
to export it as a JSON-RPC method:

   // Add returns the sum of a slice of integers.
   func Add(ctx context.Context, values []int) int {
      sum := 0
      for _, v := range values {
         sum += v
      }
      return sum
   }

To convert Add to a jrpc2.Handler, call the handler.New function, which uses
reflection to lift its argument into the jrpc2.Handler interface:

   h := handler.New(Add)  // h is a jrpc2.Handler that invokes Add

We will advertise this function under the name "Add".  For static assignments
we can use a handler.Map, which finds methods by looking them up in a Go map:

   assigner := handler.Map{
      "Add": handler.New(Add),
   }

Equipped with an Assigner we can now construct a Server:

   srv := jrpc2.NewServer(assigner, nil)  // nil for default options

To serve requests, we will next need a channel.Channel. The channel package
exports functions that can adapt various input and output streams.  For this
example, we'll use a channel that delimits messages by newlines, and
communicates on os.Stdin and os.Stdout:

   ch := channel.Line(os.Stdin, os.Stdout)
   srv.Start(ch)

Once started, the running server will handle incoming requests until the
channel closes, or until it is stopped explicitly by calling srv.Stop(). To
wait for the server to finish, call:

   err := srv.Wait()

This will report the error that led to the server exiting. A working
implementation of this example can found in cmd/examples/adder/adder.go:

    $ go run cmd/examples/adder/adder.go

You can interact with this server by typing JSON-RPC requests on stdin.


Clients

The *Client type implements a JSON-RPC client. A client communicates with a
server over a channel.Channel, and is safe for concurrent use by multiple
goroutines. It supports batched requests and may have arbitrarily many pending
requests in flight simultaneously.

To establish a client we first need a channel:

   import "net"

   conn, err := net.Dial("tcp", "localhost:8080")
   ...
   ch := channel.RawJSON(conn, conn)
   cli := jrpc2.NewClient(ch, nil)  // nil for default options

To send a single RPC, use the Call method:

   rsp, err := cli.Call(ctx, "Add", []int{1, 3, 5, 7})

This blocks until the response is received. Any error returned by the server,
including cancellation or deadline exceeded, has concrete type *jrpc2.Error.

To issue a batch of requests all at once, use the Batch method:

   rsps, err := cli.Batch(ctx, []jrpc2.Spec{
      {Method: "Math.Add", Params: []int{1, 2, 3}},
      {Method: "Math.Mul", Params: []int{4, 5, 6}},
      {Method: "Math.Max", Params: []int{-1, 5, 3, 0, 1}},
   })

The Batch method waits until all the responses are received.  An error from the
Batch call reflects an error in sending the request: The caller must check each
response separately for errors from the server. The responses will be returned
in the same order as the Spec values, save that notifications are omitted.

To decode the result from a successful response use its UnmarshalResult method:

   var result int
   if err := rsp.UnmarshalResult(&result); err != nil {
      log.Fatalln("UnmarshalResult:", err)
   }

To shut down a client and discard all its pending work, call cli.Close().


Notifications

The JSON-RPC protocol also supports a kind of request called a notification.
Notifications differ from ordinary calls in that they are one-way: The client
sends them to the server, but the server does not reply.

A jrpc2.Client supports sending notifications as follows:

   err := cli.Notify(ctx, "Alert", handler.Obj{
      "message": "A fire is burning!",
   })

Unlike ordinary requests, there are no responses for notifications; a
notification is complete once it has been sent.

On the server side, notifications are identical to ordinary requests, save that
their return value is discarded once the handler returns. If a handler does not
want to do anything for a notification, it can query the request:

   if req.IsNotification() {
      return 0, nil  // ignore notifications
   }


Cancellation

The *Client and *Server types support a non-standard cancellation protocol,
that consists of a notification method "rpc.cancel" taking an array of request
IDs to be cancelled. Upon receiving this notification, the server will cancel
the context of each method handler whose ID is named.

When the context associated with a client request is cancelled, the client will
send an "rpc.cancel" notification to the server for that request's ID.  The
"rpc.cancel" method is automatically handled (unless disabled) by the *Server
implementation from this package.


Services with Multiple Methods

The examples above show a server with only one method using handler.New; you
will often want to expose more than one. The handler.NewService function
supports this by applying New to all the exported methods of a concrete value
to produce a handler.Map for those methods:

   type math struct{}

   func (math) Add(ctx context.Context, vals ...int) int { ... }
   func (math) Mul(ctx context.Context, vals []int) int { ... }

   assigner := handler.NewService(math{})

This assigner maps the name "Add" to the Add method, and the name "Mul" to the
Mul method, of the math value.

This may be further combined with the handler.Map type to allow different
services to work together:

   type status struct{}

   func (status) Get(context.Context) (string, error) {
      return "all is well", nil
   }

   assigner := handler.ServiceMap{
      "Math":   handler.NewService(math{}),
      "Status": handler.NewService(status{}),
   }

This assigner dispatches "Math.Add" and "Math.Mul" to the math value's methods,
and "Status.Get" to the status value's method. A ServiceMap splits the method
name on the first period ("."), and you may nest ServiceMaps more deeply if you
require a more complex hierarchy.


Non-Standard Extension Methods

By default a jrpc2.Server exports the following built-in non-standard extension
methods:

  rpc.serverInfo(null) ⇒ jrpc2.ServerInfo
  Returns a jrpc2.ServerInfo value giving server metrics.

  rpc.cancel([]int)  [notification]
  Request cancellation of the specified in-flight request IDs.

The rpc.cancel method works only as a notification, and will report an error if
called as an ordinary method.

These extension methods are enabled by default, but may be disabled by setting
the DisableBuiltin server option to true when constructing the server.


Server Notifications

The AllowPush option in jrpc2.ServerOptions enables the server to "push"
notifications back to the client. This is a non-standard extension of JSON-RPC
used by some applications such as the Language Server Protocol (LSP). The Push
method sends a notification back to the client, if this feature is enabled:

  if err := s.Push(ctx, "methodName", params); err == jrpc2.ErrNotifyUnsupported {
    // server notifications are not enabled
  }

A method handler may use jrpc2.ServerPush to access this functionality.  On the
client side, the OnNotify option in jrpc2.ClientOptions provides a callback to
which any server notifications are delivered if it is set.
*/
package jrpc2

// Version is the version string for the JSON-RPC protocol understood by this
// implementation, defined at http://www.jsonrpc.org/specification.
const Version = "2.0"
