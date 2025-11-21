package tests

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charmbracelet/glamour"
)

var targets = map[string]string{
	"Local":   "http://localhost:8080/api/test",
	"Railway": "https://aegirus.up.railway.app/api/test",
	"Render":  "https://aegirus-anti-ddos-low0.onrender.com/api/test",
}

const TargetTimeout = 45 * time.Second

type TestConfig struct {
	Name        string
	Requests    int
	Concurrency int
	RandomIP    bool
	NoUserAgent bool
}

type TestResult struct {
	Target    string
	Status    string
	Duration  time.Duration
	RPS       float64
	Success   int64
	RateLimit int64
	Banned    int64
	Failed    int64
}

type ScenarioResult struct {
	Config  TestConfig
	Results []TestResult
}

// =================================================================
// TEST: RUN ALL AND PRINT REPORT
// =================================================================
func Test_Run_Full_Analysis(t *testing.T) {
	stdCfg := TestConfig{
		Name:        "STANDARD (Rate Limit Check)",
		Requests:    500,
		Concurrency: 20,
		RandomIP:    false,
		NoUserAgent: false,
	}

	ddosCfg := TestConfig{
		Name:        "DDoS SIMULATION (Detector Check)",
		Requests:    1000,
		Concurrency: 50,
		RandomIP:    true,
		NoUserAgent: true,
	}

	t.Log("\nSTARTING FULL SYSTEM ANALYSIS...")

	t.Logf("Phase 1: Running Standard Tests...")
	stdResults := runScenarioInternal(t, stdCfg)

	t.Logf("Phase 2: Running DDoS Tests...")
	ddosResults := runScenarioInternal(t, ddosCfg)

	printCombinedDashboard(t,
		ScenarioResult{Config: stdCfg, Results: stdResults},
		ScenarioResult{Config: ddosCfg, Results: ddosResults},
	)
}

// =================================================================
// TEST: DEBUG
// =================================================================
func Test_RunAll_Standard(t *testing.T) {
	cfg := TestConfig{Name: "Standard", Requests: 500, Concurrency: 20, RandomIP: false, NoUserAgent: false}
	results := runScenarioInternal(t, cfg)
	printSectionReport(t, ScenarioResult{Config: cfg, Results: results})
}

func Test_RunAll_DDoS(t *testing.T) {
	cfg := TestConfig{Name: "DDoS", Requests: 1000, Concurrency: 50, RandomIP: true, NoUserAgent: true}
	results := runScenarioInternal(t, cfg)
	printSectionReport(t, ScenarioResult{Config: cfg, Results: results})
}

// -----------------------------------------------------------------
// CORE LOGIC
// -----------------------------------------------------------------

func runScenarioInternal(t *testing.T, cfg TestConfig) []TestResult {
	var results []TestResult

	for name, url := range targets {
		res := executeStressTest(name, url, cfg)

		statusIcon := "✅"
		if res.Status == "Timed Out" {
			statusIcon = "⚠️"
		}
		t.Logf("   %s [%s] Done (%v)", statusIcon, name, res.Duration)

		results = append(results, res)
	}
	return results
}

func executeStressTest(name, url string, cfg TestConfig) TestResult {
	var success, limit, banned, failed int64

	ctx, cancel := context.WithTimeout(context.Background(), TargetTimeout)
	defer cancel()

	jobs := make(chan int, cfg.Requests)
	var wg sync.WaitGroup
	startTime := time.Now()

	for w := 0; w < cfg.Concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Set timeout for Client prevent worker from being stuck forever
			client := &http.Client{Timeout: 5 * time.Second}

			for {
				select {
				case <-ctx.Done():
					return
				case _, ok := <-jobs:
					if !ok {
						return
					}

					ipToSend := "45.14.71.11"
					if cfg.RandomIP {
						ipToSend = fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
					}

					req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
					req.Header.Set("X-Forwarded-For", ipToSend)
					if !cfg.NoUserAgent {
						req.Header.Set("User-Agent", "GoTest/1.0")
					} else {
						req.Header.Set("User-Agent", "")
					}

					resp, err := client.Do(req)
					if err != nil {
						atomic.AddInt64(&failed, 1)
						continue
					}

					switch resp.StatusCode {
					case 200:
						atomic.AddInt64(&success, 1)
					case 429:
						atomic.AddInt64(&limit, 1)
					case 403:
						atomic.AddInt64(&banned, 1)
					default:
						atomic.AddInt64(&failed, 1)
					}

					_ = resp.Body.Close()
				}
			}
		}()
	}

	go func() {
		for i := 0; i < cfg.Requests; i++ {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case jobs <- i:
			}
		}
		close(jobs)
	}()

	wg.Wait()
	duration := time.Since(startTime)
	status := "Completed"
	if ctx.Err() == context.DeadlineExceeded {
		status = "Timed Out"
	}

	totalProcessed := atomic.LoadInt64(&success) + atomic.LoadInt64(&limit) + atomic.LoadInt64(&banned) + atomic.LoadInt64(&failed)
	rps := 0.0
	if duration.Seconds() > 0 {
		rps = float64(totalProcessed) / duration.Seconds()
	}

	return TestResult{
		Target:    name,
		Status:    status,
		Duration:  duration,
		RPS:       rps,
		Success:   success,
		RateLimit: limit,
		Banned:    banned,
		Failed:    failed,
	}
}

func printCombinedDashboard(t *testing.T, scenarios ...ScenarioResult) {
	var sb strings.Builder

	sb.WriteString("# FULL SYSTEM REPORT\n\n")

	for _, sc := range scenarios {
		sb.WriteString(fmt.Sprintf("## %s\n", sc.Config.Name))
		sb.WriteString(fmt.Sprintf("> **Config:** Req: `%d` | Threads: `%d` | RandomIP: `%v` | NoUA: `%v`\n\n",
			sc.Config.Requests, sc.Config.Concurrency, sc.Config.RandomIP, sc.Config.NoUserAgent))

		sb.WriteString("| Target | Time | RPS | 🟢 200 | 🟡 429 | 🔴 403 | ❌ Err |\n")
		sb.WriteString("|:---|:---:|:---:|:---:|:---:|:---:|:---:|\n")

		total := float64(sc.Config.Requests)
		for _, r := range sc.Results {
			statusIcon := ""
			if r.Status == "Timed Out" {
				statusIcon = "⚠️ "
			}

			sb.WriteString(fmt.Sprintf("| **%s%s** | %s | %.0f | %d (%.0f%%) | %d (%.0f%%) | **%d (%.0f%%)** | %d |\n",
				statusIcon, r.Target,
				r.Duration.Round(time.Millisecond),
				r.RPS,
				r.Success, float64(r.Success)/total*100,
				r.RateLimit, float64(r.RateLimit)/total*100,
				r.Banned, float64(r.Banned)/total*100,
				r.Failed,
			))
		}
		sb.WriteString("\n")
	}

	out, err := glamour.Render(sb.String(), "dark")
	if err != nil {
		t.Log(sb.String())
	} else {
		t.Log("\n" + out)
	}
}

func printSectionReport(t *testing.T, sc ScenarioResult) {
	printCombinedDashboard(t, sc)
}
