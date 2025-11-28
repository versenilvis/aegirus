// Package config provides the configuration for the application
package config

import (
	"log"
	"os"
	"time"

	"aegirus/internal/utils"
)

type AppConfig struct {
	Port           string             `json:"port"`
	AppEnv         string             `json:"app_env"`
	TrustedProxies string             `json:"trusted_proxies"`
	Detector       DetectorConfig     `json:"detector"`
	RateLimiter    RateLimiterConfig  `json:"rate_limiter"`
	BotChallenge   BotChallengeConfig `json:"bot_challenge"`
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
	Burst           int      `json:"burst"`
	RefillRate      float64  `json:"refill_rate"`
	BanDuration     Duration `json:"ban_duration"`
	CleanupInterval Duration `json:"cleanup_interval"`
	ClientTTL       Duration `json:"client_ttl"`
}

type BotChallengeConfig struct {
	Enabled             bool     `json:"enabled"`
	SecretKey           string   `json:"-"` // sensitive information please use "-"
	CookieName          string   `json:"cookie_name"`
	CookieTTL           Duration `json:"cookie_ttl"`
	KeyRotationInterval Duration `json:"key_rotation_interval"`
	Difficulty          int      `json:"difficulty"`
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

func LoadConfig() *AppConfig {
	cfg := &AppConfig{
		Port:           utils.GetEnv("PORT", "8080"),
		AppEnv:         utils.GetEnv("APP_ENV", "development"),
		TrustedProxies: utils.GetEnv("TRUSTED_PROXIES", ""),

		Detector: DetectorConfig{
			BanThreshold:   utils.GetEnvAsInt("DETECTOR_BAN_THRESHOLD", 10),
			NoUserAgent:    utils.GetEnvAsInt("DETECTOR_NO_USER_AGENT_PENALTY", 2),
			BanDuration:    Duration(getEnvAsDuration("DETECTOR_BAN_DURATION", 5*time.Minute)),
			ScoreDecayTime: Duration(getEnvAsDuration("DETECTOR_SCORE_DECAY", 1*time.Minute)),
			CleanupTime:    Duration(getEnvAsDuration("DETECTOR_CLEANUP_TIME", 5*time.Minute)),
			CleanUpOldIPs:  Duration(getEnvAsDuration("DETECTOR_CLEANUP_OLD_IPS", 10*time.Minute)),
		},

		RateLimiter: RateLimiterConfig{
			Burst:           utils.GetEnvAsInt("RATE_LIMIT_BURST", 50),
			RefillRate:      utils.GetEnvAsFloat("RATE_LIMIT_REFILL_RATE", 10.0),
			BanDuration:     Duration(getEnvAsDuration("RATE_LIMIT_BAN_DURATION", 5*time.Minute)),
			CleanupInterval: Duration(getEnvAsDuration("RATE_LIMIT_CLEANUP_INTERVAL", 1*time.Minute)),
			ClientTTL:       Duration(getEnvAsDuration("RATE_LIMIT_CLIENT_TTL", 3*time.Minute)),
		},

		BotChallenge: BotChallengeConfig{
			Enabled:    utils.GetEnvAsBool("BOT_CHALLENGE_ENABLED", true),
			SecretKey:  utils.GetEnv("BOT_CHALLENGE_SECRET_KEY", ""),
			CookieName: utils.GetEnv("BOT_CHALLENGE_COOKIE_NAME", "bot_challenge"),
			CookieTTL:  Duration(getEnvAsDuration("BOT_CHALLENGE_COOKIE_TTL", 5*time.Minute)),

			KeyRotationInterval: Duration(getEnvAsDuration("BOT_CHALLENGE_KEY_ROTATION_INTERVAL", 30*time.Minute)),
			Difficulty:          utils.GetEnvAsInt("BOT_CHALLENGE_DIFFICULTY", 3),
		},
	}

	// Validate configuration
	if cfg.BotChallenge.Enabled && cfg.BotChallenge.SecretKey == "" {
		log.Fatal("CRITICAL ERROR: BOT_CHALLENGE_ENABLED is true but BOT_CHALLENGE_SECRET_KEY is empty. Please set a secure secret key.")
	}

	return cfg
}
