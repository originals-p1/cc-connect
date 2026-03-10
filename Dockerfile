FROM golang:1.24-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 go build \
  -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
  -o /out/cc-connect \
  ./cmd/cc-connect

FROM node:22-alpine

RUN apk add --no-cache bash ca-certificates ffmpeg git tini

WORKDIR /data

COPY --from=builder /out/cc-connect /usr/local/bin/cc-connect

ENTRYPOINT ["/sbin/tini", "--", "cc-connect"]
CMD ["--config", "/data/config.toml"]
