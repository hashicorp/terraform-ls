package protocol

const TelemetryFormatVersion = 1

type TelemetryEvent struct {
	Version int `json:"v"`

	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
}
