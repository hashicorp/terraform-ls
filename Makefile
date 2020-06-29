SOURCE_FILES ?= $$(find . -not -path "./vendor/*" -type f -name "*.go")

default: test

fmt:
	@gofmt -s -w $(SOURCE_FILES)

fmtcheck:
	@gofmt -s -l $(SOURCE_FILES) | grep ".*\.go"; if [ $$? -eq 0 ]; then exit 1; fi

test:
	go test -mod=vendor -v -cover ./...

.PHONY: fmt fmtcheck test
