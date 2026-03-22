package store

import (
    "context"
    "encoding/json"
    "errors"
    "time"

    "github.com/redis/go-redis/v9"
)

type RedisStore struct {
    client *redis.Client
}

func NewRedisStore(addr string) *RedisStore {
    client := redis.NewClient(&redis.Options{
        Addr: addr,
    })
    return &RedisStore{client: client}
}

func (r *RedisStore) Get(ctx context.Context, key string) (*IdempotencyRecord, error) {
    val, err := r.client.Get(ctx, redisKey(key)).Bytes()
    if errors.Is(err, redis.Nil) {
        return nil, nil // key doesn't exist — first time seeing this request
    }
    if err != nil {
        return nil, err
    }

    var rec IdempotencyRecord
    if err := json.Unmarshal(val, &rec); err != nil {
        return nil, err
    }
    return &rec, nil
}

func (r *RedisStore) Set(ctx context.Context, key string, rec *IdempotencyRecord, ttl time.Duration) error {
    data, err := json.Marshal(rec)
    if err != nil {
        return err
    }
    return r.client.Set(ctx, redisKey(key), data, ttl).Err()
}

func (r *RedisStore) Close() error {
    return r.client.Close()
}

func redisKey(key string) string {
    return "idemp:" + key
}