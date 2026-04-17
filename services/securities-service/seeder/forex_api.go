package seeder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

const (
	exchangeRateBase = "https://open.er-api.com/v6/latest"
	frankfurterBase  = "https://api.frankfurter.app"
)

var forexHTTPClient = &http.Client{Timeout: 30 * time.Second}

// FetchFXRateFree fetches the current exchange rate from exchangerate-api.com (no key needed).
func FetchFXRateFree(from, to string) (*FXRate, error) {
	url := fmt.Sprintf("%s/%s", exchangeRateBase, from)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("exchangerate-api: build request: %w", err)
	}

	resp, err := forexHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchangerate-api: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exchangerate-api: status %d", resp.StatusCode)
	}

	var raw struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("exchangerate-api: decode: %w", err)
	}

	rate, ok := raw.Rates[to]
	if !ok {
		return nil, fmt.Errorf("exchangerate-api: no rate for %s/%s", from, to)
	}

	return &FXRate{
		Price: rate,
		Ask:   rate * 1.0005,
		Bid:   rate * 0.9995,
	}, nil
}

// FetchFXDailyFree fetches 30 days of daily forex history from Frankfurter (ECB rates, no key needed).
func FetchFXDailyFree(from, to string) ([]DailyBar, error) {
	end := time.Now()
	start := end.AddDate(0, 0, -30)

	url := fmt.Sprintf("%s/%s..%s?from=%s&to=%s",
		frankfurterBase,
		start.Format("2006-01-02"),
		end.Format("2006-01-02"),
		from, to)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("frankfurter: build request: %w", err)
	}

	resp, err := forexHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("frankfurter: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("frankfurter: status %d", resp.StatusCode)
	}

	var raw struct {
		Rates map[string]map[string]float64 `json:"rates"` // date -> currency -> rate
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("frankfurter: decode: %w", err)
	}

	bars := make([]DailyBar, 0, len(raw.Rates))
	for dateStr, rates := range raw.Rates {
		rate, ok := rates[to]
		if !ok {
			continue
		}
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		bars = append(bars, DailyBar{
			Date:  t,
			Price: rate,
			Ask:   rate * 1.0005,
			Bid:   rate * 0.9995,
		})
	}

	sort.Slice(bars, func(i, j int) bool { return bars[i].Date.Before(bars[j].Date) })
	for i := 1; i < len(bars); i++ {
		bars[i].Change = bars[i].Price - bars[i-1].Price
	}
	return bars, nil
}
