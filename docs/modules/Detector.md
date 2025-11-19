# Module: Detector & IP Blocker

The **Detector** module is a heuristic-based security middleware designed for the Go Fiber framework. It acts as the "Police" of the Anti-DDoS system, monitoring client behavior to detect anomalies, bot scanning, and suspicious activities. Unlike a standard Rate Limiter (which counts requests), the Detector analyzes **how** the client is behaving.

## Features

  * **Score-Based System:** Assigns penalty points to IPs based on suspicious behavior.
  * **Automatic Banning:** Automatically jails an IP for a configurable duration if the score exceeds the threshold.
  * **Score Decay (Forgiveness):** Automatically resets the score of "well-behaved" clients after a period of inactivity, preventing false positives over time.
  * **Manual Ban Trigger:** exposure of `BanIP` method allows other modules (like RateLimiter or Login) to ban IPs explicitly.
  * **Memory Management:** Includes a background cleanup routine to prevent memory leaks by removing stale IP records.
  * **Concurrency Safe:** Uses Mutex locking to handle concurrent requests safely.

-----

## Configuration

The module behavior can be tuned using Environment Variables (`.env`).

<div align="center">

| Environment Variable | Default | Description |
| :--- | :--- | :--- |
| `DETECTOR_BAN_THRESHOLD` | `10` | The max score allowed before an IP is banned. |
| `DETECTOR_BAN_DURATION_SEC`| `300` (5 min) | How long (in seconds) an IP remains banned. |
| `DETECTOR_SCORE_DECAY_SEC` | `60` (1 min) | Time of inactivity required to reset an IP's score to 0. |

</div>

-----

## Detection Logic (Rules)

The detector inspects every request and applies the following rules:

1.  **High-Frequency Spam (Rule A):**

      * **Condition:** If the time gap between two requests is **\< 50ms**.
      * **Penalty:** `+3 Points`.
      * **Reason:** Human users rarely click that fast; usually indicates a script.

2.  **Missing User-Agent (Rule B):**

      * **Condition:** If the request header `User-Agent` is empty.
      * **Penalty:** `+2 Points`.
      * **Reason:** Amateur bots often forget to set this header.

3.  **Vulnerability Scanning (Rule C - Optional):**

      * **Condition:** Accessing paths like `.env`, `wp-admin`, `.git`.
      * **Penalty:** `+10 Points` (Immediate Ban).

-----

## Integration Guide

### 1\. Initialization & Cleanup

In your `main.go`, initialize the detector and start the cleanup routine.

```go
package main

import (
    "context"
    "anti-ddos/modules"
    "github.com/gofiber/fiber/v3"
)

func main() {
    app := fiber.New()
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // 1. Initialize Detector
    detector := modules.NewDetector()

    // 2. Start Memory Cleanup (Runs in background)
    go detector.Cleanup(ctx)

    // 3. Register Middleware (Should be the FIRST middleware)
    app.Use(detector.DetectorMiddleware())

    app.Listen(":3000")
}
```

### 2\. Interaction with RateLimiter

The Detector is designed to work in tandem with a RateLimiter. If the RateLimiter detects a volume flood, it can trigger the Detector to ban the IP.

```go
// Inside RateLimiter logic
if !limiter.Allow() {
    // Rate limit exceeded -> Call Detector to BAN the IP for 5 minutes
    detector.BanIP(clientIP, 5 * time.Minute)
    return c.Status(429).JSON(...)
}
```

-----

## API Reference

### `NewDetector() *Detector`

Creates a new instance of the Detector with an empty memory map.

### `DetectorMiddleware() fiber.Handler`

The core middleware function.

  * **Flow:**
<div align="center">
  <img width="1024" height="1024" alt="image" src="https://github.com/user-attachments/assets/280f3c2a-0fde-4cdf-b791-bd70fc03d277" />
</div>

### `Cleanup(ctx context.Context)`

A blocking function (should run in a goroutine) that wakes up every `5 minutes`.

  * It removes IPs that are **not banned** AND **haven't been seen** for over 10 minutes.
  * This prevents the RAM from filling up with millions of old IP addresses.

### `BanIP(ip string, duration time.Duration)`

Thread-safe method to manually ban an IP.

  * **Usage:** Can be called by RateLimiter, Login handlers (after N failed attempts), or Admin panels.
  * **Effect:** Sets the IP's `BannedUntil` to `Now + Duration` and sets Score to Max.

-----

## Important Notes

1.  **Order Matters:** The `DetectorMiddleware` should always be placed **before** the RateLimiter and Business Logic. This ensures banned IPs are rejected immediately without consuming further resources.
2.  **Deadlock Prevention:** The code carefully manages `mu.Lock()` and `mu.Unlock()`. Ensure that every `return` statement is preceded by an `Unlock()` call.
3.  **Retry-After Header:** When an IP is banned, the server responds with a `Retry-After` header. This is standard HTTP protocol behavior to inform legitimate clients (or good bots) when they can return.
