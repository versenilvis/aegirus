package config

import "go.uber.org/zap/zapcore"

func (d DetectorConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt("threshold", d.BanThreshold)
	enc.AddDuration("ban_duration", d.BanDuration)
	enc.AddDuration("score_decay", d.ScoreDecayTime)
	enc.AddDuration("cleanup_time", d.CleanupTime)
	return nil
}

func (r RateLimiterConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt("burst", r.Burst)
	enc.AddFloat64("refill_rate", r.RefillRate)
	enc.AddDuration("ban_duration", r.BanDuration)
	enc.AddDuration("cleanup_interval", r.CleanupInterval)
	enc.AddDuration("client_ttl", r.ClientTTL)
	return nil
}

func (c AppConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("port", c.Port)

	if err := enc.AddObject("detector", c.Detector); err != nil {
		return err
	}
	return enc.AddObject("rate_limiter", c.RateLimiter)
}
