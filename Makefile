.PHONY: apispec ogen golangci-lint test test-e2e lint check build set-version tag publish release

# Version for publishing (usage: make publish v=0.1.0)
v ?=

# E2E tests (requires S0_E2E_BASE_URL and S0_E2E_PASSWORD env vars)
test-e2e:
	@printf "Running E2E tests...\n"
	go test -v -count=1 -tags=e2e ./tests/e2e/... -timeout=30m

# Unit tests
test:
	@printf "Running unit tests...\n"
	go test -v -race -cover ./...

# Lint with golangci-lint
lint: golangci-lint
	$(GOLANGCI_LINT) run ./...

# Build verification
build:
	@printf "Verifying build...\n"
	go build ./...

# Run all checks
check: build test lint
	@printf "All checks passed!\n"

# Generate SDK code from OpenAPI spec
apispec: ogen
	@printf "Generating API spec code...\n"
	@PATH="$(LOCALBIN):$(PATH)" go generate ./pkg/apispec/...

# Set version by creating git tag
set-version:
ifndef v
	@echo "Error: version not specified. Usage: make set-version v=0.1.0"
	@exit 1
endif
	@echo "Creating tag v$(v)..."
	@git tag -a v$(v) -m "Release v$(v)"
	@echo "Tag v$(v) created successfully!"

# Create and push git tag (Go modules use git tags for versioning)
tag: set-version
ifndef v
	@echo "Error: version not specified. Usage: make tag v=0.1.0"
	@exit 1
endif
	@echo "Pushing tag v$(v) to origin..."
	@git push origin v$(v)
	@echo "Tag v$(v) pushed. Go proxy will index the new version automatically."

# Publish to Go proxy (just push the tag)
publish: check tag
	@echo "Published version v$(v) successfully!"
	@echo "Users can now use: go get github.com/sandbox0-ai/sdk-go@v$(v)"

# Full release workflow
release: publish
	@echo "Release v$(v) completed!"

# Tool binaries
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.8.0

golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT): $(LOCALBIN)
	@test -s $(GOLANGCI_LINT) && $(GOLANGCI_LINT) version | grep -q "$(GOLANGCI_LINT_VERSION)" || \
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

OGEN ?= $(LOCALBIN)/ogen
OGEN_VERSION ?= v1.18.0

ogen: $(OGEN)
$(OGEN): $(LOCALBIN)
	@test -s $(LOCALBIN)/ogen && $(LOCALBIN)/ogen -version | grep -q $(OGEN_VERSION) || \
	GOBIN=$(LOCALBIN) go install github.com/ogen-go/ogen/cmd/ogen@$(OGEN_VERSION)
