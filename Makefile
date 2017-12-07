
test:
	go test -v ./wsrpc

bench:
	go test -v -cpu 1 -benchmem -bench . ./wsrpc

cov:
	go test -cover -coverprofile=coverage.out -v ./wsrpc/

