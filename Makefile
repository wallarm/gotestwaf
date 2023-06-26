GOTESTWAF_VERSION := $(shell git describe)

gotestwaf:
	DOCKER_BUILDKIT=1 docker build --force-rm -t gotestwaf .

gotestwaf_bin:
	go build -o gotestwaf -ldflags "-X github.com/wallarm/gotestwaf/internal/version.Version=$(GOTESTWAF_VERSION)" ./cmd

modsec:
	docker pull mendhak/http-https-echo:20
	docker run --rm -d --name gotestwaf_test_app -p 8088:8080 -t mendhak/http-https-echo:20
	docker pull owasp/modsecurity-crs:3.3.2-nginx
	docker run --rm -d --name gotestwaf_modsec -p 8080:80 -p 8443:443 -e BACKEND="http://172.17.0.1:8088" -e PARANOIA=1 \
		-v ${PWD}/resources/default.conf.template:/etc/nginx/templates/conf.d/default.conf.template \
		owasp/modsecurity-crs:3.3.2-nginx

modsec_down:
	docker kill gotestwaf_test_app gotestwaf_modsec

modsec_stat: gotestwaf
	docker pull mendhak/http-https-echo:20
	docker pull owasp/modsecurity-crs:3.3.2-nginx
	docker run --rm -d --name gotestwaf_test_app -p 8088:8080 -t mendhak/http-https-echo:20
	docker run --rm -d -p 8080:80 -p 8443:443 -e PARANOIA=1 --name modsec_paranoia_1 -e BACKEND="http://172.17.0.1:8088" \
		-v ${PWD}/resources/default.conf.template:/etc/nginx/templates/conf.d/default.conf.template \
		owasp/modsecurity-crs:3.3.2-nginx
	docker run --rm --init -v ${PWD}/reports:/app/reports --network="host" \
		gotestwaf --url=http://127.0.0.1:8080/ --workers 100 --ignoreUnresolved --wafName "ModSecurity PARANOIA 1" --noEmailReport
	docker kill modsec_paranoia_1
	docker run --rm -d -p 8080:80 -p 8443:443 -e PARANOIA=2 --name modsec_paranoia_2 -e BACKEND="http://172.17.0.1:8088" \
		-v ${PWD}/resources/default.conf.template:/etc/nginx/templates/conf.d/default.conf.template \
		owasp/modsecurity-crs:3.3.2-nginx
	docker run --rm --init -v ${PWD}/reports:/app/reports --network="host" \
		gotestwaf --url=http://127.0.0.1:8080/ --workers 100 --ignoreUnresolved --wafName "ModSecurity PARANOIA 2" --noEmailReport
	docker kill modsec_paranoia_2
	docker run --rm -d -p 8080:80 -p 8443:443 -e PARANOIA=3 --name modsec_paranoia_3 -e BACKEND="http://172.17.0.1:8088" \
		-v ${PWD}/resources/default.conf.template:/etc/nginx/templates/conf.d/default.conf.template \
		owasp/modsecurity-crs:3.3.2-nginx
	docker run --rm --init -v ${PWD}/reports:/app/reports --network="host" \
		gotestwaf --url=http://127.0.0.1:8080/ --workers 100 --ignoreUnresolved --wafName "ModSecurity PARANOIA 3" --noEmailReport
	docker kill modsec_paranoia_3
	docker run --rm -d -p 8080:80 -p 8443:443 -e PARANOIA=4 --name modsec_paranoia_4 -e BACKEND="http://172.17.0.1:8088" \
		-v ${PWD}/resources/default.conf.template:/etc/nginx/templates/conf.d/default.conf.template \
		owasp/modsecurity-crs:3.3.2-nginx
	docker run --rm --init -v ${PWD}/reports:/app/reports --network="host" \
		gotestwaf --url=http://127.0.0.1:8080/ --workers 100 --ignoreUnresolved --wafName "ModSecurity PARANOIA 4" --noEmailReport
	docker kill modsec_paranoia_4

scan_local:
	go run ./cmd --url=http://127.0.0.1:8080/ --workers 200 --noEmailReport

scan_local_from_docker:
	docker run --rm --init -v ${PWD}/reports:/app/reports --network="host" \
		gotestwaf --url=http://127.0.0.1:8080/ --workers 200 --noEmailReport

modsec_crs_regression_tests_convert:
	rm -rf .tmp/coreruleset
	rm -rf testcases/modsec_crs/
	rm -f testcases/false-pos/fp_*
	git clone https://github.com/coreruleset/coreruleset .tmp/coreruleset
	ruby misc/modsec_regression_testset_converter.rb
	mv testcases/modsec_crs/fp_* testcases/false-pos
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
