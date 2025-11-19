// Package config - Config
package config

import (
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
}

type RateLimiterConfig struct {
	Burst           int
	RefillRate      float64
	BanDuration     time.Duration
	CleanupInterval time.Duration
	ClientTTL       time.Duration
}

func LoadConfig() *AppConfig {
	return &AppConfig{
		Port: utils.GetEnv("PORT", "8080"),

		Detector: DetectorConfig{
			BanThreshold:   utils.GetEnvAsInt("DETECTOR_BAN_THRESHOLD", 10),
			BanDuration:    time.Duration(utils.GetEnvAsInt("DETECTOR_BAN_DURATION_SEC", 300)) * time.Second,
			ScoreDecayTime: time.Duration(utils.GetEnvAsInt("DETECTOR_SCORE_DECAY_SEC", 60)) * time.Second,
			CleanupTime:    5 * time.Minute,
		},

		RateLimiter: RateLimiterConfig{
			Burst:           utils.GetEnvAsInt("RATE_LIMIT_BURST", 50),
			RefillRate:      utils.GetEnvAsFloat("RATE_LIMIT_REFILL_RATE", 10.0),
			BanDuration:     time.Duration(utils.GetEnvAsInt("RATE_LIMIT_BAN_DURATION_SEC", 300)) * time.Second,
			CleanupInterval: time.Duration(utils.GetEnvAsInt("RATE_LIMIT_CLEANUP_INTERVAL", 60)) * time.Second,
			ClientTTL:       time.Duration(utils.GetEnvAsInt("RATE_LIMIT_CLIENT_TTL", 180)) * time.Second,
		},
	}
}
