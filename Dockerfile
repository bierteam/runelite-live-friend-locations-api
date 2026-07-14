# syntax=docker.io/docker/dockerfile:1@sha256:87999aa3d42bdc6bea60565083ee17e86d1f3339802f543c0d03998580f9cb89

FROM golang:1.26-alpine@sha256:9097beb5536220f7857bdcb65c1b4b340630dd7a70b85f03d5af29640b06693d AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/friend-tracker ./...

FROM gcr.io/distroless/cc:nonroot@sha256:d97bc0a941b8d4be647dc0ee75b264ddbb772f1ac5ba690a4309c00723b23775
WORKDIR /app
COPY --from=builder /app/friend-tracker ./friend-tracker

EXPOSE 3000
ENTRYPOINT ["./friend-tracker"]
