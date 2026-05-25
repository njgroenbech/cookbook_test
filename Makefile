.PHONY: lint test vet test-integration install-hooks

APP_DIR := ./app

lint:
	cd $(APP_DIR) && golangci-lint run ./...

vet:
	cd $(APP_DIR) && go vet ./...

test:
	cd $(APP_DIR) && go test -race ./...

test-integration:
	cd $(APP_DIR) && go test -race -tags integration -coverprofile=coverage.out ./...

install-hooks:
	lefthook install
