BINARY_NAME=filemodtracker

.PHONY: clean daemon ui test

clean:
	go clean modcache
	rm -f $(BINARY_NAME)

daemon:
	sudo go run main.go daemon

ui:
	go run main.go ui

test:
	go test -v -coverprofile=cover.out.tmp -coverpkg=./... ./...

