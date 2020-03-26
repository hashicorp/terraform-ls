VERSION?=$$(curl -s "https://api.github.com/repos/hashicorp/terraform-ls/releases/latest" | jq -r .name)

default: test

test:
	go test -mod=vendor ./...

build:
ifeq (,$(VERSION))
	@echo "ERROR: Set VERSION to a valid semver version. For example,";
	@echo " VERSION=0.1.0";
	@exit 1;
endif
	$(eval LDFLAGS := "-s -w -X github.com/hashicorp/terraform-ls/version.Version="$$(VERSION))
	go install -ldflags $(LDFLAGS) .

.PHONY: test build
