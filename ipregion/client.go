package ipregion

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type httpClient struct {
	ipv4 *http.Client
	ipv6 *http.Client
	ua   string
}

func newHTTPClient(timeout time.Duration, userAgent string) *httpClient {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if userAgent == "" {
		userAgent = defaultUserAgent
	}
	return &httpClient{
		ipv4: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					d := net.Dialer{Timeout: timeout}
					return d.DialContext(ctx, "tcp4", addr)
				},
			},
		},
		ipv6: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					d := net.Dialer{Timeout: timeout}
					return d.DialContext(ctx, "tcp6", addr)
				},
			},
		},
		ua: userAgent,
	}
}

type requestOptions struct {
	method      string
	url         string
	headers     map[string]string
	body        string
	contentType string
	ipVersion   int
}

func (c *httpClient) do(ctx context.Context, opts requestOptions) (string, int, error) {
	client := c.ipv4
	if opts.ipVersion == 6 {
		client = c.ipv6
	}

	method := opts.method
	if method == "" {
		method = http.MethodGet
	}

	var bodyReader io.Reader
	if opts.body != "" {
		bodyReader = strings.NewReader(opts.body)
	}

	req, err := http.NewRequestWithContext(ctx, method, opts.url, bodyReader)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", c.ua)
	for k, v := range opts.headers {
		req.Header.Set(k, v)
	}
	if opts.contentType != "" {
		req.Header.Set("Content-Type", opts.contentType)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, err
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return "", resp.StatusCode, nil
	}
	if resp.StatusCode >= 400 {
		return "", resp.StatusCode, fmt.Errorf("http %d", resp.StatusCode)
	}
	return string(data), resp.StatusCode, nil
}

func (c *httpClient) get(ctx context.Context, url string, ipVersion int, headers map[string]string) (string, error) {
	body, _, err := c.do(ctx, requestOptions{
		method:    http.MethodGet,
		url:       url,
		headers:   headers,
		ipVersion: ipVersion,
	})
	return body, err
}

func (c *httpClient) postForm(ctx context.Context, url string, ipVersion int, form string, headers map[string]string) (string, error) {
	body, _, err := c.do(ctx, requestOptions{
		method:      http.MethodPost,
		url:         url,
		body:        form,
		contentType: "application/x-www-form-urlencoded",
		headers:     headers,
		ipVersion:   ipVersion,
	})
	return body, err
}

func (c *httpClient) postJSON(ctx context.Context, url string, ipVersion int, payload string, headers map[string]string) (string, error) {
	body, _, err := c.do(ctx, requestOptions{
		method:      http.MethodPost,
		url:         url,
		body:        payload,
		contentType: "application/json",
		headers:     headers,
		ipVersion:   ipVersion,
	})
	return body, err
}

func (c *httpClient) doHeadHeaders(ctx context.Context, url string, ipVersion int) (string, error) {
	client := c.ipv4
	if ipVersion == 6 {
		client = c.ipv6
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", c.ua)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	var b strings.Builder
	for k, vals := range resp.Header {
		for _, v := range vals {
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(v)
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}

func cleanResult(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\t", " ")
	if s == "" || s == "null" || strings.EqualFold(s, "n/a") {
		return NotAvailable
	}
	return s
}
