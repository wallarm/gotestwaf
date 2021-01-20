FROM golang:1.13-alpine

WORKDIR $GOPATH/src/gotestwaf
COPY . .

ENV GO111MODULE=on
RUN go build -mod vendor

ENTRYPOINT ["/go/src/gotestwaf/gotestwaf"]
