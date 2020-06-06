SOURCE_FILES ?= $$(find . -not -path "./vendor/*" -type f -name "*.go")

default: test

fmt:
	@gofmt -s -w $(SOURCE_FILES)

fmtcheck:
	@gofmt -s -l $(SOURCE_FILES) | grep ^; if [ $$? -eq 0 ]; then exit 1; fi

test:
	go test -mod=vendor ./...

.PHONY: fmt fmtcheck test
