default: test

test:
	go test -mod=vendor -v -cover ./...

.PHONY: test
