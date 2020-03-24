package server

import (
	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
)

// Local represents a client and server connected by an in-memory pipe.
type Local struct {
	Server *jrpc2.Server
	Client *jrpc2.Client
}

// Close shuts down the client and waits for the server to exit, returning the
// result from the server's Wait method.
func (l Local) Close() error {
	l.Client.Close()
	return l.Server.Wait()
}

// NewLocal constructs a *jrpc2.Server and a *jrpc2.Client connected to it via
// an in-memory pipe, using the specified assigner and options.
// If opts == nil, it behaves as if the client and server options are also nil.
func NewLocal(assigner jrpc2.Assigner, opts *LocalOptions) Local {
	if opts == nil {
		opts = new(LocalOptions)
	}
	cpipe, spipe := channel.Direct()
	return Local{
		Server: jrpc2.NewServer(assigner, opts.Server).Start(spipe),
		Client: jrpc2.NewClient(cpipe, opts.Client),
	}
}

// LocalOptions control the behaviour of the server and client constructed by
// the NewLocal function.
type LocalOptions struct {
	Client *jrpc2.ClientOptions
	Server *jrpc2.ServerOptions
}
