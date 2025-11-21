REQUESTS=500
WORKERS=20
DDoS_REQUESTS=1000
DDoS_WORKERS=50

URL_RAILWAY_OLD=https://ahihi.up.railway.app/api/test
URL_RAILWAY_NEW=aegirus-anti-ddos-production.up.railway.app/api/test
URL_LOCAL=http://localhost:8080/api/test
URL_RENDER=https://aegirus-anti-ddos-low0.onrender.com/api/test


# =========================================================================
# === 1. STANDARD RATE LIMIT TEST - 1 IP ===
# =========================================================================

t-local:
	TARGET_URL=$(URL_LOCAL) TOTAL_REQUESTS=$(REQUESTS) CONCURRENCY=$(WORKERS) go test -v ./tests/...

# feat bracnh
t-railway-new:
	TARGET_URL=$(URL_RAILWAY_NEW) TOTAL_REQUESTS=$(REQUESTS) CONCURRENCY=$(WORKERS) go test -v ./tests/...

# main bracnh
t-railway-old:
	TARGET_URL=$(URL_RAILWAY_OLD) TOTAL_REQUESTS=$(REQUESTS) CONCURRENCY=$(WORKERS) go test -v ./tests/...

t-render:
	TARGET_URL=$(URL_RENDER) TOTAL_REQUESTS=$(REQUESTS) CONCURRENCY=$(WORKERS) go test -v ./tests/...

#test standard
t-std: t-local t-railway-new t-railway-old t-render
.PHONY: t-std


# =========================================================================
# === 2. DDOS TEST (DETECTOR TEST - RANDOM IP) ===
# =========================================================================

t-ddos-railway-new:
	TARGET_URL=$(URL_RAILWAY_NEW) TOTAL_REQUESTS=$(DDoS_REQUESTS) CONCURRENCY=$(DDoS_WORKERS) RANDOM_IP=true go test -v ./tests/...

t-ddos-railway-old:
	TARGET_URL=$(URL_RAILWAY_OLD) TOTAL_REQUESTS=$(DDoS_REQUESTS) CONCURRENCY=$(DDoS_WORKERS) RANDOM_IP=true go test -v ./tests/...

t-ddos-render:
	TARGET_URL=$(URL_RENDER) TOTAL_REQUESTS=$(DDoS_REQUESTS) CONCURRENCY=$(DDoS_WORKERS) RANDOM_IP=true go test -v ./tests/...

t-ddos-local:
	TARGET_URL=$(URL_LOCAL) TOTAL_REQUESTS=$(DDoS_REQUESTS) CONCURRENCY=$(DDoS_WORKERS) RANDOM_IP=true go test -v ./tests/...

t-ddos: t-ddos-local t-ddos-railway-new t-ddos-railway-old t-ddos-render
.PHONY: t-ddos


# =========================================================================
# === 3. ALL TEST ===
# =========================================================================

t-all: t-std t-ddos
.PHONY: t-all

default: t-local
