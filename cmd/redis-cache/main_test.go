package main

import (
	"testing"

	cachejob "github.com/guruperl/aofei/internal/jobs/cache"
)

func TestCacheMutationLockKeyIsSharedAcrossModes(t *testing.T) {
	for _, mode := range []string{cachejob.ModeRedis, cachejob.ModeSpread, cachejob.ModeAll, cachejob.ModeRoutes} {
		if got := cacheMutationLockKey(mode); got != redisCacheMutationLockKey {
			t.Fatalf("cacheMutationLockKey(%q) = %q, want %q", mode, got, redisCacheMutationLockKey)
		}
	}
}
