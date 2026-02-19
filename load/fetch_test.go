/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package load

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPFetcher_Success(t *testing.T) {
	body := `{"color": {"$value": "#fff", "$type": "color"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	f := NewHTTPFetcher(DefaultMaxSize)
	content, err := f.Fetch(context.Background(), srv.URL+"/tokens.json")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if string(content) != body {
		t.Errorf("Fetch() = %q, want %q", string(content), body)
	}
}

func TestHTTPFetcher_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte("too late"))
	}))
	defer srv.Close()

	f := NewHTTPFetcher(DefaultMaxSize)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := f.Fetch(ctx, srv.URL+"/tokens.json")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestHTTPFetcher_MaxSizeExceeded(t *testing.T) {
	// Response is 100 bytes, limit to 50
	body := strings.Repeat("x", 100)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	f := NewHTTPFetcher(50)
	_, err := f.Fetch(context.Background(), srv.URL+"/tokens.json")
	if err == nil {
		t.Fatal("expected max size error")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("expected max size error, got: %v", err)
	}
}

func TestHTTPFetcher_Non200Status(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	f := NewHTTPFetcher(DefaultMaxSize)
	_, err := f.Fetch(context.Background(), srv.URL+"/tokens.json")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error, got: %v", err)
	}
}
