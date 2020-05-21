package session

import (
	"context"
	"log"

	"github.com/creachadair/jrpc2"

	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
)

type Session interface {
	Assigner() (jrpc2.Assigner, error)
	Finish(jrpc2.ServerStatus)
	SetLogger(*log.Logger)
	SetDiscoveryFunc(discovery.DiscoveryFunc)
}

type SessionFactory func(context.Context) Session
