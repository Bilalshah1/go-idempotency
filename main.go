package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-idempotency/internal/middleware"
	"go-idempotency/internal/store"

	"github.com/redis/go-redis/v9"
)

const hmacSecret = "super-secret-key"

var ctx = context.Background()

func createOrderHandler(
	idempStore store.IdempotencyStore,
	orderStore *store.OrderStore,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			http.Error(w, "Idempotency-Key header required", http.StatusBadRequest)
			return
		}

	
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read body", http.StatusBadRequest)
			return
		}

		var req store.CreateOrderRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		h := sha256.Sum256(body)
		payloadHash := hex.EncodeToString(h[:])

		
		rec, err := idempStore.Get(ctx, key)
		if err != nil {
			log.Printf("redis error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if rec != nil {
			// Key exists — this is a replay
			if rec.PayloadHash != payloadHash {
				// Same key, different body — client is doing something wrong
				http.Error(w, "idempotency key reused with different payload", http.StatusConflict)
				return
			}
			// Replay the exact same response
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Idempotent-Replayed", "true")
			w.WriteHeader(rec.StatusCode)
			w.Write(rec.Response)
			log.Printf("replayed key=%s", key)
			return
		}

		
		order, err := orderStore.CreateOrderIdempotent(ctx, req, key)
		if err != nil {
			log.Printf("create order error: %v", err)
			http.Error(w, "failed to create order", http.StatusInternalServerError)
			return
		}

		
		respBody, _ := json.Marshal(map[string]any{"order": order})

		_ = idempStore.Set(ctx, key, &store.IdempotencyRecord{
			StatusCode:  http.StatusCreated,
			Response:    respBody,
			PayloadHash: payloadHash,
			CreatedAt:   time.Now().UTC(),
		}, 24*time.Hour)

		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(respBody)
		log.Printf("created order key=%s id=%s", key, order.ID)
	}
}

func main() {

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6380"})
	redisStore := store.NewRedisStore("localhost:6380")

	orderStore, err := store.NewOrderStore(
		"postgres://bilal:bilal123@localhost:5433/bilal?sslmode=disable",
	)
	if err != nil {
		log.Fatalf("postgres failed: %v", err)
	}
	defer redisStore.Close()
	defer orderStore.Close()

	if err := orderStore.Init(ctx); err != nil {
		log.Fatalf("table init failed: %v", err)
	}
	rateLimiter := middleware.NewRateLimiter(redisClient, 10, time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("/orders", createOrderHandler(redisStore, orderStore))

	
	handler := middleware.HMACAuth(hmacSecret)(rateLimiter.Limit(mux))

	
	srv := &http.Server{Addr: ":8080", Handler: handler}

	go func() {
		log.Println("server running → http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Block until Ctrl+C or kill signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}
