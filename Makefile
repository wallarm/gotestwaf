gotestwaf:
	docker build . --force-rm -t gotestwaf

modsec:
	docker pull owasp/modsecurity-crs
	docker run -p 8080:80 -d -e PARANOIA=1 --rm owasp/modsecurity-crs

scan_local:
	go run ./cmd --url=http://127.0.0.1:8080/ --verbose

scan_local_from_docker:
	docker run -v ${PWD}/reports:/go/src/gotestwaf/reports --network="host" gotestwaf --url=http://127.0.0.1:8080/ --verbose

lint:
	golangci-lint -v run ./...

tidy:
	go mod tidy
	go mod vendor

delete_reports:
	rm -f ./reports/*.pdf
	rm -f ./reports/*.csv

.PHONY: lint gotestwaf scan_local scan_local_from_docker modsec tidy

