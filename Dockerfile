# --------------------------
# Stage 1: Build Go binary
# --------------------------
FROM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH:-amd64}

WORKDIR /app

# Copy dependencies and download modules
COPY go.mod go.sum ./
RUN go mod download

# Copy Go source files directly into /app
COPY app/* ./

# Build Go binary
RUN go build -o mcstats-go .

# --------------------------
# Stage 2: Download MinecraftStats CLI + Web
# --------------------------
FROM alpine:3.20 AS mcstats

ARG MCSTATS_VERSION=3.3.1

WORKDIR /opt

RUN apk add --no-cache openjdk21-jre wget unzip bash

# MinecraftStats CLI
RUN wget -q https://github.com/pdinklag/MinecraftStats/releases/download/v${MCSTATS_VERSION}/MinecraftStatsCLI.zip \
    && unzip MinecraftStatsCLI.zip -d /opt/mcstats \
    && rm MinecraftStatsCLI.zip

# MinecraftStats Web
RUN wget -q https://github.com/pdinklag/MinecraftStats/releases/download/v${MCSTATS_VERSION}/MinecraftStatsWeb.zip \
    && unzip MinecraftStatsWeb.zip -d /opt/web \
    && rm MinecraftStatsWeb.zip

# --------------------------
# Stage 3: Runtime Alpine image
# --------------------------
FROM alpine:3.20

RUN apk add --no-cache openjdk21-jre bash

RUN addgroup -S mcstats && adduser -S mcstats -G mcstats

RUN mkdir -p /config /opt/mcstats /opt/web \
    && chown -R mcstats:mcstats /config /opt

USER mcstats
WORKDIR /opt

COPY --from=builder /app/mcstats-go /usr/local/bin/mcstats-go
COPY --from=mcstats /opt/mcstats /opt/mcstats
COPY --from=mcstats /opt/web /opt/web
COPY --from=ghcr.io/tarampampam/microcheck:1.3.0 /bin/httpcheck /usr/local/bin/httpcheck

VOLUME ["/config"]

ENV PORT=8080
ENV INTERVAL_SECONDS=300

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/mcstats-go"]
