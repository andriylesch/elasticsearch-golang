BINARY=elasticsearch-golang

${BINARY}:
	CGO_ENABLED=0 go build -a -ldflags '-s' -installsuffix cgo -o $(BINARY) .

.PHONY: build
build:
	CGO_ENABLED=0 go build -a -ldflags '-s' -installsuffix cgo -o $(BINARY) .

.PHONY: run
run: 
	@go run main.go
