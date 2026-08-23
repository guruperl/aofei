package match

import (
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestGetSizeIDNativeForImpRejectsUnrepresentableDimensions(t *testing.T) {
	pointer := func(value int64) *int64 { return &value }
	tests := []struct {
		name string
		imp  openrtb2.Imp
	}{
		{name: "banner too wide", imp: openrtb2.Imp{Banner: &openrtb2.Banner{W: pointer(65536), H: pointer(250)}}},
		{name: "banner partial zero", imp: openrtb2.Imp{Banner: &openrtb2.Banner{W: pointer(0), H: pointer(250)}}},
		{name: "video negative", imp: openrtb2.Imp{Video: &openrtb2.Video{W: pointer(-1), H: pointer(360)}}},
		{name: "native image too wide", imp: openrtb2.Imp{Native: &openrtb2.Native{Request: `{"native":{"assets":[{"id":1,"img":{"w":65536,"h":250}}]}}`}}},
		{name: "native video too wide", imp: openrtb2.Imp{Native: &openrtb2.Native{Request: `{"native":{"assets":[{"id":1,"video":{"w":70000,"h":360}}]}}`}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := getSizeIDNativeForImp(&test.imp); err == nil || !strings.Contains(err.Error(), "supported range") {
				t.Fatalf("error = %v, want supported-range rejection", err)
			}
		})
	}
}

func TestNativeImageIncompletePreferredFallsBackToCompleteMinimum(t *testing.T) {
	imp := &openrtb2.Imp{Native: &openrtb2.Native{Request: `{"native":{"assets":[{"id":1,"img":{"w":300,"h":0,"wmin":64,"hmin":64}}]}}`}}
	sizeID, _, err := getSizeIDNativeForImp(imp)
	if err != nil {
		t.Fatalf("incomplete preferred image with complete minimum errored: %v", err)
	}
	if sizeID != SizeID2To1(64, 64) {
		t.Fatalf("size id = %d, want %d from wmin/hmin fallback", sizeID, SizeID2To1(64, 64))
	}
}

func TestNativeImageIncompleteMinimumRejected(t *testing.T) {
	imp := &openrtb2.Imp{Native: &openrtb2.Native{Request: `{"native":{"assets":[{"id":1,"img":{"wmin":64,"hmin":0}}]}}`}}
	if _, _, err := getSizeIDNativeForImp(imp); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("error = %v, want incomplete-minimum rejection", err)
	}
}

func TestNativeVideoZeroDimensionsOmitted(t *testing.T) {
	imp := &openrtb2.Imp{Native: &openrtb2.Native{Request: `{"native":{"assets":[{"id":1,"video":{"w":0,"h":0}}]}}`}}
	sizeID, _, err := getSizeIDNativeForImp(imp)
	if err != nil {
		t.Fatalf("zero-dimension native video errored: %v", err)
	}
	if sizeID != 0 {
		t.Fatalf("size id = %d, want 0 for omitted native video dimensions", sizeID)
	}
}

func TestBannerZeroUsesFirstExactFormat(t *testing.T) {
	pointer := func(value int64) *int64 { return &value }
	imp := &openrtb2.Imp{Banner: &openrtb2.Banner{
		W: pointer(0), H: pointer(0),
		Format: []openrtb2.Format{{W: 300, H: 250}, {W: 320, H: 50}},
	}}
	sizeID, _, err := getSizeIDNativeForImp(imp)
	if err != nil {
		t.Fatalf("banner 0x0 with exact format errored: %v", err)
	}
	if sizeID != SizeID2To1(300, 250) {
		t.Fatalf("size id = %d, want %d from first exact format", sizeID, SizeID2To1(300, 250))
	}
}

func TestGetSizeIDNativeForImpAcceptsLargestRepresentableDimension(t *testing.T) {
	w, h := int64(65535), int64(1)
	sizeID, _, err := getSizeIDNativeForImp(&openrtb2.Imp{Banner: &openrtb2.Banner{W: &w, H: &h}})
	if err != nil {
		t.Fatal(err)
	}
	if sizeID != SizeID2To1(65535, 1) {
		t.Fatalf("size id = %d, want %d", sizeID, SizeID2To1(65535, 1))
	}
}

func TestGetSizeIDNativeForImpValidatesEveryAsset(t *testing.T) {
	requests := []string{
		`{"native":{"assets":[{"id":1,"img":{"w":300,"h":250}},{"id":2,"img":{"w":65536,"h":250}}]}}`,
		`{"native":{"assets":[{"id":1,"img":{"w":300,"h":250},"video":{"w":70000,"h":360}}]}}`,
	}
	for _, request := range requests {
		imp := &openrtb2.Imp{Native: &openrtb2.Native{Request: request}}
		if _, _, err := getSizeIDNativeForImp(imp); err == nil || !strings.Contains(err.Error(), "supported range") {
			t.Fatalf("Native request %s error = %v, want supported-range rejection", request, err)
		}
	}
}
