# Module: Rate Limiter

The **Rate Limiter** module is the "Traffic Controller" of the Anti-DDoS system. It implements a high-performance **Token Bucket** algorithm to control the flow of incoming requests. While the Detector focuses on *quality* (bad behavior), the Rate Limiter focuses on *quantity* (volume control).

## Features

  * **Token Bucket Algorithm:** Uses `golang.org/x/time/rate` for smooth, burst-tolerant limiting.
  * **Per-Endpoint Granularity:** Limits are tracked based on a combination of `IP + Endpoint`. A user spamming `/api/login` won't necessarily block their access to `/api/home` (unless triggered by the Detector).
  * **Double-Checked Locking:** Optimized for high concurrency. Uses `RWMutex` (Read-Write Lock) to minimize bottlenecking during high traffic.
  * **Detector Integration:** Automatically triggers an **IP Ban** via the Detector module when a rate limit is exceeded, upgrading a temporary 429 error to a full 403 network ban.
  * **Automatic Cleanup:** Background garbage collection removes stale user records to prevent memory leaks.

-----

## Configuration

The module behavior is controlled via Environment Variables (`.env`) and constructor parameters.

<div align=center>

| Environment Variable | Default | Description |
| :--- | :--- | :--- |
| `RATE_LIMIT_CLEANUP_INTERVAL` | `60` (1 min) | How often the cleanup routine runs. |
| `RATE_LIMIT_CLIENT_TTL` | `180` (3 min) | How long a client record stays in memory after their last request. |

</div>

**Constructor Parameters (`NewRateLimiter`):**

  * **maxRequests:** The maximum "burst" size allowed.
  * **refillRate:** How many requests are allowed per second (RPS).
  * **blockSec:** Duration to ban the IP if the limit is exceeded.

-----

## 🛠️ Internal Logic

### 1\. The Key Strategy

The limiter identifies clients using a composite key:
`Key = ClientIP + "|" + EndpointPath`

  * **Benefit:** Prevents a bottleneck on one API from affecting others.
  * **Example:** Spamming `100 req/s` on `/api/download` will trigger the limit for that specific path.

### 2\. Optimization: Double-Checked Locking

To handle thousands of concurrent requests without slowing down the server, the module uses a two-step locking mechanism in `getLimiter()`:

1.  **Read Lock (`RLock`):** Allows multiple requests to read the map simultaneously. Extremely fast.
2.  **Write Lock (`Lock`):** Only used if the client does not exist yet. Before writing, it checks again to ensure another goroutine didn't write it first.

### 3\. The "Escalation" Protocol

This is the critical link between RateLimiter and Detector:

1.  Client sends too many requests.
2.  `limiter.Allow()` returns `false`.
3.  **Action A:** Return `429 Too Many Requests` to the client.
4.  **Action B:** Call `rl.detector.BanIP()`. This "escalates" the punishment from a simple "Please wait" to "You are now banned for X minutes".

-----

## Integration Guide

Initialize the RateLimiter **after** the Detector, as it depends on the Detector instance.

```go
func main() {
    // 1. Init Detector first
    detector := modules.NewDetector()
    go detector.Cleanup(ctx)

    // 2. Init RateLimiter
    // Config: 10 req/s, Burst 5, Ban for 60s if exceeded
    rl := modules.NewRateLimiter(10, 5, 5, 60, detector)

    // 3. Start RateLimiter Cleanup
    go rl.Cleanup(ctx)

    // 4. Register Middleware (AFTER Detector)
    // Detector (Police) -> RateLimiter (Traffic Light) -> Handler
    app.Use(detector.DetectorMiddleware())
    app.Use(rl.RateLimiterMiddleware())
}
```

-----

## Flowchart

```graph TD
    A[Request arrives at RateLimiter] --> B{Get Limiter Object};
    B --> C(RLock: Check Map);
    C -- Found --> D[Return Limiter];
    C -- Not Found --> E(Lock: Create New Limiter);
    E --> D;

    D --> F{limiter.Allow?};
    
    F -- YES (Has Token) --> G[Next Middleware c.Next()];
    
    F -- NO (Empty Bucket) --> H[Increment Metrics];
    H --> I[Trigger Detector.BanIP()];
    I --> J[Log Warning];
    J --> K[Return 429 Too Many Requests];
```

-----

## 📖 API Reference

### `NewRateLimiter(...)`

Creates a new RateLimiter instance.

  * **Params:** `maxRequests` (Burst), `refillRate` (RPS), `burst` (Token capacity), `blockSec` (Ban duration), `d` (Detector instance).

### `RateLimiterMiddleware() fiber.Handler`

The middleware function to be used in Fiber `app.Use()`.

  * It extracts the IP.
  * Retrieves the specific token bucket for that IP+Path.
  * Consumes a token.
  * If no tokens left -\> Ban IP & Return 429.

### `Cleanup(ctx context.Context)`

Background worker. It iterates through the `clients` map and removes entries that haven't been active for `RATE_LIMIT_CLIENT_TTL` seconds. This ensures the server RAM doesn't fill up with one-time visitors.

### `Metrics() (int, int)`

Returns current statistics:

1.  `activeClients`: Number of IPs currently being tracked in memory.
2.  `blockedRequests`: Total number of requests rejected since startup.
