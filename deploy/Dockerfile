# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /src/cmd/desktop/frontend
COPY cmd/desktop/frontend/package.json cmd/desktop/frontend/package-lock.json* ./
RUN npm install
COPY cmd/desktop/frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/cmd/desktop/frontend/dist ./cmd/gateway/frontend/dist
RUN CGO_ENABLED=0 go build -o /gateway ./cmd/gateway

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 1000 -S clawfirm && adduser -u 1000 -S clawfirm -G clawfirm \
    && mkdir -p /data && chown 1000:1000 /data
COPY --from=builder /gateway /usr/local/bin/gateway

# All state lives under /data (config.yml, data.db, memory/).
# Mount a host volume here:
#   docker run -v /path/on/host:/data clawfirm-gateway
# Ensure the host directory is owned by UID 1000:
#   chown -R 1000:1000 /path/on/host
ENV HOME=/data
VOLUME /data
USER clawfirm
EXPOSE 9988
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -qO- http://localhost:9988/health || exit 1
ENTRYPOINT ["gateway"]
