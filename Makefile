BINARY_NAME=filemodtracker

.PHONY: all build clean run run-service run-ui test mocks

all: build

build:
	go build -o $(BINARY_NAME) main.go

clean:
	go clean modcache
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME) service & ./$(BINARY_NAME) ui &

run-service: build
	./$(BINARY_NAME) service

run-ui: build
	./$(BINARY_NAME) ui

rm-mocks:
	rm -rf ./testutils/mocks.*

gen-mocks:
	mockery --all --output=testutils/mocks --case=underscore --keeptree

mocks: rm-mocks gen-mocks

test:
	go test -v -coverprofile=cover.out.tmp -coverpkg=./... ./...

