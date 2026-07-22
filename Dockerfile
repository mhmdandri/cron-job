# Stage 1: Build binary
FROM golang:1.26.2-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum* ./
RUN go mod download

# Copy source files
COPY . .

# Build lightweight static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o cron-job main.go

# Stage 2: Minimal runtime image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Jakarta

WORKDIR /app

# Copy compiled binary and static assets from builder
COPY --from=builder /app/cron-job .
COPY --from=builder /app/static ./static
COPY --from=builder /app/.env.example ./.env

EXPOSE 8080

CMD ["./cron-job"]
