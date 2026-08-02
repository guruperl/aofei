package main

import (
	"regexp"
	"testing"
	"time"
)

func TestParseDateUsesUTCDate(t *testing.T) {
	got, err := parseDate("2026-08-01")
	if err != nil {
		t.Fatal(err)
	}
	if got.Location() != time.UTC || got.Format("2006-01-02") != "2026-08-01" {
		t.Fatalf("date = %s (%s)", got, got.Location())
	}
}

func TestEffectiveActorIsBoundToOSPrincipal(t *testing.T) {
	actor, err := effectiveActor()
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^unix-uid:[0-9]+$`).MatchString(actor) {
		t.Fatalf("effective actor = %q", actor)
	}
}
