FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o agent-orchestrator ./src

FROM alpine:latest AS runtime

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder /app/agent-orchestrator .

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/agent-orchestrator"]
