// Package frankfurter implements the outbound Frankfurter API gateway.
package frankfurter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
)

const maximumResponseSize = 1 << 20

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 20
	transport.MaxIdleConnsPerHost = 10
	transport.IdleConnTimeout = 90 * time.Second

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

func (c *Client) Fetch(ctx context.Context, pair domain.CurrencyPair) (domain.Quote, error) {
	requestURL, err := url.JoinPath(c.baseURL, "rate", string(pair.Base), string(pair.Quote))
	if err != nil {
		return domain.Quote{}, fmt.Errorf("build quote provider URL: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return domain.Quote{}, fmt.Errorf("build quote provider request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "plata-go-currency-quotes/1.0")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return domain.Quote{}, fmt.Errorf("request quote provider: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maximumResponseSize))
		return domain.Quote{}, fmt.Errorf("quote provider returned HTTP %d", response.StatusCode)
	}

	var payload struct {
		Base  string  `json:"base"`
		Quote string  `json:"quote"`
		Rate  float64 `json:"rate"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maximumResponseSize))
	if err := decoder.Decode(&payload); err != nil {
		return domain.Quote{}, fmt.Errorf("decode quote provider response: %w", err)
	}

	if payload.Base != string(pair.Base) || payload.Quote != string(pair.Quote) {
		return domain.Quote{}, fmt.Errorf(
			"quote provider returned unexpected pair %s/%s",
			payload.Base,
			payload.Quote,
		)
	}
	if payload.Rate <= 0 || math.IsNaN(payload.Rate) || math.IsInf(payload.Rate, 0) {
		return domain.Quote{}, fmt.Errorf("quote provider returned invalid rate %v", payload.Rate)
	}

	return domain.Quote{Price: payload.Rate}, nil
}
