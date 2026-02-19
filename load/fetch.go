/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package load

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"bennypowers.dev/asimonim/internal/version"
)

const (
	// DefaultTimeout is the maximum time to wait for a network fetch.
	DefaultTimeout = 30 * time.Second

	// DefaultMaxSize is the maximum allowed response size (10 MB).
	DefaultMaxSize int64 = 10 * 1024 * 1024
)

// Fetcher fetches content from a URL.
type Fetcher interface {
	Fetch(ctx context.Context, url string) ([]byte, error)
}

// HTTPFetcher fetches content over HTTP with size limiting.
type HTTPFetcher struct {
	maxSize int64
	client  *http.Client
}

// NewHTTPFetcher creates an HTTPFetcher with the given maximum response size.
func NewHTTPFetcher(maxSize int64) *HTTPFetcher {
	return &HTTPFetcher{
		maxSize: maxSize,
		client:  &http.Client{},
	}
}

// Fetch fetches content from the given URL.
func (f *HTTPFetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", url, err)
	}

	req.Header.Set("User-Agent", "asimonim/"+version.Get())

	resp, err := f.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("timeout fetching %s: %w", url, err)
		}
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: %s", url, resp.Status)
	}

	limitedReader := io.LimitReader(resp.Body, f.maxSize+1)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", url, err)
	}

	if int64(len(content)) > f.maxSize {
		return nil, fmt.Errorf("response from %s exceeds maximum size of %d bytes", url, f.maxSize)
	}

	return content, nil
}
