default: test

test:
	go test -mod=vendor ./...

.PHONY: test
