BINARY_NAME=filemodtracker

.PHONY: clean daemon ui test install

clean:
	go clean modcache
	rm -f $(BINARY_NAME)

daemon:
	sudo go run -ldflags="-extldflags=-Wl,-ld_classic,-no_warn_duplicate_libraries,-v" . daemon

ui:
	filemodtracker ui

test:
	go test -v -coverprofile=cover.out.tmp -coverpkg=./... ./...

install:
	go install -ldflags="-extldflags=-Wl,-ld_classic,-no_warn_duplicate_libraries,-v" .