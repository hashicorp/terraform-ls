package svcctl

import (
	"context"
	"log"

	"github.com/creachadair/jrpc2"
)

type Service interface {
	Assigner() (jrpc2.Assigner, error)
	Finish(jrpc2.ServerStatus)
	SetLogger(*log.Logger)
}

type ServiceFactory func(context.Context) Service
