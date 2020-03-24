// Package server provides support routines for running jrpc2 servers.
package server

import (
	"net"
	"sync"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
)

// Service is the interface used by the Loop function to start up a server.
type Service interface {
	// This method is called to create an assigner and initialize the service
	// for use.  If it reports an error, the server is not started.
	Assigner() (jrpc2.Assigner, error)

	// This method is called when the server for this service has exited.
	Finish(jrpc2.ServerStatus)
}

type singleton struct{ assigner jrpc2.Assigner }

func (s singleton) Assigner() (jrpc2.Assigner, error) { return s.assigner, nil }
func (singleton) Finish(jrpc2.ServerStatus)           {}

// NewStatic creates a static (singleton) service from the given assigner.
func NewStatic(assigner jrpc2.Assigner) func() Service {
	svc := singleton{assigner}
	return func() Service { return svc }
}

// Loop obtains connections from lst and starts a server for each with the
// given service constructor and options, running in a new goroutine. If accept
// reports an error, the loop will terminate and the error will be reported
// once all the servers currently active have returned.
//
// TODO: Add options to support sensible rate-limitation.
func Loop(lst net.Listener, newService func() Service, opts *LoopOptions) error {
	newChannel := opts.framing()
	serverOpts := opts.serverOpts()
	log := func(string, ...interface{}) {}
	if serverOpts != nil && serverOpts.Logger != nil {
		log = serverOpts.Logger.Printf
	}

	var wg sync.WaitGroup
	for {
		conn, err := lst.Accept()
		if err != nil {
			if channel.IsErrClosing(err) {
				err = nil
			} else {
				log("Error accepting new connection: %v", err)
			}
			wg.Wait()
			return err
		}
		ch := newChannel(conn, conn)
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc := newService()
			assigner, err := svc.Assigner()
			if err != nil {
				log("Service initialization failed: %v", err)
				return
			}
			srv := jrpc2.NewServer(assigner, serverOpts).Start(ch)
			stat := srv.WaitStatus()
			svc.Finish(stat)
			if stat.Err != nil {
				log("Server exit: %v", stat.Err)
			}
		}()
	}
}

// LoopOptions control the behaviour of the Loop function.  A nil *LoopOptions
// provides default values as described.
type LoopOptions struct {
	// If non-nil, this function is used to convert a stream connection to an
	// RPC channel. If this field is nil, channel.RawJSON is used.
	Framing channel.Framing

	// If non-nil, these options are used when constructing the server to
	// handle requests on an inbound connection.
	ServerOptions *jrpc2.ServerOptions
}

func (o *LoopOptions) serverOpts() *jrpc2.ServerOptions {
	if o == nil {
		return nil
	}
	return o.ServerOptions
}

func (o *LoopOptions) framing() channel.Framing {
	if o == nil || o.Framing == nil {
		return channel.RawJSON
	}
	return o.Framing
}
