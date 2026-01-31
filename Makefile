.PHONY: apispec oapi-codegen

# Tool binaries
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

OAPI_CODEGEN ?= $(LOCALBIN)/oapi-codegen
OAPI_CODEGEN_VERSION ?= v2.4.1

# Generate SDK code from OpenAPI spec
apispec: oapi-codegen
	@printf "Generating API spec code...\n"
	@PATH="$(LOCALBIN):$(PATH)" go generate ./pkg/apispec/...

oapi-codegen: $(OAPI_CODEGEN)
$(OAPI_CODEGEN): $(LOCALBIN)
	@test -s $(LOCALBIN)/oapi-codegen && $(LOCALBIN)/oapi-codegen --version | grep -q $(OAPI_CODEGEN_VERSION) || \
	GOBIN=$(LOCALBIN) go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)
