FROM golang:1.12

WORKDIR /go/src/gotestwaf
COPY . .

RUN go get github.com/jung-kurt/gofpdf
RUN go get gopkg.in/yaml.v2
RUN go install -v ./...

RUN go build gotestwaf

ENTRYPOINT ["/go/src/gotestwaf/gotestwaf"]
