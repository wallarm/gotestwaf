FROM golang:1.13

WORKDIR /go/src/gotestwaf
COPY . .

ENV GO111MODULE=on
RUN go get github.com/jung-kurt/gofpdf
RUN go get gopkg.in/yaml.v2
RUN go install -v ./...

RUN go build

ENTRYPOINT ["/go/src/gotestwaf/gotestwaf"]
