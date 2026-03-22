# Go Idempotency API

A Go HTTP server demonstrating **idempotent request handling** using Redis caching and PostgreSQL persistence.

## Features

- ✅ **Idempotent Order Creation** - Same request always returns the same response
- ✅ **HMAC-SHA256 Authentication** - All requests must be signed with a shared secret
- ✅ **Rate Limiting** - 10 requests per minute per client
- ✅ **Payload Fingerprinting** - Detects when the same key is used with different payloads
- ✅ **Redis Caching** - 24-hour idempotency cache for fast replays
- ✅ **PostgreSQL Persistence** - Orders stored in database with unique idempotency keys

## Architecture

```
Request → HMAC Auth Middleware → Rate Limiter → Handler
                                                    ↓
                                        Check Redis Cache
                                                    ↓
                                    Cache Hit? → Return Cached
                                        ↓ Miss
                                    Create Order (PostgreSQL)
                                        ↓
                                    Cache in Redis (24h)
                                        ↓
                                    Return Response
```

## Setup

### Prerequisites

- Docker & Docker Compose
- Go 1.21+
- PostgreSQL 16 & Redis 7 (via Docker)

### 1. Start Services

```bash
docker-compose down -v  # Clean slate
docker-compose up      # Start PostgreSQL & Redis
```

### 2. Run Server

```bash
go run main.go
```

## API Usage

### Create Order (with Idempotency)

**Request:**

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: key-001" \
  -H "X-Signature: <HMAC-SHA256>" \
  -d '{"user_id":"user-1","item":"book","amount":500}'
```

**Calculate X-Signature:**

```bash
echo -n '{"user_id":"user-1","item":"book","amount":500}' | \
  openssl dgst -sha256 -hmac "super-secret-key" | awk '{print $2}'
```

**Response:**

```json
{
  "order": {
    "id": "8d6e5a0d-dd64-4fe3-be44-4a9fdf65b305",
    "user_id": "user-1",
    "item": "book",
    "amount": 500,
    "idempotency_key": "key-001",
    "created_at": "2026-03-22T12:37:20Z"
  }
}
```

## Status Codes

| Code    | Scenario                                        |
| ------- | ----------------------------------------------- |
| **201** | Fresh order created OR cached response replayed |
| **400** | Missing `Idempotency-Key` header                |
| **401** | Invalid HMAC signature                          |
| **409** | Same key used with different payload            |
| **429** | Rate limit exceeded (>10/min)                   |
| **500** | Database error                                  |

## Data Storage

### PostgreSQL (Orders)

```bash
docker exec go-idempotency-postgres-1 psql -U bilal -d bilal -c "SELECT * FROM orders;"
```

### Redis (24-hour Cache)

```bash
docker exec go-idempotency-redis-1 redis-cli GET "idemp:key-001"
```

## Configuration

**Current Settings:**

- HMAC Secret: `super-secret-key`
- Rate Limit: 10 requests/minute
- Redis TTL: 24 hours
- PostgreSQL Port: 5433
- Redis Port: 6380

## Project Structure

```
go-idempotency/
├── main.go                 # HTTP server & handlers
├── docker-compose.yml      # PostgreSQL & Redis setup
├── test.sh                 # Integration tests
└── internal/
    ├── middleware/
    │   ├── hmac.go        # HMAC authentication
    │   └── rate_limiter.go # Rate limiting
    └── store/
        ├── idempotency.go  # Idempotency interface
        ├── redis_store.go  # Redis implementation
        ├── order_store.go  # PostgreSQL orders
        └── postgres_store.go
```

**Manual Test with Postman:**

1. Create POST request to `http://localhost:8080/orders`
2. Add headers: `Content-Type`, `Idempotency-Key`, `X-Signature`
3. Send same request twice → should get identical responses
4. Change body, keep key → should get 409 Conflict
