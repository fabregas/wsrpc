
test:
	go test -v ./wsrpc

bench:
	go test -v -benchmem -bench . ./wsrpc

