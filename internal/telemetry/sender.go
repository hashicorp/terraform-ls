package telemetry

import "context"

type Sender interface {
	SendEvent(ctx context.Context, name string, properties map[string]interface{})
}
