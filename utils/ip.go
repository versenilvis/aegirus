package utils

import (
	"net"
	"strings"
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
