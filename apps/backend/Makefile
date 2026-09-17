OAPI_CODEGEN_VERSION := v2.8.0
OAPI_CODEGEN := $(shell go env GOPATH)/bin/oapi-codegen
SPEC := ../contracts/openapi/api.yaml
HANDLERS_DIR := internal/api/handlers

.PHONY: tools generate

tools:
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)

generate: tools
	mkdir -p $(HANDLERS_DIR)
	$(OAPI_CODEGEN) -config oapi-codegen-types.yaml $(SPEC)
	$(OAPI_CODEGEN) -config oapi-codegen-server.yaml $(SPEC)
