# syntax=docker/dockerfile:1

# Build Stage ==================================================================
FROM golang:1.19-alpine AS build

RUN apk --no-cache add git

WORKDIR /app
COPY . .

RUN go build -o gotestwaf -ldflags "-X github.com/wallarm/gotestwaf/internal/version.Version=$(git describe --tags)" ./cmd/


# Main Stage ===================================================================
FROM alpine

# Prepare environment
RUN <<EOF
    set -e -o pipefail

    # install all dependencies
    apk add --no-cache  \
        tini            \
        chromium        \
        font-inter      \
        font-iosevka    \
        fontconfig

    fc-cache -fv

    # add non-root user
    addgroup gtw
    adduser -D -G gtw gtw

    # create dir for application
    mkdir /app
    mkdir /app/reports
    chown -R gtw:gtw /app
EOF

WORKDIR /app

COPY --from=build /app/gotestwaf ./
COPY ./testcases ./testcases
COPY ./config.yaml ./

USER gtw

VOLUME [ "/app/reports" ]

ENTRYPOINT [ "/sbin/tini", "--", "/app/gotestwaf" ]
