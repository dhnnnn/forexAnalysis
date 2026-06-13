package sentiment

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Compile-time check that NewsFetcher implements HeadlineFetcher.
var _ HeadlineFetcher = (*NewsFetcher)(nil)

// NewsFetcher retrieves news headlines from multiple sources concurrently.
// It implements the HeadlineFetcher interface.
type NewsFetcher struct {
	alphaVantageKey string
	twelveDataKey   string
	rssURLs         []string
	httpClient      *http.Client
}

// NewNewsFetcher creates a NewsFetcher with the given API keys and RSS feed URLs.
// A default HTTP client with a 10-second timeout is used for all requests.
func NewNewsFetcher(avKey, tdKey string, rssURLs []string) *NewsFetcher {
	return &NewsFetcher{
		alphaVantageKey: avKey,
		twelveDataKey:   tdKey,
		rssURLs:         rssURLs,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FetchForPair retrieves headlines relevant to the given currency pair from all
// configured sources concurrently. It tolerates partial failures: if some sources
// fail, headlines from successful sources are still returned.
//
// If ALL sources fail, an empty slice and nil error are returned (not an error).
func (f *NewsFetcher) FetchForPair(ctx context.Context, pair string) ([]string, error) {
	var (
		mu        sync.Mutex
		headlines []string
	)

	// Use a plain errgroup (no WithContext) so that individual source failures
	// do NOT cancel the context for other in-flight goroutines. This ensures
	// partial failure tolerance: successful sources still contribute headlines.
	var g errgroup.Group

	// Alpha Vantage source
	if f.alphaVantageKey != "" {
		g.Go(func() error {
			results, err := f.fetchAlphaVantage(ctx, pair)
			if err != nil {
				return err
			}
			mu.Lock()
			headlines = append(headlines, results...)
			mu.Unlock()
			return nil
		})
	}

	// Twelve Data source
	if f.twelveDataKey != "" {
		g.Go(func() error {
			results, err := f.fetchTwelveData(ctx, pair)
			if err != nil {
				return err
			}
			mu.Lock()
			headlines = append(headlines, results...)
			mu.Unlock()
			return nil
		})
	}

	// RSS feeds — each feed in its own goroutine
	for _, url := range f.rssURLs {
		url := url // capture loop variable
		g.Go(func() error {
			results, err := f.fetchRSS(ctx, url)
			if err != nil {
				return err
			}
			mu.Lock()
			headlines = append(headlines, results...)
			mu.Unlock()
			return nil
		})
	}

	// Wait for all goroutines. errgroup returns the first error, but we
	// already collected successful results via the mutex. We ignore the
	// aggregate error and return whatever headlines we gathered.
	_ = g.Wait()

	// Return empty slice (not nil) when no headlines were collected.
	if headlines == nil {
		return []string{}, nil
	}

	return headlines, nil
}

// ════════════════════════════════════════════════════════════════════════
// Alpha Vantage
// ════════════════════════════════════════════════════════════════════════

// alphaVantageResponse represents the JSON structure returned by the
// Alpha Vantage NEWS_SENTIMENT endpoint.
type alphaVantageResponse struct {
	Feed []struct {
		Title string `json:"title"`
	} `json:"feed"`
}

// fetchAlphaVantage fetches headlines from Alpha Vantage News Sentiment API.
// The pair (e.g. "EUR_USD") is converted to the base currency ticker (e.g. "EUR")
// for the FOREX tickers parameter.
func (f *NewsFetcher) fetchAlphaVantage(ctx context.Context, pair string) ([]string, error) {
	// Convert pair format: "EUR_USD" -> "EUR" (base currency)
	ticker := pairToAlphaVantageTicker(pair)

	url := fmt.Sprintf(
		"https://www.alphavantage.co/query?function=NEWS_SENTIMENT&tickers=FOREX:%s&apikey=%s",
		ticker, f.alphaVantageKey,
	)

	body, err := f.doGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("alpha vantage: %w", err)
	}

	var resp alphaVantageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("alpha vantage: failed to parse response: %w", err)
	}

	var headlines []string
	for _, item := range resp.Feed {
		if title := strings.TrimSpace(item.Title); title != "" {
			headlines = append(headlines, title)
		}
	}

	return headlines, nil
}

// pairToAlphaVantageTicker converts a pair like "EUR_USD" to the base currency "EUR".
func pairToAlphaVantageTicker(pair string) string {
	parts := strings.SplitN(pair, "_", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return pair
}

// ════════════════════════════════════════════════════════════════════════
// Twelve Data
// ════════════════════════════════════════════════════════════════════════

// twelveDataResponse represents the JSON structure returned by the
// Twelve Data news endpoint.
type twelveDataResponse struct {
	Data []struct {
		Title string `json:"title"`
	} `json:"data"`
}

// fetchTwelveData fetches headlines from the Twelve Data news endpoint.
// The pair (e.g. "EUR_USD") is converted to "EUR/USD" for the API.
func (f *NewsFetcher) fetchTwelveData(ctx context.Context, pair string) ([]string, error) {
	// Convert pair format: "EUR_USD" -> "EUR/USD"
	symbol := strings.ReplaceAll(pair, "_", "/")

	url := fmt.Sprintf(
		"https://api.twelvedata.com/news?symbol=%s&apikey=%s",
		symbol, f.twelveDataKey,
	)

	body, err := f.doGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("twelve data: %w", err)
	}

	var resp twelveDataResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("twelve data: failed to parse response: %w", err)
	}

	var headlines []string
	for _, item := range resp.Data {
		if title := strings.TrimSpace(item.Title); title != "" {
			headlines = append(headlines, title)
		}
	}

	return headlines, nil
}

// ════════════════════════════════════════════════════════════════════════
// RSS Feeds
// ════════════════════════════════════════════════════════════════════════

// rssFeed represents a minimal RSS 2.0 XML structure.
type rssFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

// rssItem represents a single item in an RSS feed.
type rssItem struct {
	Title string `xml:"title"`
}

// fetchRSS fetches headlines from a single RSS feed URL.
func (f *NewsFetcher) fetchRSS(ctx context.Context, feedURL string) ([]string, error) {
	body, err := f.doGet(ctx, feedURL)
	if err != nil {
		return nil, fmt.Errorf("rss [%s]: %w", feedURL, err)
	}

	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("rss [%s]: failed to parse XML: %w", feedURL, err)
	}

	var headlines []string
	for _, item := range feed.Channel.Items {
		if title := strings.TrimSpace(item.Title); title != "" {
			headlines = append(headlines, title)
		}
	}

	return headlines, nil
}

// ════════════════════════════════════════════════════════════════════════
// HTTP helper
// ════════════════════════════════════════════════════════════════════════

// doGet performs an HTTP GET request with context support and returns the
// response body bytes.
func (f *NewsFetcher) doGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}
