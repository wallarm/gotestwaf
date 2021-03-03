FROM golang:1.13-alpine

WORKDIR $GOPATH/src/gotestwaf
COPY . .

ENV GO111MODULE=on
RUN go build -o gotestwaf -mod vendor /go/src/gotestwaf/cmd/main.go

ENTRYPOINT ["/go/src/gotestwaf/gotestwaf"]
