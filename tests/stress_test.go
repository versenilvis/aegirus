// Package tests contains the load test code for the anti-ddos project
package tests

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charmbracelet/glamour"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func formatAndRenderReport(t *testing.T, totalRequests int, duration time.Duration, rps float64, success, limit, banned, failure int32) string {
	var sb strings.Builder

	sb.WriteString("# Load Test Report\n\n")
	sb.WriteString("## Configuration\n")
	sb.WriteString(fmt.Sprintf("- **Target URL:** %s\n", getEnv("TARGET_URL", "http://localhost:8080/api/test")))
	sb.WriteString(fmt.Sprintf("- **Total Requests:** %d\n", totalRequests))
	sb.WriteString(fmt.Sprintf("- **Concurrency:** %s\n\n", getEnv("CONCURRENCY", "20")))

	sb.WriteString("## Performance\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|:---|:---|\n")
	sb.WriteString(fmt.Sprintf("| **Duration** | %s |\n", duration.Round(time.Millisecond)))
	sb.WriteString(fmt.Sprintf("| **RPS** | %.2f req/s |\n\n", rps))

	sb.WriteString("## Results Summary\n")
	sb.WriteString("| Status | Count | Percentage |\n")
	sb.WriteString("|:---|:---:|:---:|\n")

	totalSent := float64(totalRequests)

	sb.WriteString(fmt.Sprintf("| Success (200) | %d | %.2f%% |\n", success, float64(success)/totalSent*100))
	sb.WriteString(fmt.Sprintf("| Rate Limit (429) | %d | %.2f%% |\n", limit, float64(limit)/totalSent*100))
	sb.WriteString(fmt.Sprintf("| Banned (403) | %d | %.2f%% |\n", banned, float64(banned)/totalSent*100))
	sb.WriteString(fmt.Sprintf("| Failed/Error | %d | %.2f%% |\n", failure, float64(failure)/totalSent*100))

	out, err := glamour.Render(sb.String(), "dark")
	if err != nil {
		t.Logf("Glamour Render Error: %v", err)
		return sb.String()
	}
	return out
}

func TestAntiDDoS_Stress(t *testing.T) {
	targetURL := getEnv("TARGET_URL", "http://localhost:8080/api/test")

	totalRequests, _ := strconv.Atoi(getEnv("TOTAL_REQUESTS", "200"))
	concurrency, _ := strconv.Atoi(getEnv("CONCURRENCY", "20")) // Số luồng chạy song song

	randomizeIP := getEnv("RANDOM_IP", "false") == "true"
	fixedIP := "45.14.71.11"

	var configSB strings.Builder
	configSB.WriteString("# 🚀 Starting Load Test\n\n")
	configSB.WriteString("## Test Configuration\n")
	configSB.WriteString("| Setting | Value |\n")
	configSB.WriteString("|:---|:---|\n")
	configSB.WriteString(fmt.Sprintf("| Target URL | **%s** |\n", targetURL))
	configSB.WriteString(fmt.Sprintf("| Total Requests | %d |\n", totalRequests))
	configSB.WriteString(fmt.Sprintf("| Concurrency (Workers) | %d |\n", concurrency))
	configSB.WriteString(fmt.Sprintf("| Random IP Spoofing | **%v** |\n", randomizeIP))
	configSB.WriteString(fmt.Sprintf("| Fixed Test IP | %s |\n", fixedIP))

	configOutput, err := glamour.Render(configSB.String(), "dark")
	if err != nil {
		t.Logf("Configuration: Target: %s, Requests: %d, Concurrency: %d", targetURL, totalRequests, concurrency)
	} else {
		t.Logf("\n%s", configOutput)
	}

	var (
		countSuccess int32
		countRateLim int32
		countBanned  int32
		countError   int32
	)

	var wg sync.WaitGroup
	startTime := time.Now()

	sem := make(chan struct{}, concurrency)

	for i := range totalRequests {
		wg.Add(1)
		sem <- struct{}{}

		go func(id int) {
			defer wg.Done()
			defer func() { <-sem }()

			status := sendRequest(t, id, targetURL, fixedIP, randomizeIP)

			switch status {
			case 200:
				atomic.AddInt32(&countSuccess, 1)
			case 429:
				atomic.AddInt32(&countRateLim, 1)
			case 403:
				atomic.AddInt32(&countBanned, 1)
			default:
				atomic.AddInt32(&countError, 1)
			}
		}(i + 1)
	}

	wg.Wait()
	duration := time.Since(startTime)

	rps := 0.0
	if duration.Seconds() > 0 {
		rps = float64(totalRequests) / duration.Seconds()
	}

	report := formatAndRenderReport(t,
		totalRequests,
		duration,
		rps,
		atomic.LoadInt32(&countSuccess),
		atomic.LoadInt32(&countRateLim),
		atomic.LoadInt32(&countBanned),
		atomic.LoadInt32(&countError),
	)

	t.Logf("\n%s", report)
}

func sendRequest(t *testing.T, id int, url string, fixedIP string, random bool) int {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Logf("[Worker %d] Init Error: %v", id, err)
		return 0
	}

	ipToSend := fixedIP
	if random {
		ipToSend = fmt.Sprintf("%d.%d.%d.%d",
			rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
	}

	req.Header.Set("X-Forwarded-For", ipToSend)
	req.Header.Set("User-Agent", "GoTest-Client/1.0")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	return resp.StatusCode
}
