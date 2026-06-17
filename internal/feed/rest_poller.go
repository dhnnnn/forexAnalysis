package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// RESTPoller mengambil data OHLCV dari REST API (Twelve Data / Alpha Vantage)
// sebagai fallback ketika WebSocket tidak tersedia.
type RESTPoller struct {
	Output chan OHLCVCandle

	baseURL string
	apiKey  string
	pairs   []string
	client  *http.Client
}

// NewRESTPoller membuat REST poller baru.
func NewRESTPoller(baseURL, apiKey string, pairs []string) *RESTPoller {
	return &RESTPoller{
		Output:  make(chan OHLCVCandle, 100),
		baseURL: baseURL,
		apiKey:  apiKey,
		pairs:   pairs,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// HasAPIKey returns true jika API key tersedia dan tidak kosong.
func (p *RESTPoller) HasAPIKey() bool {
	return p.apiKey != "" && p.apiKey != "${TWELVE_DATA_KEY}"
}

// TwelveDataResponse adalah response dari Twelve Data time_series endpoint.
type TwelveDataResponse struct {
	Values []TwelveDataValue `json:"values"`
	Status string            `json:"status"`
}

// TwelveDataValue adalah satu candle dari Twelve Data.
type TwelveDataValue struct {
	Datetime string `json:"datetime"`
	Open     string `json:"open"`
	High     string `json:"high"`
	Low      string `json:"low"`
	Close    string `json:"close"`
	Volume   string `json:"volume"`
}

// FetchCandles mengambil candle terbaru dari Twelve Data API.
// Dipanggil oleh MarketDataAgent sebagai fallback.
func (p *RESTPoller) FetchCandles(ctx context.Context, pair, timeframe string, count int) ([]OHLCVCandle, error) {
	// Konversi pair format: EUR_USD → EUR/USD
	symbol := convertPairFormat(pair)

	url := fmt.Sprintf("%s/time_series?symbol=%s&interval=%s&outputsize=%d&apikey=%s",
		p.baseURL, symbol, timeframe, count, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("rest_poller: failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rest_poller: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rest_poller: API returned HTTP %d", resp.StatusCode)
	}

	var data TwelveDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("rest_poller: failed to decode response: %w", err)
	}

	if data.Status == "error" {
		return nil, fmt.Errorf("rest_poller: API returned error status")
	}

	var candles []OHLCVCandle
	for _, v := range data.Values {
		c, err := parseTwelveDataCandle(pair, timeframe, v)
		if err != nil {
			slog.Warn("rest_poller: skipping invalid candle", "error", err)
			continue
		}

		normalized, err := Normalize(c)
		if err != nil {
			slog.Warn("rest_poller: normalization failed", "error", err)
			continue
		}
		candles = append(candles, normalized)
	}

	return candles, nil
}

// parseTwelveDataCandle mengkonversi TwelveDataValue ke OHLCVCandle.
func parseTwelveDataCandle(pair, timeframe string, v TwelveDataValue) (OHLCVCandle, error) {
	t, err := time.Parse("2006-01-02 15:04:05", v.Datetime)
	if err != nil {
		return OHLCVCandle{}, fmt.Errorf("parse time: %w", err)
	}

	open, err := parseFloat(v.Open)
	if err != nil {
		return OHLCVCandle{}, err
	}
	high, err := parseFloat(v.High)
	if err != nil {
		return OHLCVCandle{}, err
	}
	low, err := parseFloat(v.Low)
	if err != nil {
		return OHLCVCandle{}, err
	}
	closeP, err := parseFloat(v.Close)
	if err != nil {
		return OHLCVCandle{}, err
	}
	vol, _ := parseFloat(v.Volume) // volume bisa 0

	return OHLCVCandle{
		Pair:      pair,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     closeP,
		Volume:    vol,
		Timeframe: timeframe,
		Timestamp: t,
	}, nil
}

// parseFloat mengkonversi string ke float64.
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// convertPairFormat mengkonversi EUR_USD → EUR/USD.
func convertPairFormat(pair string) string {
	if len(pair) == 7 && pair[3] == '_' {
		return pair[:3] + "/" + pair[4:]
	}
	return pair
}
