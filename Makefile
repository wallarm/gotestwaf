GOTESTWAF_VERSION := $(shell git describe)

gotestwaf:
	docker build --build-arg GOTESTWAF_VERSION="$(GOTESTWAF_VERSION)" \
		--force-rm -t gotestwaf .

gotestwaf-bin:
	go build -ldflags "-X main.Version=$(GOTESTWAF_VERSION)" -o gotestwaf ./cmd

modsec:
	docker pull mendhak/http-https-echo:20
	docker run -d --rm -p 8088:8080 -t mendhak/http-https-echo:20
	docker pull owasp/modsecurity-crs:3.3.2-nginx
	docker run --rm -d -p 8080:80 -p 8443:443 -e PARANOIA=1 \
		-v ${PWD}/cmd/resources/default.conf:/etc/nginx/conf.d/default.conf \
		owasp/modsecurity-crs:3.3.2-nginx

scan_local:
	go run ./cmd --url=http://127.0.0.1:8080/ --verbose

scan_local_from_docker:
	docker run -v ${PWD}/reports:/go/src/gotestwaf/reports --network="host" \
		gotestwaf --url=http://127.0.0.1:8080/ --verbose

test:
	go test -count=1 -v ./...

lint:
	golangci-lint -v run ./...

tidy:
	go mod tidy
	go mod vendor

fmt:
	go fmt -w $(shell go list ./... | grep -v /vendor/)

delete_reports:
	rm -f ./reports/*.pdf
	rm -f ./reports/*.csv

.PHONY: gotestwaf gotestwaf-bin modsec scan_local \
	scan_local_from_docker test lint tidy fmt delete_reports
