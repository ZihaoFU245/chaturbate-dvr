FROM golang:1.26-alpine AS builder
WORKDIR /workspace

COPY ./ ./
RUN go build -o chaturbate-dvr .

FROM alpine:latest AS runnable
WORKDIR /usr/src/app

RUN apk add --no-cache \
	ca-certificates \
	ffmpeg

COPY --from=builder /workspace/chaturbate-dvr /chaturbate-dvr

ENTRYPOINT ["/chaturbate-dvr"]
