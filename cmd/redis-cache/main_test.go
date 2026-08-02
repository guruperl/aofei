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

func TestValidateCommandModesRejectsAmbiguousOperations(t *testing.T) {
	if err := validateCommandModes(true, false, true, false, cachejob.MiddlemanStagePreflight); err == nil {
		t.Fatal("read and publisher validation were accepted together")
	}
	if err := validateCommandModes(false, true, false, true, cachejob.MiddlemanStagePreflight); err == nil {
		t.Fatal("update and middleman validation were accepted together")
	}
	if err := validateCommandModes(false, false, false, false, cachejob.MiddlemanStageAlways); err == nil {
		t.Fatal("activation stage without middleman validation was accepted")
	}
	if err := validateCommandModes(false, false, false, true, cachejob.MiddlemanStageAlways); err != nil {
		t.Fatalf("valid middleman activation flags: %v", err)
	}
}
