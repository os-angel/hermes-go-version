FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -buildvcs=false -ldflags="-s -w" -o /hermes-go ./cmd/agent

FROM node:20-alpine AS bridge-builder

WORKDIR /bridge
COPY bridge/package*.json ./
RUN npm install --omit=dev

FROM alpine:3.20

RUN apk add --no-cache ca-certificates nodejs

WORKDIR /app

COPY --from=builder /hermes-go /usr/local/bin/hermes-go
COPY --from=bridge-builder /bridge/node_modules /app/bridge/node_modules
COPY bridge /app/bridge

ENV HERMES_GO_HOME=/data
ENV HERMES_BRIDGE_JS=/app/bridge/bridge.js

VOLUME ["/data"]
EXPOSE 8080

ENTRYPOINT ["hermes-go"]
CMD ["--config", "/data/config.yaml"]
