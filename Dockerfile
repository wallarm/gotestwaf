FROM golang:1.17-alpine AS build
WORKDIR /app/
COPY . .
RUN go build -o gotestwaf ./cmd/main.go

FROM alpine
WORKDIR /app
COPY --from=build /app/gotestwaf ./
COPY ./testcases/ ./testcases/
COPY ./config.yaml ./

ENTRYPOINT ["/app/gotestwaf"]
