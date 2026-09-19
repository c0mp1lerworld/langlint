OAPI_CODEGEN_VERSION := v2.8.0
OAPI_CODEGEN := $(shell go env GOPATH)/bin/oapi-codegen
MOCKGEN_VERSION := v0.6.0
MOCKGEN := $(shell go env GOPATH)/bin/mockgen
SPEC := ../contracts/openapi/api.yaml
HANDLERS_DIR := internal/api/handlers
MODULE := github.com/c0mp1lerworld/langlint/backend
MOCKS_DIR := internal/api/ports/mocks
PROVISIONER_MOCKS_DIR := internal/provisioner/ports/mocks

.PHONY: tools generate mocks

tools:
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)
	go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION)

generate: tools
	mkdir -p $(HANDLERS_DIR)
	$(OAPI_CODEGEN) -config oapi-codegen-types.yaml $(SPEC)
	$(OAPI_CODEGEN) -config oapi-codegen-server.yaml $(SPEC)

mocks: tools
	mkdir -p $(MOCKS_DIR)
	$(MOCKGEN) -package mocks -destination $(MOCKS_DIR)/storage.go $(MODULE)/internal/api/ports/storage PracticeRepository,AnalysisRepository,ErrorMetricRepository,DeletionRequestRepository,AccessLogRepository,UnitOfWork
	$(MOCKGEN) -package mocks -destination $(MOCKS_DIR)/events.go $(MODULE)/internal/api/ports/events EventDispatcher,EventHandler,Outbox
	$(MOCKGEN) -package mocks -destination $(MOCKS_DIR)/llm_extractor.go $(MODULE)/internal/api/ports LLMExtractor
	mkdir -p $(PROVISIONER_MOCKS_DIR)
	$(MOCKGEN) -package mocks -destination $(PROVISIONER_MOCKS_DIR)/storage.go $(MODULE)/internal/provisioner/ports/storage AnalyticsSourceRepository,ErrorMetricRepository,RawDataRepository,DeletionRepository
