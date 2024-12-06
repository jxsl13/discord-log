FROM golang:alpine as build


LABEL maintainer "github.com/jxsl13"

RUN apk --update add git openssh && \
    rm -rf /var/lib/apt/lists/* && \
    rm /var/cache/apk/*

WORKDIR /build
COPY . ./
COPY go.* ./


RUN CGO_ENABLED=0 go build -a -ldflags '-w -extldflags "-static"' -o discord-log .


FROM alpine:latest as minimal

WORKDIR /app
COPY --from=build /build/discord-log .
VOLUME ["/data", "/app/.env"]
ENTRYPOINT ["/app/discord-log"]