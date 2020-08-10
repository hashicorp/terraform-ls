package handlers

import (
	"context"
	"time"
)

func newTickReporter(d time.Duration) *tickReporter {
	return &tickReporter{
		t:   time.NewTicker(d),
		rfs: make([]reportFunc, 0),
	}
}

type reportFunc func()

type tickReporter struct {
	t   *time.Ticker
	rfs []reportFunc
}

func (tr *tickReporter) AddReporter(f reportFunc) {
	tr.rfs = append(tr.rfs, f)
}

func (tr *tickReporter) StartReporting(ctx context.Context) {
	go func(ctx context.Context, tr *tickReporter) {
		for {
			select {
			case <-ctx.Done():
				tr.t.Stop()
				return
			case <-tr.t.C:
				for _, rf := range tr.rfs {
					rf()
				}
			}
		}
	}(ctx, tr)
}
