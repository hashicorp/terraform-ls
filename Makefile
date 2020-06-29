export GOFLAGS = -mod=vendor

default: test

fmt:
	go run github.com/mh-cbon/go-fmt-fail ./...

test:
	go test -v -cover ./...

.PHONY: fmt test
