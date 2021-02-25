gotestwaf:
	docker build . --force-rm -t gotestwaf

scan_local:
	docker run -v ${PWD}/reports:/go/src/gotestwaf/reports --network="host" gotestwaf --url=http://127.0.0.1:8080/ --verbose

lint:
	golangci-lint -v run ./...

tidy:
	go mod tidy
	go mod vendor

.PHONY: lint gotestwaf scan_local tidy

