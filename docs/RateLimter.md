## 1. **Overview**

This middleware is an **IP + endpoint-based rate limiter** using the **token bucket algorithm**.

* Each IP has a separate limiter per endpoint.
* Limit is configurable via `refillRate` (requests per second) and `burst` (max tokens).
* Requests exceeding the limit are **blocked** and return a JSON 429 response.
* A **cleanup routine** removes inactive clients to save memory.

---

## 2. **Structures**

### `Client`

```go
type Client struct {
    limiter  *rate.Limiter // token bucket limiter
    lastSeen time.Time     // last request time
}
```

### `RateLimiter`

```go
type RateLimiter struct {
    clients       map[string]*Client
    mu            sync.RWMutex
    maxRequests   int
    refillRate    float64
    burst         int
    blockDuration time.Duration

    blockedRequests int // metric
}
```

* `clients`: stores limiter per `"IP|endpoint"`
* `mu`: protects the map
* `refillRate` & `burst`: token bucket parameters
* `blockedRequests`: tracks blocked requests

---

## 3. **Client IP Extraction**

Function `getClientIP(c fiber.Ctx)` extracts the real client IP:

1. Checks `Forwarded` header according to RFC 7239
2. Checks common proxy headers: `X-Forwarded-For`, `X-Real-IP`, `CF-Connecting-IP`, etc.
3. Fallback: `c.IP()`

Helper `parseForwardedHeader` parses:

```
Forwarded: for=1.2.3.4;by=...
```

and returns `1.2.3.4`.

---

## 4. **Limiter Retrieval**

```go
func (rl *RateLimiter) getLimiter(ip, endpoint string) *rate.Limiter
```

* Key = `"IP|endpoint"`
* If client does not exist → create a new token bucket
* If client exists → update `lastSeen`

---

## 5. **Middleware Handler**

```go
func (rl *RateLimiter) RateLimiterMiddleware() fiber.Handler
```

* Retrieves IP and endpoint
* Checks token bucket using `limiter.Allow()`
* If allowed → `c.Next()`
* If blocked → return JSON 429:

```json
{
  "error": "Too many requests",
  "message": "Try again later"
}
```

---

## 6. **Cleanup Routine**

```go
func (rl *RateLimiter) Cleanup(ctx context.Context)
```

* Runs every `cleanupInterval` seconds
* Removes clients where `lastSeen > clientTTL`
* Prevents memory leak in long-running applications

---

## 7. **Metrics**

```go
func (rl *RateLimiter) Metrics() (activeClients int, blocked int)
```

* `activeClients`: number of active IP|endpoint limiters
* `blocked`: total blocked requests

---

## 8. **Usage Example**

```go
func main() {
    app := fiber.New()
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Create a rate limiter: 5 RPS, burst 10, block 60s
    rl := NewRateLimiter(5, 5, 10, 60)

    // Apply middleware to the whole app
    app.Use(rl.RateLimiterMiddleware())

    // Example endpoint
    app.Get("/api/v1/download", func(c *fiber.Ctx) error {
        return c.SendString("Download OK")
    })

    // Start cleanup routine
    go rl.Cleanup(ctx)

    app.Listen(":3000")
}
```

---

## 9. **Workflow**

1. Client sends a request → middleware extracts IP
2. Retrieve limiter for `"IP|endpoint"`
3. Check token bucket:

   * If tokens left → allow request
   * If no tokens → block request, increment `blockedRequests`
4. Cleanup routine deletes inactive clients
5. Metrics provide monitoring data

---

## 10. **Advantages**

* Limits requests to prevent **basic DDoS**
* JSON response standard, no external packages needed
* Cleanup routine avoids memory leaks
* Metrics for monitoring request flow

