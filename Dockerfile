FROM golang:1.13

WORKDIR /go/src/gotestwaf
COPY . .

ENV GO111MODULE=on
RUN go build

ENTRYPOINT ["/go/src/gotestwaf/gotestwaf"]
