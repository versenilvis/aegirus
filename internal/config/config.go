// Package config provides the configuration for the application
package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"aegirus/internal/modules/upstream"
	"aegirus/internal/utils"
)

type AppConfig struct {
	Port            string            `json:"port"`
	AppEnv          string            `json:"app_env"`
	TrustedProxies  string            `json:"trusted_proxies"`
	LogSamplingRate int               `json:"log_sampling_rate"`
	Detector        DetectorConfig    `json:"detector"`
	RateLimiter     RateLimiterConfig `json:"rate_limiter"`
	Redis           RedisConfig       `json:"redis_config"`
	Upstreams       []upstream.Target `json:"upstreams"`
}

type DetectorConfig struct {
	BanThreshold   int      `json:"ban_threshold"`
	BanDuration    Duration `json:"ban_duration"`
	ScoreDecayTime Duration `json:"score_decay_time"`
	CleanupTime    Duration `json:"cleanup_time"`
	NoUserAgent    int      `json:"no_user_agent"`
	CleanUpOldIPs  Duration `json:"cleanup_old_ips"`
}

type RateLimiterConfig struct {
	Rate           int      `json:"rate"`
	WindowDuration Duration `json:"window_duration"`

	BanDuration     Duration `json:"ban_duration"`
	CleanupInterval Duration `json:"cleanup_interval"`
	ClientTTL       Duration `json:"client_ttl"`
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"-"` // sensitive information please use "-"
	DB       int    `json:"db"`
}

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Duration(d).String() + `"`), nil
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

func parseUpstreamTargets(envVal string) []upstream.Target {
	if envVal == "" {
		return nil
	}

	var targets []upstream.Target
	parts := strings.Split(envVal, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Format: "addr|weight" or just "addr" (default weight 1)
		// Use pipe separator to avoid conflicts with URL colons (e.g., http://host:port)
		pipeIdx := strings.LastIndex(part, "|")
		if pipeIdx == -1 {
			// No pipe found, use default weight of 1
			targets = append(targets, upstream.Target{
				Addr:   part,
				Weight: 1,
			})
			continue
		}

		possibleWeight := part[pipeIdx+1:]
		if weight, err := strconv.Atoi(possibleWeight); err == nil && weight > 0 {
			addr := part[:pipeIdx]
			targets = append(targets, upstream.Target{
				Addr:   addr,
				Weight: weight,
			})
		} else {
			targets = append(targets, upstream.Target{
				Addr:   part,
				Weight: 1,
			})
		}
	}

	return targets
}

func LoadConfig() *AppConfig {
	cfg := &AppConfig{
		Port:           utils.GetEnv("PORT", "8080"),
		AppEnv:         utils.GetEnv("APP_ENV", "development"),
		TrustedProxies: utils.GetEnv("TRUSTED_PROXIES", ""),

		LogSamplingRate: utils.GetEnvAsInt("LOG_SAMPLING_RATE", 10),

		Detector: DetectorConfig{
			BanThreshold:   utils.GetEnvAsInt("DETECTOR_BAN_THRESHOLD", 10),
			NoUserAgent:    utils.GetEnvAsInt("DETECTOR_NO_USER_AGENT_PENALTY", 2),
			BanDuration:    Duration(getEnvAsDuration("DETECTOR_BAN_DURATION", 5*time.Minute)),
			ScoreDecayTime: Duration(getEnvAsDuration("DETECTOR_SCORE_DECAY", 1*time.Minute)),
			CleanupTime:    Duration(getEnvAsDuration("DETECTOR_CLEANUP_TIME", 5*time.Minute)),
			CleanUpOldIPs:  Duration(getEnvAsDuration("DETECTOR_CLEANUP_OLD_IPS", 10*time.Minute)),
		},

		RateLimiter: RateLimiterConfig{
			Rate:            utils.GetEnvAsInt("RATE_LIMIT_RATE", 10),
			WindowDuration:  Duration(getEnvAsDuration("RATE_LIMIT_WINDOW_DURATION", 1*time.Minute)),
			BanDuration:     Duration(getEnvAsDuration("RATE_LIMIT_BAN_DURATION", 5*time.Minute)),
			CleanupInterval: Duration(getEnvAsDuration("RATE_LIMIT_CLEANUP_INTERVAL", 1*time.Minute)),
			ClientTTL:       Duration(getEnvAsDuration("RATE_LIMIT_CLIENT_TTL", 3*time.Minute)),
		},

		Redis: RedisConfig{
			Addr:     utils.GetEnv("REDIS_ADDR", ""),
			Password: utils.GetEnv("REDIS_PASS", ""),
			DB:       utils.GetEnvAsInt("REDIS_DB", 0),
		},

		Upstreams: parseUpstreamTargets(utils.GetEnv("UPSTREAM_TARGETS", "")),
	}

	return cfg
}
