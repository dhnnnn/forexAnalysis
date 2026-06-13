# ════════════════════════════════════════
# Stage 1: Build Go binary
# ════════════════════════════════════════
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy module files dulu (cache layer)
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /forex-agent ./cmd/main.go

# ════════════════════════════════════════
# Stage 2: Runtime (minimal image)
# ════════════════════════════════════════
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary dari builder
COPY --from=builder /forex-agent .

# Copy config
COPY config/ ./config/

# Timezone WIB
ENV TZ=Asia/Jakarta

EXPOSE 8080

CMD ["./forex-agent"]
