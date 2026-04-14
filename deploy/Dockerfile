# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /src/cmd/desktop/frontend
COPY cmd/desktop/frontend/package.json cmd/desktop/frontend/package-lock.json* ./
RUN npm install
COPY cmd/desktop/frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS builder
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
    && addgroup -S clawfirm && adduser -S clawfirm -G clawfirm \
    && mkdir -p /var/lib/clawfirm && chown clawfirm:clawfirm /var/lib/clawfirm
COPY --from=builder /gateway /usr/local/bin/gateway
USER clawfirm
ENV HOME=/var/lib/clawfirm
EXPOSE 9988
ENTRYPOINT ["gateway"]
