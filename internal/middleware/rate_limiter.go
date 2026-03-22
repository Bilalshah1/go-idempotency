package middleware

import (
    "fmt"
    "net/http"
    "time"

    "github.com/redis/go-redis/v9"
    "context"
)

type RateLimiter struct {
    client   *redis.Client
    limit    int
    window   time.Duration
}

func NewRateLimiter(client *redis.Client, limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        client: client,
        limit:  limit,
        window: window,
    }
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

        // Use client IP as the rate limit key
        clientIP := r.RemoteAddr
        key := fmt.Sprintf("ratelimit:%s", clientIP)

        ctx := context.Background()

        // Increment the counter for this client
        count, err := rl.client.Incr(ctx, key).Result()
        if err != nil {
            http.Error(w, "internal error", http.StatusInternalServerError)
            return
        }

        // On first request, set the expiry for the window
        if count == 1 {
            rl.client.Expire(ctx, key, rl.window)
        }

        // Add headers so the client knows their limit status
        w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.limit))
        w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, rl.limit-int(count))))

        // Reject if over limit
        if int(count) > rl.limit {
            http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
            return
        }

        next.ServeHTTP(w, r)
    })
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
