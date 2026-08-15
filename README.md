# order-api

Clean Architecture order management. Both `CreateOrder` and `ListOrders` are exposed over REST, gRPC, and GraphQL. Creating an order also publishes an `OrderCreated` event to RabbitMQ.

## Run

```bash
docker compose up
```

That's the only command needed. It starts MySQL and RabbitMQ, waits for both to be healthy, then starts the app, which waits for the database, runs migrations, and starts serving.

The migrations also seed 5 orders, so `GET /order` (or the gRPC/GraphQL equivalents) returns data immediately, with no manual setup.

If you change the code and re-run, use `docker compose up --build`. Compose only rebuilds automatically when no image exists yet; otherwise it reuses whatever was built last, which is stale after any code change.

## Ports

| Service | Port |
|---|---|
| REST (web) | 8000 |
| gRPC | 50051 |
| GraphQL | 8080 |
| RabbitMQ management UI | 15672 |

## Test

`api.http` at the repo root has ready-to-run REST requests (create + list orders) for editors with an HTTP client extension (VS Code REST Client, JetBrains HTTP Client, etc).

REST:
```bash
curl -X POST http://localhost:8000/order -H "Content-Type: application/json" \
  -d '{"id":"a","price":100.5,"tax":0.5}'

curl http://localhost:8000/order
```

GraphQL: open http://localhost:8080 for the playground, or query directly:
```bash
curl -X POST http://localhost:8080/query -H "Content-Type: application/json" \
  -d '{"query":"query { listOrders { id Price Tax FinalPrice } }"}'
```

gRPC (with [grpcurl](https://github.com/fullstorydev/grpcurl), reflection is enabled):
```bash
grpcurl -plaintext localhost:50051 pb.OrderService/ListOrders
```

## Architecture

- `internal/entity`: `Order`, `OrderRepositoryInterface`
- `internal/usecase`: `CreateOrderUseCase`, `ListOrdersUseCase`
- `internal/infra/web`: REST handlers (chi)
- `internal/infra/grpc`: gRPC service (proto in `internal/infra/grpc/protofiles`)
- `internal/infra/graph`: GraphQL resolvers (gqlgen)
- `internal/infra/database`: MySQL repository + embedded migrations, applied automatically on startup
- `pkg/events`: event dispatcher. `CreateOrder` publishes an `OrderCreated` event to RabbitMQ

Dependency injection is wired with [google/wire](https://github.com/google/wire) (`cmd/order-api/wire.go` → generated `wire_gen.go`).
