
all: deps gen build

rebuild: gen build

build:
	@GO111MODULE=on go build -o goliac ./cmd/goliac

test:
	@GO111MODULE=on go test -race -covermode=atomic -coverprofile=coverage.txt ./internal/...
	@go tool cover -html coverage.txt -o cover.html

gen: api_docs swagger

deps:
	@go install github.com/go-swagger/go-swagger/cmd/swagger@v0.30.5
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.1

################################
### Private
################################

api_docs:
	@echo "Installing swagger-merger" && npm install swagger-merger -g
	@swagger-merger -i $(PWD)/swagger/index.yaml -o $(PWD)/docs/api_docs/bundle.yaml

verifiers: verify_lint verify_swagger

verify_lint:
	@echo "Running $@"
	@golangci-lint run -D errcheck ./pkg/...

verify_swagger:
	@echo "Running $@"
	@swagger validate $(PWD)/docs/api_docs/bundle.yaml

swagger: verify_swagger
	@echo "Regenerate swagger files"
	@rm -f /tmp/configure_goliac.go
	@cp ./internal/swagger_gen/restapi/configure_goliac.go /tmp/configure_goliac.go 2>/dev/null || :
	@rm -rf ./internal/swagger_gen
	@mkdir ./internal/swagger_gen
	@swagger generate server -t ./internal/swagger_gen -f ./docs/api_docs/bundle.yaml
	@cp /tmp/configure_goliac.go ./internal/swagger_gen/restapi/configure_goliac.go 2>/dev/null || :
