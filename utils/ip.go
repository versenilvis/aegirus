package utils

import (
	"net"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// ------------------- IP extraction -------------------

func ParseForwardedHeader(forwarded string) string {
	entries := strings.SplitSeq(forwarded, ",")
	for entry := range entries {
		entry = strings.TrimSpace(entry)
		if !strings.HasPrefix(entry, "for=") {
			continue
		}

		forPart, _ := strings.CutPrefix(entry, "for=")
		forPart = strings.Trim(forPart, `"`)

		if i := strings.IndexByte(forPart, ';'); i != -1 {
			forPart = forPart[:i]
		}

		ip := strings.TrimSpace(forPart)
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	return ""
}

func GetClientIP(c fiber.Ctx) string {
	// WARN: REMEMBER TO REMOVE TRUSTED_PROXIES VARIABLE IN YOUR .env IF YOU ARE USING YOUR VPS/LOCAL MACHINE,
	// BUT IF YOU ARE USING RAILWAY OR NGINX,... SET IT TO THE IP THAT YOUR SERVICE IS RUNNING ON
	trustedProxies := os.Getenv("TRUSTED_PROXIES")

	if trustedProxies != "" {
		if xff := c.Get("X-Forwarded-For"); xff != "" {
			ips := strings.Split(xff, ",")
			return strings.TrimSpace(ips[0])
		}
		// (Optional) For cloudflare
		// if cf := c.Get("CF-Connecting-IP"); cf != "" { return cf }
	}

	return c.IP()
}
