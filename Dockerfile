FROM --platform=$BUILDPLATFORM golang:alpine AS build

COPY go.mod /robertoTelegram/go.mod
COPY go.sum /robertoTelegram/go.sum
WORKDIR /robertoTelegram

ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go mod download

COPY . /robertoTelegram

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -trimpath -ldflags '-s -w' -o robertoTelegram

FROM alpine

RUN --mount=type=cache,target=/var/cache/apk \
    ln -s /var/cache/apk /etc/apk/cache && \
    apk add ca-certificates ffmpeg

COPY --from=build /robertoTelegram/robertoTelegram /usr/bin/

CMD ["robertoTelegram"]