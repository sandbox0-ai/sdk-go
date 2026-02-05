.PHONY: apispec ogen

# Generate SDK code from OpenAPI spec
apispec: ogen
	@printf "Generating API spec code...\n"
	@PATH="$(LOCALBIN):$(PATH)" go generate ./pkg/apispec/...

# Tool binaries
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

OGEN ?= $(LOCALBIN)/ogen
OGEN_VERSION ?= v1.18.0

ogen: $(OGEN)
$(OGEN): $(LOCALBIN)
	@test -s $(LOCALBIN)/ogen && $(LOCALBIN)/ogen -version | grep -q $(OGEN_VERSION) || \
	GOBIN=$(LOCALBIN) go install github.com/ogen-go/ogen/cmd/ogen@$(OGEN_VERSION)
