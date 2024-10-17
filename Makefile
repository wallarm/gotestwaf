GOTESTWAF_VERSION := $(shell git describe --tags)

gotestwaf:
	DOCKER_BUILDKIT=1 docker build --force-rm -t gotestwaf .

gotestwaf_bin:
	go build -o gotestwaf \
		-ldflags "-X github.com/wallarm/gotestwaf/internal/version.Version=$(GOTESTWAF_VERSION)" \
		./cmd/gotestwaf

modsec:
	docker pull mendhak/http-https-echo:31
	docker run --rm -d --name gotestwaf_test_app -p 8088:8080 mendhak/http-https-echo:31
	docker pull owasp/modsecurity-crs:nginx-alpine
	docker run --rm -d --name gotestwaf_modsec -p 8080:8080 -p 8443:8443 \
		-e BACKEND="http://$$(docker inspect --format '{{.NetworkSettings.IPAddress}}' gotestwaf_test_app):8080" \
		-e PARANOIA=1 \
		owasp/modsecurity-crs:nginx-alpine

modsec_down:
	docker kill gotestwaf_test_app gotestwaf_modsec

modsec_stat: gotestwaf
	./misc/modsec_stat.sh

scan_local:
	go run ./cmd --url=http://127.0.0.1:8080/ --workers 200 --noEmailReport

scan_local_from_docker:
	docker run --rm -v ${PWD}/reports:/app/reports --network="host" \
		gotestwaf --url=http://127.0.0.1:8080/ --workers 200 --noEmailReport

modsec_crs_regression_tests_convert:
	rm -rf .tmp/coreruleset
	rm -rf testcases/modsec-crs/
	rm -rf testcases/modsec-crs-false-pos/
	git clone --depth 1 https://github.com/coreruleset/coreruleset .tmp/coreruleset
	ruby misc/modsec_regression_testset_converter.rb
	mkdir testcases/modsec-crs-false-pos
	mv testcases/modsec-crs/fp_* testcases/modsec-crs-false-pos/
	rm -rf .tmp

test:
	go test -count=1 -v ./...

lint:
	golangci-lint -v run ./...

tidy:
	go mod tidy
	go mod vendor

fmt:
	go fmt $(shell go list ./... | grep -v /vendor/)
	goimports -local "github.com/wallarm/gotestwaf" -w $(shell find . -type f -name '*.go' -not -name '*_mocks.go' -not -name '*.pb.go' -not -path "./vendor/*")


delete_reports:
	rm -f ./reports/*.pdf
	rm -f ./reports/*.csv

.PHONY: gotestwaf gotestwaf_bin modsec modsec_down scan_local \
	scan_local_from_docker test lint tidy fmt delete_reports
