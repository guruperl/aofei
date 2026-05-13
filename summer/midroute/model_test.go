package midroute

import "testing"

func TestRouteCacheFreshness(t *testing.T) {
	tests := []struct {
		name      string
		cacheHigh string
		dbHigh    string
		want      string
	}{
		{"fresh", "2026-05-13T10:00:00Z", "2026-05-13T10:00:00Z", "fresh"},
		{"fresh empty", "", "", "fresh"},
		{"stale", "2026-05-13T09:59:59Z", "2026-05-13T10:00:00Z", "stale"},
		{"stale missing cache", "", "2026-05-13T10:00:00Z", "stale"},
		{"unknown generated", "2026-05-13T10:00:00Z", "2026-05-13T10:00:00Z", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generatedAt := "2026-05-13T10:01:00Z"
			if tt.name == "unknown generated" {
				generatedAt = "bad"
			}
			got := routeCacheFreshness(generatedAt, tt.cacheHigh, tt.dbHigh)
			if got != tt.want {
				t.Fatalf("routeCacheFreshness = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRouteHealthQueries(t *testing.T) {
	queries := routeHealthQueries()
	if len(queries) != 5 {
		t.Fatalf("len(routeHealthQueries) = %d, want 5", len(queries))
	}
}
