package vercelcli

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const defaultTimeout = 20 * time.Second

type Snapshot struct {
	Date      string
	AmountUSD float64
	Metadata  map[string]any
}

type Collector struct {
	NowFn func() time.Time
}

func NewCollector() *Collector {
	return &Collector{NowFn: time.Now}
}

func (c *Collector) Collect(ctx context.Context) ([]Snapshot, error) {
	if c.NowFn == nil {
		c.NowFn = time.Now
	}
	if _, err := exec.LookPath("vercel"); err != nil {
		return nil, fmt.Errorf("vercel cli not found on PATH")
	}
	if _, stderr, err := runVercel(ctx, "--version"); err != nil {
		if stderr == "" {
			stderr = err.Error()
		}
		return nil, fmt.Errorf("vercel cli unavailable: %s", stderr)
	}
	if _, stderr, err := runVercel(ctx, "whoami"); err != nil {
		if stderr == "" {
			stderr = "run `vercel login` and retry"
		}
		return nil, fmt.Errorf("vercel session invalid: %s", strings.TrimSpace(stderr))
	}

	candidates := [][]string{
		{"usage", "--json"},
		{"billing", "usage", "--json"},
		{"billing", "ls", "--json"},
	}
	for _, args := range candidates {
		stdout, _, err := runVercel(ctx, args...)
		if err != nil {
			continue
		}
		snaps, parseErr := parseSnapshots(stdout, c.NowFn().UTC())
		if parseErr == nil && len(snaps) > 0 {
			return snaps, nil
		}
	}
	return nil, fmt.Errorf("unable to collect cost data from vercel cli")
}

func runVercel(ctx context.Context, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	path, err := exec.LookPath("vercel")
	if err != nil {
		return "", "", err
	}
	cmd := exec.CommandContext(ctx, path, args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return strings.TrimSpace(string(out)), strings.TrimSpace(string(ee.Stderr)), err
		}
		return strings.TrimSpace(string(out)), "", err
	}
	return strings.TrimSpace(string(out)), "", nil
}

func parseSnapshots(raw string, now time.Time) ([]Snapshot, error) {
	var doc any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &doc); err != nil {
		return nil, err
	}
	acc := map[string]float64{}
	visit(doc, func(m map[string]any) {
		amount, ok := pickAmount(m)
		if !ok || amount <= 0 {
			return
		}
		date := pickDate(m, now)
		acc[date] += amount
	})
	if len(acc) == 0 {
		return nil, fmt.Errorf("no cost points found")
	}
	out := make([]Snapshot, 0, len(acc))
	for d, a := range acc {
		out = append(out, Snapshot{
			Date:      d,
			AmountUSD: a,
			Metadata:  map[string]any{"source": "vercel_cli"},
		})
	}
	return out, nil
}

func visit(v any, fn func(map[string]any)) {
	switch x := v.(type) {
	case map[string]any:
		fn(x)
		for _, vv := range x {
			visit(vv, fn)
		}
	case []any:
		for _, vv := range x {
			visit(vv, fn)
		}
	}
}

func pickAmount(m map[string]any) (float64, bool) {
	keys := []string{"amount_usd", "amountUSD", "cost_usd", "costUSD", "cost", "amount", "total", "total_usd", "value"}
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch n := v.(type) {
		case float64:
			return n, true
		case string:
			f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
			if err == nil {
				return f, true
			}
		}
	}
	return 0, false
}

func pickDate(m map[string]any, now time.Time) string {
	for _, k := range []string{"date", "day", "snapshot_date", "timestamp", "start", "period_start", "from"} {
		v, ok := m[k]
		if !ok {
			continue
		}
		s, ok := v.(string)
		if !ok || strings.TrimSpace(s) == "" {
			continue
		}
		s = strings.TrimSpace(s)
		if len(s) >= 10 {
			return s[:10]
		}
	}
	return now.Format("2006-01-02")
}
