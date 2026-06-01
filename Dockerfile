# --- Stage 1: Build ---
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /auth ./cmd/server

RUN printf 'm9d:x:65532:65532:maintainerd:/nonexistent:/sbin/nologin\n' > /tmp/passwd && \
    printf 'm9d:x:65532:\n' > /tmp/group

# --- Stage 2: Distroless static (m9d user) ---
FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /tmp/passwd /tmp/group /etc/
COPY --from=builder /auth /auth

EXPOSE 8080 8081

USER m9d

ENTRYPOINT ["/auth"]
