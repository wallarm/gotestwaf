FROM golang:1.17-alpine AS build
ARG GOTESTWAF_VERSION="unknown"
WORKDIR /app/
COPY . .
RUN go build -ldflags "-X github.com/wallarm/gotestwaf/internal/version.Version=${GOTESTWAF_VERSION}" \
		-o gotestwaf ./cmd/

FROM alpine
WORKDIR /app
COPY --from=build /app/gotestwaf ./
COPY ./testcases/ ./testcases/
COPY ./config.yaml ./

ENTRYPOINT ["/app/gotestwaf"]
