package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/web/api/db"
)

// fixedNow returns a deterministic UTC instant so the per-day counter key is
// stable across test runs.
func fixedNow() time.Time {
	return time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
}

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.NewDB(":memory:")
	if err != nil {
		t.Fatalf("open in-memory DB: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func newTestRouter(t *testing.T, cfg RateLimitConfig) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	r.With(RateLimit(cfg)).Post("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"normal":true}`))
	})
	return r
}

func doPost(t *testing.T, r *chi.Mux, xff string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/test", strings.NewReader(`{}`))
	req.Header.Set("X-Forwarded-For", xff)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	var body map[string]any
	if rr.Body.Len() > 0 {
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("parse response body: %v (body=%q)", err, rr.Body.String())
		}
	}
	return rr.Code, body
}

func TestRateLimit_SameIP_SixthRequestEasterEgg(t *testing.T) {
	database := setupTestDB(t)
	r := newTestRouter(t, RateLimitConfig{
		DB:        database,
		Threshold: 5,
		Now:       fixedNow,
	})

	// First 5 POSTs from 192.0.2.1 should pass through to the wrapped handler.
	for i := 1; i <= 5; i++ {
		code, body := doPost(t, r, "192.0.2.1")
		if code != http.StatusOK {
			t.Fatalf("request %d: want 200, got %d", i, code)
		}
		if body["normal"] != true {
			t.Fatalf("request %d: want normal=true, got %v", i, body)
		}
	}

	// 6th POST should short-circuit to easter-egg payload.
	code, body := doPost(t, r, "192.0.2.1")
	if code != http.StatusOK {
		t.Fatalf("6th request: want 200, got %d", code)
	}
	if body["easter_egg"] != true {
		t.Fatalf("6th request: want easter_egg=true, got %v", body)
	}
	if body["normal"] == true {
		t.Fatalf("6th request: wrapped handler should not have run")
	}
}

func TestRateLimit_DifferentIPs_IndependentCounters(t *testing.T) {
	database := setupTestDB(t)
	r := newTestRouter(t, RateLimitConfig{
		DB:        database,
		Threshold: 5,
		Now:       fixedNow,
	})

	// Exhaust 192.0.2.1's allowance.
	for i := 0; i < 5; i++ {
		doPost(t, r, "192.0.2.1")
	}
	code1, body1 := doPost(t, r, "192.0.2.1")
	if code1 != http.StatusOK || body1["easter_egg"] != true {
		t.Fatalf("IP1 6th: want easter_egg=true, got code=%d body=%v", code1, body1)
	}

	// 192.0.2.2 must have its own counter — IP1 hitting threshold should not
	// affect IP2's first 5 requests (independence). The 6th from IP2 would
	// itself cross threshold (6 > 5), but that's the threshold behavior, not
	// the independence property we're proving here.
	for i := 1; i <= 5; i++ {
		code, body := doPost(t, r, "192.0.2.2")
		if code != http.StatusOK {
			t.Fatalf("IP2 request %d: want 200, got %d", i, code)
		}
		if body["normal"] != true {
			t.Fatalf("IP2 request %d: counter should be per-IP; got %v", i, body)
		}
	}
}

func TestRateLimit_KillSwitch_FirstRequestEasterEggNoIncrement(t *testing.T) {
	database := setupTestDB(t)
	r := newTestRouter(t, RateLimitConfig{
		DB:         database,
		Threshold:  5,
		KillSwitch: true,
		Now:        fixedNow,
	})

	// KillSwitch=true: every request returns easter-egg from the first.
	code, body := doPost(t, r, "192.0.2.1")
	if code != http.StatusOK {
		t.Fatalf("kill-switch first request: want 200, got %d", code)
	}
	if body["easter_egg"] != true {
		t.Fatalf("kill-switch first request: want easter_egg=true, got %v", body)
	}
	if body["normal"] == true {
		t.Fatalf("kill-switch: wrapped handler should not have run")
	}

	// Kill-switch must NOT have incremented the counter. Probe directly: a
	// fresh upsert from the same IP+day should return 1, not 2.
	count, err := database.UpsertRateLimitCount("192.0.2.1", "2026-05-26")
	if err != nil {
		t.Fatalf("post-kill-switch upsert: %v", err)
	}
	if count != 1 {
		t.Fatalf("kill-switch leaked an increment; first manual upsert should be 1, got %d", count)
	}
}
