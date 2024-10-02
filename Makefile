BINARY_NAME=filemodtracker

.PHONY: clean daemon ui test

clean:
	go clean modcache
	rm -f $(BINARY_NAME)

daemon:
	sudo go run -ldflags="-extldflags=-Wl,-ld_classic,-no_warn_duplicate_libraries,-v" . daemon

ui:
	go run -ldflags="-extldflags=-Wl,-ld_classic,-no_warn_duplicate_libraries,-v" . ui

test:
	go test -v -coverprofile=cover.out.tmp -coverpkg=./... ./...

