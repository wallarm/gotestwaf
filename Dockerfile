# syntax=docker/dockerfile:1

# Build Stage ==================================================================
FROM golang:1.19-alpine AS build

RUN apk --no-cache add git curl

WORKDIR /fonts

ADD https://api.github.com/repos/rsms/inter/releases/latest inter_version.json
RUN <<EOF
    set -e -o pipefail

    # download inter fonts
    (
        cd /tmp
        curl -s https://api.github.com/repos/rsms/inter/releases/latest \
            | grep "browser_download_url.*zip"                          \
            | cut -d '"' -f 4                                           \
            | xargs -I {} curl -s -L -o inter.zip {}

        mkdir inter && unzip inter.zip -d inter
        mkdir -p /fonts/inter
        mv ./inter/Inter\ Desktop/* /fonts/inter/
        rm -rf ./inter*
    )
EOF

ADD https://api.github.com/repos/be5invis/Iosevka/releases/latest iosevka_version.json
RUN <<EOF
    set -e -o pipefail

    # download iosevka fonts
    (
        cd /tmp
        curl -s https://api.github.com/repos/be5invis/Iosevka/releases/latest \
            | grep "browser_download_url.*ttf-iosevka-[0-9\.]*\.zip"          \
            | cut -d '"' -f 4                                                 \
            | xargs -I {} curl -s -L -o iosevka.zip {}

        mkdir iosevka && unzip iosevka.zip -d iosevka
        mv ./iosevka /fonts/
        rm -rf ./iosevka*
    )
EOF

WORKDIR /app
COPY . .

RUN go build -o gotestwaf -ldflags "-X github.com/wallarm/gotestwaf/internal/version.Version=$(git describe)" ./cmd/


# Main Stage ===================================================================
FROM alpine

# Prepare environment
RUN <<EOF
    set -e -o pipefail

    # install all dependencies
    apk add --no-cache chromium fontconfig

    # add non-root user
    addgroup gtw
    adduser -D -G gtw gtw

    # create dir for application
    mkdir /app
    chown gtw:gtw /app
EOF

# add fonts
COPY --from=build /fonts/inter /usr/share/fonts/inter
COPY --from=build /fonts/iosevka /usr/share/fonts/iosevka
RUN fc-cache -fv

WORKDIR /app

COPY --from=build /app/gotestwaf ./
COPY ./testcases ./testcases
COPY ./config.yaml ./

USER gtw

VOLUME [ "/app/reports" ]

ENTRYPOINT [ "/app/gotestwaf" ]
