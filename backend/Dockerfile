# =============================================================================
# Arkheion Backend — Multi-stage Dockerfile
# Stage 1: build   — Go compiler + dependencies
# Stage 2: final   — Minimal runtime image
# =============================================================================

# ---- Stage 1: Build ---------------------------------------------------------
FROM golang:1.22-alpine AS build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Download dependencies first (layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the binary
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s -X main.Version=${VERSION}" \
    -o /arkheion \
    .

# ---- Stage 2: Development (with air hot reload) ------------------------------
FROM golang:1.22-alpine AS development

RUN apk add --no-cache git ca-certificates curl tzdata wget
RUN go install github.com/cosmtrek/air@latest

WORKDIR /app

# air config
COPY --from=build /build/go.mod /build/go.sum ./
RUN go mod download

EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

# ---- Stage 3: Production runtime --------------------------------------------
# Use alpine (not scratch) so healthcheck tools (wget) are available
FROM alpine:3.19 AS final

RUN apk add --no-cache ca-certificates tzdata wget

# Copy the compiled binary
COPY --from=build /arkheion /arkheion

# Copy OpenAPI spec for serving at /api/v1/docs
COPY --from=build /build/docs /docs

EXPOSE 8080

ENTRYPOINT ["/arkheion"]
