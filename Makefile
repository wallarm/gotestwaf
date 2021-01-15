gotestwaf:
	docker build . --force-rm -t gotestwaf

lint:
	golangci-lint -v run ./...

.PHONY: lint gotestwaf scan_local