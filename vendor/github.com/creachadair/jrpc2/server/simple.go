package server

import (
	"errors"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
)

// A Simple server manages execution of a server for a single service instance.
type Simple struct {
	running bool
	svc     Service
	opts    *jrpc2.ServerOptions
}

// NewSimple constructs a new, unstarted *Simple instance for the given
// service.  When run, the server will use the specified options.
func NewSimple(svc Service, opts *jrpc2.ServerOptions) *Simple {
	return &Simple{svc: svc, opts: opts}
}

// Run starts a server on the given channel, and blocks until it returns.  The
// server exit status is reported to the service, and the error value returned.
// Once Run returns, it can be run again with a new channel.
//
// If the caller does not need the error value and does not want to wait for
// the server to complete, call Run in a goroutine.
func (s *Simple) Run(ch channel.Channel) error {
	if s.running { // sanity check
		return errors.New("server is already running")
	}
	assigner, err := s.svc.Assigner()
	if err != nil {
		return err
	}
	s.running = true
	srv := jrpc2.NewServer(assigner, s.opts).Start(ch)
	stat := srv.WaitStatus()
	s.svc.Finish(stat)
	s.running = false
	return stat.Err
}
