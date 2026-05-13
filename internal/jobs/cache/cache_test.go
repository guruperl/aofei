package cache

import (
	"context"
	"testing"
)

func TestValidateMode(t *testing.T) {
	for _, mode := range []string{ModeRedis, ModeSpread, ModeAll} {
		if err := ValidateMode(mode); err != nil {
			t.Fatalf("ValidateMode(%q) = %v", mode, err)
		}
	}
	if err := ValidateMode("bad"); err == nil {
		t.Fatal("ValidateMode(bad) = nil, want error")
	}
}

func TestRunRejectsInvalidUpdateInterval(t *testing.T) {
	err := Run(context.TODO(), nil, nil, nil, nil, Options{Mode: ModeRedis})
	if err == nil {
		t.Fatal("Run with zero update interval = nil, want error")
	}
}
