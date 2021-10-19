package telemetry

import (
	"context"
	"io/ioutil"
	"log"
)

type NoopSender struct {
	Logger *log.Logger
}

func (t *NoopSender) log() *log.Logger {
	if t.Logger != nil {
		return t.Logger
	}
	return log.New(ioutil.Discard, "", 0)
}

func (t *NoopSender) SendEvent(ctx context.Context, name string, properties map[string]interface{}) {
	t.log().Printf("telemetry disabled %q: %#v", name, properties)
}
