FROM golang:1.19-alpine AS build

RUN apk --no-cache add git

WORKDIR /app/
COPY . .

RUN go build -ldflags "-X github.com/wallarm/gotestwaf/internal/version.Version=$(git describe)" \
		-o gotestwaf ./cmd/

FROM alpine

RUN apk add --no-cache chromium fontconfig
RUN apk add --no-cache wget curl && \
	( \
		cd /tmp && \
		curl -s https://api.github.com/repos/rsms/inter/releases/latest \
		| grep "browser_download_url.*zip" \
		| cut -d '"' -f 4 | wget -qi - -O inter.zip && \
		unzip inter.zip -d inter && \
		mkdir -p /usr/share/fonts/inter && \
		mv ./inter/Inter\ Desktop/* /usr/share/fonts/inter/ && \
		rm -rf ./inter* && \
		curl -s https://api.github.com/repos/be5invis/Iosevka/releases/latest \
		| grep "browser_download_url.*ttf-iosevka-[0-9\.]*\.zip" \
		| cut -d '"' -f 4 | wget -qi - -O iosevka.zip && \
		unzip iosevka.zip -d iosevka && \
		mkdir -p /usr/share/fonts/ && \
		mv ./iosevka /usr/share/fonts/ && \
		rm -rf ./iosevka* \
	) && \
	fc-cache -fv && \
	apk del --no-cache wget curl

WORKDIR /app

COPY --from=build /app/gotestwaf ./
COPY ./testcases/ ./testcases/
COPY ./config.yaml ./

ENTRYPOINT ["/app/gotestwaf"]
