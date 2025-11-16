                      ┌───────────────────┐
                      │   Input Layer     │
                      │ (nhận request)    │
                      └─────────┬─────────┘
                                │
                                v
                      ┌───────────────────┐
                      │ Rate Control Layer │
                      │ (Token Bucket,     │
                      │ Sliding Window)    │
                      └─────────┬─────────┘
                                │
                                v
                      ┌───────────────────┐
                      │ Behavior Analyst  │
                      │ (phát hiện mẫu    │
                      │ bất thường)       │
                      └─────────┬─────────┘
                                │
                                v
                      ┌───────────────────┐
                      │  Decision Engine  │
                      │ (allow / throttle │
                      │ / block / blacklist)│
                      └─────────┬─────────┘
                                │
               ┌────────────────┴────────────────┐
               │                                 │
               v                                 v
       ┌───────────────┐                 ┌───────────────┐
       │ Storage Layer  │                 │ Interface/API │
       │ (bucket, IP    │                 │ (Check,       │
       │ blacklist, log)│                 │ UpdateConfig, │
       └───────────────┘                 │ GetStats)     │
                                         └───────────────┘
                                │
                                v
                       ┌───────────────────┐
                       │     Application    │
                       │    / Web Server    │
                       └───────────────────┘

