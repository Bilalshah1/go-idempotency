#!/bin/bash

SECRET="super-secret-key"
BASE_URL="http://localhost:8080/orders"

sign() {
    echo -n "$1" | openssl dgst -sha256 -hmac "$SECRET" | awk '{print $2}'
}

send() {
    local label=$1
    local key=$2
    local body=$3
    local sig=$(sign "$body")

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "TEST: $label"
    echo "KEY:  $key"
    echo "BODY: $body"
    echo "SIG:  $sig"
    echo "------"
    
    curl -s -w "\nHTTP STATUS: %{http_code}\n" -X POST "$BASE_URL" \
        -H "Content-Type: application/json" \
        -H "Idempotency-Key: $key" \
        -H "X-Signature: $sig" \
        -d "$body"
    
    echo ""
}

# ── Test 1: Fresh request ─────────────────────────────────────
send \
    "Fresh order (should return 201)" \
    "key-001" \
    '{"user_id":"user-1","item":"book","amount":500}'

sleep 1

# ── Test 2: Exact replay ──────────────────────────────────────
send \
    "Replay same request (should return 201 from cache)" \
    "key-001" \
    '{"user_id":"user-1","item":"book","amount":500}'

sleep 1

# ── Test 3: Same key different body ──────────────────────────
send \
    "Same key, different body (should return 409)" \
    "key-001" \
    '{"user_id":"user-1","item":"laptop","amount":150000}'

sleep 1

# ── Test 4: Missing idempotency key ──────────────────────────
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST: No Idempotency-Key header (should return 400)"
echo "------"
BODY='{"user_id":"user-1","item":"book","amount":500}'
SIG=$(sign "$BODY")
curl -s -w "\nHTTP STATUS: %{http_code}\n" -X POST "$BASE_URL" \
    -H "Content-Type: application/json" \
    -H "X-Signature: $SIG" \
    -d "$BODY"
echo ""

sleep 1

# ── Test 5: Missing signature ─────────────────────────────────
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST: No X-Signature header (should return 401)"
echo "------"
curl -s -w "\nHTTP STATUS: %{http_code}\n" -X POST "$BASE_URL" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: key-002" \
    -d '{"user_id":"user-1","item":"book","amount":500}'
echo ""

sleep 1

# ── Test 6: Wrong signature ───────────────────────────────────
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST: Wrong X-Signature (should return 401)"
echo "------"
curl -s -w "\nHTTP STATUS: %{http_code}\n" -X POST "$BASE_URL" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: key-002" \
    -H "X-Signature: thisisawrongsignature" \
    -d '{"user_id":"user-1","item":"book","amount":500}'
echo ""

sleep 1

# ── Test 7: Rate limiter ──────────────────────────────────────
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST: Rate limiter (11 requests, last one should 429)"
echo "------"
BODY='{"user_id":"user-1","item":"book","amount":500}'
for i in {1..11}; do
    SIG=$(sign "$BODY")
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL" \
        -H "Content-Type: application/json" \
        -H "Idempotency-Key: rate-test-$i" \
        -H "X-Signature: $SIG" \
        -d "$BODY")
    echo "  Request $i → $STATUS"
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "ALL TESTS DONE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"