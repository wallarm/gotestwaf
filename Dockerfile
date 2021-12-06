FROM golang:1.17-alpine AS build
ARG GOTESTWAF_VERSION="unknown"
WORKDIR /app/
COPY . .
RUN go build -ldflags "-X github.com/wallarm/gotestwaf/internal/version.Version=${GOTESTWAF_VERSION}" \
		-o gotestwaf ./cmd/

FROM alpine
WORKDIR /app
RUN apk add --no-cache chromium fontconfig
RUN apk add --no-cache wget curl && \
    ( \
        cd /tmp && \
        curl -s https://api.github.com/repos/rsms/inter/releases/latest \
        | grep "browser_download_url.*zip" \
        | cut -d '"' -f 4 | wget -qi - -O inter.zip && \
        unzip inter.zip -d inter && \
        mv ./inter/Inter\ Desktop /usr/share/fonts/inter && \
        rm -rf ./inter* \
    ) && \
    fc-cache -fv && \
    apk del --no-cache wget curl
COPY --from=build /app/gotestwaf ./
COPY ./testcases/ ./testcases/
COPY ./config.yaml ./

ENTRYPOINT ["/app/gotestwaf"]
