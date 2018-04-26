BINARY=elasticsearch-golang
DIMAGENAME=elasticsearch-golang-image
DCONTAINERNAME=elasticsearch-golang-container

${BINARY}:
	CGO_ENABLED=0 go build -a -ldflags '-s' -installsuffix cgo -o $(BINARY) .

.PHONY: build
build:
	CGO_ENABLED=0 go build -a -ldflags '-s' -installsuffix cgo -o $(BINARY) .


.PHONY: run
run: 
	@go run main.go

.PHONY: test-unit
test-unit:
	go test -v `go list ./... | grep -v /vendor/` -tags=unit

.PHONY: kafka-run
kafka-run: 
	@docker run -p 2181:2181 -p 9092:9092 --env ADVERTISED_HOST=localhost --env ADVERTISED_PORT=9092 spotify/kafka

.PHONY: zipkin-run
zipkin-run: 
	@docker run -d -p 9411:9411 openzipkin/zipkin 


## FOR local testing
.PHONY: docker-build
docker-build:
	env GOOS=linux GOARH=amd64 make
	docker build -t ${DIMAGENAME}:latest .

.PHONY: docker-run
docker-run:
	@docker run --name "${DCONTAINERNAME}" ${DIMAGENAME}

.PHONY: docker
docker: 
	make docker-build && make docker-run

.PHONY: docker-rm
docker-rm: 
	docker stop ${DCONTAINERNAME}
	docker rm ${DCONTAINERNAME}
	docker rmi ${DIMAGENAME}
	rm -f ${BINARY}