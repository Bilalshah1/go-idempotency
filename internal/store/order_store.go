package store

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/google/uuid"
    _ "github.com/lib/pq"
)

type OrderStore struct {
    db *sql.DB
}

func NewOrderStore(dsn string) (*OrderStore, error) {
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("open db: %w", err)
    }
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("ping db: %w", err)
    }
    return &OrderStore{db: db}, nil
}

func (o *OrderStore) Init(ctx context.Context) error {
    _, err := o.db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS orders (
            id               TEXT PRIMARY KEY,
            user_id          TEXT NOT NULL,
            item             TEXT NOT NULL,
            amount           NUMERIC NOT NULL,
            idempotency_key  TEXT UNIQUE NOT NULL,
            created_at       TIMESTAMPTZ NOT NULL
        )
    `)
    return err
}

func (o *OrderStore) CreateOrderIdempotent(
    ctx context.Context,
    req CreateOrderRequest,
    idempKey string,
) (*Order, error) {

    order := &Order{
        ID:             uuid.New().String(),
        UserID:         req.UserID,
        Item:           req.Item,
        Amount:         req.Amount,
        IdempotencyKey: idempKey,
        CreatedAt:      time.Now().UTC(),
    }

    _, err := o.db.ExecContext(ctx, `
        INSERT INTO orders (id, user_id, item, amount, idempotency_key, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (idempotency_key) DO NOTHING
    `, order.ID, order.UserID, order.Item, order.Amount, order.IdempotencyKey, order.CreatedAt)

    if err != nil {
        return nil, fmt.Errorf("insert order: %w", err)
    }

    return order, nil
}

func (o *OrderStore) Close() error {
    return o.db.Close()
}
