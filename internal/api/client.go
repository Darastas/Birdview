package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/egoist/mygo/netproxy"
)

const defaultUserAgent = "BirdviewDesktop/1.0 (Go + MyGo Native UI)"

// Client 封装遵循系统代理（netproxy）的统一 HTTP 客户端。
type Client struct {
	HTTP         *http.Client
	EBirdBaseURL string
	INatBaseURL  string
	WikiBaseURL  string
	XenoBaseURL  string
}

// NewClient 创建使用系统代理、默认超时 15 秒的 API 客户端。
func NewClient() *Client {
	return &Client{
		HTTP: &http.Client{
			Transport: netproxy.Transport(),
			Timeout:   15 * time.Second,
		},
		EBirdBaseURL: "https://api.ebird.org/v2",
		INatBaseURL:  "https://api.inaturalist.org/v1",
		WikiBaseURL:  "https://zh.wikipedia.org/api/rest_v1",
		XenoBaseURL:  "https://xeno-canto.org/api/3",
	}
}

func (c *Client) getJSON(ctx context.Context, rawURL string, headers map[string]string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 24*1024*1024))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
	}
	return json.Unmarshal(body, dst)
}

// FetchBytes 下载图片或音频的原始字节（限制最大 20MB）。
func (c *Client) FetchBytes(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 20*1024*1024))
}
