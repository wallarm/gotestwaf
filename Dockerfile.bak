FROM golang:1.12 as builder

WORKDIR /go/src/gotestwaf
RUN git clone https://github.com/wallarm/gotestwaf.git .
RUN GO111MODULE=off go get github.com/jung-kurt/gofpdf
RUN GO111MODULE=off go get gopkg.in/yaml.v2
COPY config.yaml .
RUN go install -v ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o gotestwaf .

FROM alpine
COPY --from=builder /go/src/gotestwaf/testcases /gotestwaf/testcases
COPY --from=builder /go/src/gotestwaf/config.yaml /gotestwaf/config.yaml
COPY --from=builder /go/src/gotestwaf/gotestwaf /gotestwaf/gotestwaf
RUN echo "hosts: files dns" > /etc/nsswitch.conf
WORKDIR /gotestwaf
RUN apk --no-cache add ca-certificates
ENTRYPOINT ["/gotestwaf/gotestwaf"]
