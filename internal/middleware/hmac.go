package middleware

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "io"
    "net/http"
    "bytes"
)

// HMACAuth returns a middleware that verifies every request is signed
// with the shared secret. Unsigned or tampered requests are rejected.
func HMACAuth(secret string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

            // 1. Read the signature the client sent
            clientSig := r.Header.Get("X-Signature")
            if clientSig == "" {
                http.Error(w, "missing X-Signature header", http.StatusUnauthorized)
                return
            }

            // 2. Read the request body
            body, err := io.ReadAll(r.Body)
            if err != nil {
                http.Error(w, "cannot read body", http.StatusBadRequest)
                return
            }
            // Restore the body so the next handler can read it again
            r.Body = io.NopCloser(bytes.NewReader(body))

            // 3. Compute what the signature SHOULD be
            mac := hmac.New(sha256.New, []byte(secret))
            mac.Write(body)
            expectedSig := hex.EncodeToString(mac.Sum(nil))

            // 4. Compare — use hmac.Equal to prevent timing attacks
            if !hmac.Equal([]byte(clientSig), []byte(expectedSig)) {
                http.Error(w, "invalid signature", http.StatusUnauthorized)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

        