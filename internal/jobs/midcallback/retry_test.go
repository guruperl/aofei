package midcallback

import (
	"net/http"
	"testing"
	"time"
)

func TestRetryableForward(t *testing.T) {
	tests := []struct {
		status string
		code   int
		want   bool
	}{
		{"error", 0, true},
		{"request_error", 0, true},
		{"http_error", http.StatusTooManyRequests, true},
		{"http_error", http.StatusInternalServerError, true},
		{"http_error", http.StatusBadRequest, false},
		{"missing", 0, false},
		{"invalid_url", 0, false},
		{"duplicate", 0, false},
		{"ok", http.StatusNoContent, false},
	}
	for _, tt := range tests {
		if got := RetryableForward(tt.status, tt.code); got != tt.want {
			t.Fatalf("RetryableForward(%q, %d) = %v, want %v", tt.status, tt.code, got, tt.want)
		}
	}
}

func TestBackoff(t *testing.T) {
	base := time.Minute
	max := 5 * time.Minute
	if got := backoff(1, base, max); got != time.Minute {
		t.Fatalf("attempt 1 backoff = %s", got)
	}
	if got := backoff(3, base, max); got != 4*time.Minute {
		t.Fatalf("attempt 3 backoff = %s", got)
	}
	if got := backoff(10, base, max); got != max {
		t.Fatalf("capped backoff = %s", got)
	}
}
