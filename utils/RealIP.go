package utils

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// ------------------- Data Definitions -------------------
var privateAndLocalRanges = []net.IPNet{
	mustParseCIDR("10.0.0.0/8"),     // RFC1918 (Private A)
	mustParseCIDR("172.16.0.0/12"),  // RFC1918 (Private B - Docker)
	mustParseCIDR("192.168.0.0/16"), // RFC1918 (Private C - LAN)
	mustParseCIDR("127.0.0.0/8"),    // Loopback
	mustParseCIDR("0.0.0.0/8"),      // Current network
	mustParseCIDR("169.254.0.0/16"), // Link-local
	mustParseCIDR("100.64.0.0/10"),  // CGNAT (Carrier-Grade NAT)
	mustParseCIDR("::1/128"),        // IPv6 Loopback
	mustParseCIDR("fc00::/7"),       // IPv6 Unique Local
	mustParseCIDR("fe80::/10"),      // IPv6 Link Local
}

// ------------------- IP extraction -------------------

func goodIPAddr(ipStr string) *net.IPAddr {
	ipAddr, err := ParseIPAddr(ipStr)
	if err != nil {
		return nil
	}

	if ipAddr.IP.IsUnspecified() {
		return nil
	}

	return &ipAddr
}

func ParseIPAddr(ipStr string) (net.IPAddr, error) {
	host, _, err := net.SplitHostPort(ipStr)
	if err == nil {
		ipStr = host
	}
	ipStr = trimMatchedEnds(ipStr, "[]")
	ipStr, zone := SplitHostZone(ipStr)

	res := net.IPAddr{
		IP:   net.ParseIP(ipStr),
		Zone: zone,
	}

	if res.IP == nil {
		return net.IPAddr{}, fmt.Errorf("net.ParseIP failed")
	}

	return res, nil
}

func isIPContainedInRanges(ip net.IP, ranges []net.IPNet) bool {
	for _, r := range ranges {
		if r.Contains(ip) {
			return true
		}
	}
	return false
}

func isPrivateOrLocal(ip net.IP) bool {
	return isIPContainedInRanges(ip, privateAndLocalRanges)
}

// mustParseCIDR: a helper để parse static CIDR
func mustParseCIDR(s string) net.IPNet {
	_, ipNet, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return *ipNet
}

// SplitHostZone splits IPv6 zone
func SplitHostZone(s string) (host, zone string) {
	if i := strings.LastIndexByte(s, '%'); i > 0 {
		host, zone = s[:i], s[i+1:]
	} else {
		host = s
	}
	return
}

// trimMatchedEnds cuts off the matched ends (e.g. the endings of IPv6)
func trimMatchedEnds(s string, chars string) string {
	if len(chars) != 1 && len(chars) != 2 {
		panic("trimMatchedEnds chars must be length 1 or 2")
	}

	first, last := chars[0], chars[0]
	if len(chars) > 1 {
		last = chars[1]
	}

	if len(s) < 2 {
		return s
	}

	if s[0] != first {
		return s
	}

	if s[len(s)-1] != last {
		return s
	}

	return s[1 : len(s)-1]
}

var trustedProxies = os.Getenv("TRUSTED_PROXIES") == "true"

func GetRealIP(c fiber.Ctx) string {
	// WARN: REMEMBER TO REMOVE TRUSTED_PROXIES VARIABLE IN YOUR .env IF YOU ARE USING YOUR VPS/LOCAL MACHINE,
	// BUT IF YOU ARE USING RAILWAY OR NGINX,... SET IT TO TRUE IF YOUR SERVICE IS RUNNING
	if !trustedProxies {
		return c.IP()
	}

	if cf := c.Get("CF-Connecting-IP"); cf != "" {
		ipAddr := goodIPAddr(cf)
		if ipAddr != nil {
			return ipAddr.String()
		}
	}

	xff := c.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")

		// Right to left strategy, instead of taking the last ip (last), we take the first public ip
		for i := len(ips) - 1; i >= 0; i-- {
			ipStr := strings.TrimSpace(ips[i])
			ipAddr := goodIPAddr(ipStr)

			if ipAddr != nil && !isPrivateOrLocal(ipAddr.IP) {
				return ipAddr.String()
			}
		}
	}

	return c.IP()
}
