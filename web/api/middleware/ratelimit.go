package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/jtsilverman/agentdiff/web/api/db"
)

// RateLimitConfig configures the RateLimit middleware. KillSwitch is read once
// at startup from DEMO_KILL_SWITCH=true; Now is injectable for tests so the
// UTC-day key is deterministic.
type RateLimitConfig struct {
	DB         *db.DB
	Threshold  int
	KillSwitch bool
	Now        func() time.Time
}

// RateLimit returns chi-compatible middleware that enforces a per-IP per-UTC-day
// cap on wrapped handlers. Past threshold (count > Threshold), the middleware
// short-circuits with a 200 JSON {"easter_egg":true} payload instead of running
// the wrapped handler. KillSwitch=true short-circuits unconditionally without
// incrementing the counter or running the wrapped handler. Fails open on internal
// errors (DB write failure, IP extraction failure) per fail-open-on-pre-spawn-
// cost-filter — the worst case from one fail-open is one extra LLM call's cost,
// vs blocking a legitimate reviewer.
func RateLimit(cfg RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.KillSwitch {
				writeEasterEgg(w)
				return
			}

			ip := extractIP(r)
			if ip == "" {
				// Fail-open: unknown client (XFF missing, local dev, etc.).
				next.ServeHTTP(w, r)
				return
			}

			day := cfg.Now().UTC().Format("2006-01-02")
			count, err := cfg.DB.UpsertRateLimitCount(ip, day)
			if err != nil {
				// Fail-open per fail-open-on-pre-spawn-cost-filter.
				next.ServeHTTP(w, r)
				return
			}

			if count > cfg.Threshold {
				writeEasterEgg(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// extractIP pulls the originating client IP from proxy headers. Leftmost
// X-Forwarded-For wins (Render's proxy convention); X-Real-IP is a fallback.
// Spoofing is acknowledged but not defended-against — the threat model is
// portfolio-demo abuse, where the DEMO_KILL_SWITCH env var + Anthropic
// dashboard alerts are the real defense at scale.
func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	return ""
}

func writeEasterEgg(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"easter_egg":true}`))
}
