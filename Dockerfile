FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/order-api ./cmd/order-api

FROM scratch
WORKDIR /app
COPY --from=builder /app/order-api .
COPY --from=builder /build/cmd/order-api/.env .
EXPOSE 8000 50051 8080
ENTRYPOINT ["./order-api"]
