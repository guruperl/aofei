package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/guruperl/aofei/internal/jobs/midcallback"
)

func TestWriteRetryReportText(t *testing.T) {
	var buf bytes.Buffer
	err := writeRetryReport(&buf, false,
		midcallback.BacklogStats{Due: 3, StaleProcessing: 1},
		midcallback.Result{Selected: 2, Succeeded: 1, Retrying: 1, Abandoned: 0},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "due=3 stale_processing=1 selected=2 succeeded=1 retrying=1 abandoned=0\n"
	if buf.String() != want {
		t.Fatalf("text report = %q, want %q", buf.String(), want)
	}
}

func TestWriteRetryReportJSON(t *testing.T) {
	var buf bytes.Buffer
	err := writeRetryReport(&buf, true,
		midcallback.BacklogStats{Due: 3, StaleProcessing: 1},
		midcallback.Result{Selected: 2, Succeeded: 1, Retrying: 1, Abandoned: 0},
	)
	if err != nil {
		t.Fatal(err)
	}
	var got retryReport
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := retryReport{
		Due:             3,
		StaleProcessing: 1,
		Selected:        2,
		Succeeded:       1,
		Retrying:        1,
		Abandoned:       0,
	}
	if got != want {
		t.Fatalf("json report = %#v, want %#v", got, want)
	}
}
