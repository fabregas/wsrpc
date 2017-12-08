
test:
	go test -v .

bench:
	go test -v -cpu 1 -benchmem -bench .

cov:
	go test -cover -coverprofile=coverage.out -v .

