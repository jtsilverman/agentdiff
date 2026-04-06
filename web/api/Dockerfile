# Build stage
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o agentdiff-api ./web/api/

# Runtime stage
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/agentdiff-api /usr/local/bin/agentdiff-api
VOLUME /data
EXPOSE 8080
CMD ["agentdiff-api", "--port", "8080", "--db", "/data/agentdiff.db"]
