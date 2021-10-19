package session

import (
	"context"
	"log"

	"github.com/creachadair/jrpc2"
)

type Session interface {
	Assigner() (jrpc2.Assigner, error)
	Finish(jrpc2.Assigner, jrpc2.ServerStatus)
	SetLogger(*log.Logger)
}

type ClientNotifier interface {
	Notify(ctx context.Context, method string, params interface{}) error
}

type SessionFactory func(context.Context) Session
