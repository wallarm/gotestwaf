FROM golang:1.13-alpine

ARG GOTESTWAF_VERSION="unknown"

WORKDIR $GOPATH/src/gotestwaf
COPY . .

ENV GO111MODULE=on
RUN go build -ldflags "-X main.Version=${GOTESTWAF_VERSION}" \
	-o gotestwaf -mod vendor /go/src/gotestwaf/cmd/

ENTRYPOINT ["/go/src/gotestwaf/gotestwaf"]
