package config

import (
	"log"
	"os"
	"time"

	"anti-ddos/utils"
)

type AppConfig struct {
	Port        string
	Detector    DetectorConfig
	RateLimiter RateLimiterConfig
}

type DetectorConfig struct {
	BanThreshold   int
	BanDuration    time.Duration
	ScoreDecayTime time.Duration
	CleanupTime    time.Duration
	NoUserAgent    int
	CleanUpOldIPs  time.Duration
}

type RateLimiterConfig struct {
	Burst           int
	RefillRate      float64
	BanDuration     time.Duration
	CleanupInterval time.Duration
	ClientTTL       time.Duration
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	duration, err := time.ParseDuration(val)
	if err != nil {
		log.Printf("Warning: Invalid duration format for %s. Using default: %v", key, defaultVal)
		return defaultVal
	}
	return duration
}

func LoadConfig() *AppConfig {
	return &AppConfig{
		Port: utils.GetEnv("PORT", "8080"),

		Detector: DetectorConfig{
			BanThreshold: utils.GetEnvAsInt("DETECTOR_BAN_THRESHOLD", 10),
			NoUserAgent:  utils.GetEnvAsInt("DETECTOR_NO_USER_AGENT_PENALTY", 2),

			BanDuration:    getEnvAsDuration("DETECTOR_BAN_DURATION", 5*time.Minute),
			ScoreDecayTime: getEnvAsDuration("DETECTOR_SCORE_DECAY", 1*time.Minute),
			CleanupTime:    getEnvAsDuration("DETECTOR_CLEANUP_TIME", 5*time.Minute),
			CleanUpOldIPs:  getEnvAsDuration("DETECTOR_CLEANUP_OLD_IPS", 10*time.Minute),
		},

		RateLimiter: RateLimiterConfig{
			Burst:      utils.GetEnvAsInt("RATE_LIMIT_BURST", 50),
			RefillRate: utils.GetEnvAsFloat("RATE_LIMIT_REFILL_RATE", 10.0),

			BanDuration:     getEnvAsDuration("RATE_LIMIT_BAN_DURATION", 5*time.Minute),
			CleanupInterval: getEnvAsDuration("RATE_LIMIT_CLEANUP_INTERVAL", 1*time.Minute),
			ClientTTL:       getEnvAsDuration("RATE_LIMIT_CLIENT_TTL", 3*time.Minute),
		},
	}
}
