package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func resetData() {
	mu.Lock()
	data = make(map[string]FriendLocation)
	mu.Unlock()
}

func TestMissingNameReturnsBadRequest(t *testing.T) {
	resetData()

	mux := http.NewServeMux()
	mux.Handle("/", authMiddleware("test-key", http.HandlerFunc(handleGet)))
	mux.Handle("/post", authMiddleware("test-key", http.HandlerFunc(handlePost)))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	body := map[string]interface{}{"x": 1, "y": 2, "plane": 0}
	b, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/post", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "test-key")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	var got map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got["error"] != "name is required" {
		t.Fatalf("expected error message %q, got %q", "name is required", got["error"])
	}
}

func TestAuthRequired(t *testing.T) {
	resetData()

	mux := http.NewServeMux()
	mux.Handle("/", authMiddleware("test-key", http.HandlerFunc(handleGet)))
	mux.Handle("/post", authMiddleware("test-key", http.HandlerFunc(handlePost)))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Missing header
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
	resp.Body.Close()

	// Wrong key
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/", nil)
	req.Header.Set("Authorization", "wrong")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, resp.StatusCode)
	}
	resp.Body.Close()
}

func TestCleanupOldRemovesExpiredEntries(t *testing.T) {
	resetData()

	loc := FriendLocation{Name: "foo", X: 1, Y: 2, Plane: 0, Timestamp: time.Now().UnixMilli() - (expirationMs + 1)}
	updateData(loc)

	cleanupOld(time.Now().UnixMilli())
	if got := getData(); len(got) != 0 {
		t.Fatalf("expected 0 entries after cleanup, got %d", len(got))
	}
}

func TestStartCleanupLoopRemovesExpiredEntries(t *testing.T) {
	resetData()

	loc := FriendLocation{Name: "bar", X: 1, Y: 2, Plane: 0, Timestamp: time.Now().UnixMilli() - (expirationMs + 1)}
	updateData(loc)

	stop := make(chan struct{})
	go startCleanupLoop(10*time.Millisecond, stop)
	defer close(stop)

	// Give the loop time to run at least once
	time.Sleep(50 * time.Millisecond)

	if got := getData(); len(got) != 0 {
		t.Fatalf("expected 0 entries after cleanup loop, got %d", len(got))
	}
}
