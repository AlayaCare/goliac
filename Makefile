
test:
	@GO111MODULE=on go test -race -covermode=atomic -coverprofile=coverage.txt ./internal/...
	@go tool cover -html coverage.txt -o cover.html

