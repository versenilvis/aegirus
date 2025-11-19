package tests

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	TargetURL     = "http://localhost:8080/api/test"
	TotalRequests = 200
	Delay         = 100 * time.Millisecond
	RandomizeIP   = false     // set this to true if you want to test from multiple IPs
	FixedIP       = "1.2.3.4" // any ip you want
)

func TestAntiDDoS_Stress(t *testing.T) {
	var (
		countSuccess int32
		countRateLim int32
		countBanned  int32
		countError   int32
	)

	t.Logf("STARTING LOAD TEST (Standard Go Test)")
	t.Logf("Target: %s | Total: %d", TargetURL, TotalRequests)

	var wg sync.WaitGroup
	startTime := time.Now()

	for i := range TotalRequests {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			status := sendRequest(t, id, TargetURL)

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

		// if Delay > 0 {
		// 	time.Sleep(Delay)
		// }
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("--------------------------------------------------")
	t.Logf("✅ TEST FINISHED in %s", duration)
	t.Logf("📊 SUMMARY:")
	t.Logf("   🟢 Success (200):     %d", atomic.LoadInt32(&countSuccess))
	t.Logf("   🟡 Rate Limit (429):  %d", atomic.LoadInt32(&countRateLim))
	t.Logf("   🔴 Banned (403):      %d", atomic.LoadInt32(&countBanned))
	t.Logf("   ❌ Errors:            %d", atomic.LoadInt32(&countError))
	t.Logf("--------------------------------------------------")
}

func sendRequest(t *testing.T, id int, url string) int {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Logf("[Worker %d] Init Error: %v", id, err)
		return 0
	}
	ipToSend := FixedIP
	if RandomizeIP {
		ipToSend = fmt.Sprintf("10.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255))
	}
	req.Header.Set("X-Forwarded-For", ipToSend)
	req.Header.Set("User-Agent", "GoTest-Client/1.0")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("[Worker %d] Network Error: %v", id, err)
		return 0
	}
	defer resp.Body.Close()

	// t.Logf("[Worker %d] IP: %s -> Status %d", id, ipToSend, resp.StatusCode)

	return resp.StatusCode
}
