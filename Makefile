APP_NAME := simply-cli
export GOWORK := off

.PHONY: run build vet test fmt tidy check clean

run:
	go run ./cmd/$(APP_NAME)

build:
	mkdir -p bin
	go build -o bin/$(APP_NAME) ./cmd/$(APP_NAME)

vet:
	go vet ./...

test:
	go test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

tidy:
	go mod tidy

check: fmt vet test build

clean:
	rm -rf bin/