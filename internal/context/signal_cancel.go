// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package context

import (
	"context"
	"log"
	"os"
	"os/signal"
)

func WithSignalCancel(ctx context.Context, l *log.Logger, sigs ...os.Signal) (
	context.Context, context.CancelFunc) {
	ctx, cancelFunc := context.WithCancel(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, sigs...)

	go func() {
		select {
		case sig := <-sigChan:
			l.Printf("Cancellation signal (%s) received", sig)
			cancelFunc()
		case <-ctx.Done():
		}
	}()

	f := func() {
		signal.Stop(sigChan)
		cancelFunc()
	}

	return ctx, f
}
