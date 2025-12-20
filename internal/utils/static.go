package utils

import "strings"

func IsStaticAsset(path string) bool {
	staticExtensions := []string{".css", ".js", ".jpg", ".jpeg", ".png", ".gif", ".ico", ".svg", ".woff", ".woff2", ".ttf"}
	for _, ext := range staticExtensions {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			return true
		}
	}
	return false
}
