package registry

import (
	"time"
)

const (
	defaultBaseURL = "https://registry.terraform.io"
	defaultTimeout = 5 * time.Second
)

type Client struct {
	BaseURL          string
	Timeout          time.Duration
	ProviderPageSize int
}

func NewClient() Client {
	return Client{
		BaseURL:          defaultBaseURL,
		Timeout:          defaultTimeout,
		ProviderPageSize: 100,
	}
}
