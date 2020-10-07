default: test

fmt:
	go run github.com/mh-cbon/go-fmt-fail ./...

test:
	go test -v -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: fmt test
