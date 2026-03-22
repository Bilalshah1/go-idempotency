package store

import (
    "context"
    "time"
)


type IdempotencyRecord struct {
    StatusCode  int       `json:"status_code"`
    Response    []byte    `json:"response"`
    PayloadHash string    `json:"payload_hash"`
    CreatedAt   time.Time `json:"created_at"`
}


type IdempotencyStore interface {
    Get(ctx context.Context, key string) (*IdempotencyRecord, error)
    Set(ctx context.Context, key string, rec *IdempotencyRecord, ttl time.Duration) error
    Close() error
}



type CreateOrderRequest struct {
    UserID string  `json:"user_id"`
    Item   string  `json:"item"`
    Amount float64 `json:"amount"`
}

type Order struct {
    ID             string    `json:"id"`
    UserID         string    `json:"user_id"`
    Item           string    `json:"item"`
    Amount         float64   `json:"amount"`
    IdempotencyKey string    `json:"idempotency_key"`
    CreatedAt      time.Time `json:"created_at"`
}