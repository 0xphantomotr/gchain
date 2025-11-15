run-node:
    go run ./cmd/gchain-node
run-light:
    go run ./cmd/gchain-light
fmt:
    gofmt -w .
test:
    go test ./...