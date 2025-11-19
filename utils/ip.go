package utils

import (
	"net"
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
	if forwarded := c.Get("Forwarded"); forwarded != "" {
		if ip := ParseForwardedHeader(forwarded); ip != "" {
			return ip
		}
	}

	headers := []string{
		"CF-Connecting-IP",
		"True-Client-IP",
		"X-Real-IP",
		"X-Forwarded-For",
		"X-Client-IP",
	}
	for _, h := range headers {
		if ip := c.Get(h); ip != "" {
			ips := strings.Split(ip, ",")
			if len(ips) > 0 {
				clientIP := strings.TrimSpace(ips[0])
				if net.ParseIP(clientIP) != nil {
					return clientIP
				}
			}
		}
	}
	ip := c.IP()
	if ip != "" && net.ParseIP(ip) != nil {
		return ip
	}
	return ""
}
