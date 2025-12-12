FROM golang:1.25-alpine AS builder
WORKDIR /workspace

COPY ./ ./
RUN go build -o chaturbate-dvr .

FROM scratch AS runnable
WORKDIR /usr/src/app

COPY --from=builder /workspace/chaturbate-dvr /chaturbate-dvr

ENTRYPOINT ["/chaturbate-dvr"]