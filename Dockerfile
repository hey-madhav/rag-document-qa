FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install CA certificates for HTTPS during dependency download.
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /out/server /app/server
COPY config.yml /app/config.yml

EXPOSE 8080

ENTRYPOINT ["/app/server"]
