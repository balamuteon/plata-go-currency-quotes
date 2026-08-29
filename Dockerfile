FROM golang:1.26.1 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o quotes ./cmd/quotes

FROM gcr.io/distroless/base-debian12
WORKDIR /

COPY --from=builder /app/quotes /quotes

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/quotes"]
