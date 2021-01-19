gotestwaf:
	docker build . --force-rm -t gotestwaf

scan_local:
	docker run -v /tmp:/tmp/report gotestwaf --url=https://127.0.0.1:8080/

lint:
	golangci-lint -v run ./...

.PHONY: lint gotestwaf scan_local