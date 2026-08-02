package match

import (
	"context"
	"database/sql/driver"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreativeMapFromIOKeysByCreativeID(t *testing.T) {
	top := t.TempDir()
	dir := filepath.Join(top, HashNameCreative)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, "42"))
	if err != nil {
		t.Fatal(err)
	}
	creative := &Creative{CreativeName: "creative", SizeID: 300250}
	if err := creative.PackIO(f); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	creatives, err := CreativeMapFromIO(top)
	if err != nil {
		t.Fatal(err)
	}
	if creatives[42] == nil {
		t.Fatalf("creative map missing creative id key")
	}
	if creatives[300250] != nil {
		t.Fatalf("creative map unexpectedly keyed by size id")
	}
}

type recordingCreativeSink struct {
	creativeIDs []uint32
	creatives   []*Creative
}

func (recordingCreativeSink) ResetRAdvs(context.Context, uint32) error { return nil }
func (recordingCreativeSink) PutRAdvs(context.Context, uint32, uint32, []byte, bool) error {
	return nil
}
func (recordingCreativeSink) CleanupRAdvs(context.Context, uint32) error { return nil }
func (recordingCreativeSink) PutAudience(context.Context, uint32, []byte) error {
	return nil
}
func (s *recordingCreativeSink) PutCreative(_ context.Context, creativeID uint32, data []byte) error {
	s.creativeIDs = append(s.creativeIDs, creativeID)
	creative, err := UnpackCreative(data)
	if err != nil {
		return err
	}
	s.creatives = append(s.creatives, creative)
	return nil
}

func TestDBGetCreativesToRedisSpreadFilters(t *testing.T) {
	tests := []struct {
		name      string
		extra     []string
		wantQuery string
		wantArgs  []driver.Value
	}{
		{
			name:      "empty",
			wantQuery: `(?s)WHERE a\.active="Yes" AND c\.active="Yes" AND i\.active="Yes" AND r\.active="Yes"$`,
		},
		{
			name:      "item_id",
			extra:     []string{"item_id", "10"},
			wantQuery: `(?s)WHERE a\.active="Yes" AND c\.active="Yes" AND i\.active="Yes" AND r\.active="Yes" AND i\.item_id=\?`,
			wantArgs:  []driver.Value{"10"},
		},
		{
			name:      "campaign_id",
			extra:     []string{"campaign_id", "20"},
			wantQuery: `(?s)WHERE a\.active="Yes" AND c\.active="Yes" AND i\.active="Yes" AND r\.active="Yes" AND c\.campaign_id=\?`,
			wantArgs:  []driver.Value{"20"},
		},
		{
			name:      "creative_id",
			extra:     []string{"creative_id", "30"},
			wantQuery: `(?s)WHERE a\.active="Yes" AND c\.active="Yes" AND i\.active="Yes" AND r\.active="Yes" AND r\.creative_id=\?`,
			wantArgs:  []driver.Value{"30"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			rows := sqlmock.NewRows([]string{
				"creative_id", "size_id", "weight", "iurl", "item_click", "imp_url", "click_url", "creative_name", "content", "media_type", "mime",
			}).AddRow(100, SizeID2To1(300, 250), 1, nil, "https://advertiser.example/landing", nil, nil, "creative", "https://cdn.example/banner.html", "Banner", "text/html")
			expect := mock.ExpectQuery(tt.wantQuery)
			if len(tt.wantArgs) != 0 {
				expect.WithArgs(tt.wantArgs...)
			}
			expect.WillReturnRows(rows)

			sink := &recordingCreativeSink{}
			if err := DBGetCreativesToRedisSpread(context.Background(), sink, db, tt.extra...); err != nil {
				t.Fatal(err)
			}
			if len(sink.creativeIDs) != 1 || sink.creativeIDs[0] != 100 {
				t.Fatalf("creative ids = %#v, want [100]", sink.creativeIDs)
			}
			if len(sink.creatives) != 1 || sink.creatives[0].Failback != "" {
				t.Fatalf("database compiler reused campaign external id as fallback: %#v", sink.creatives)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDBGetCreativesToRedisSpreadRejectsMalformedFilters(t *testing.T) {
	tests := [][]string{
		{"item_id"},
		{"item_id", "1", "extra"},
		{"unknown_id", "1"},
		{"creative_id", ""},
	}
	for _, extra := range tests {
		t.Run(strings.Join(extra, ","), func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			err = DBGetCreativesToRedisSpread(context.Background(), &recordingCreativeSink{}, db, extra...)
			if err == nil {
				t.Fatalf("DBGetCreativesToRedisSpread(%q) error = nil, want error", extra)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDBGetCreativesValidatesWholeSelectionBeforePublishing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	columns := []string{
		"creative_id", "size_id", "weight", "iurl", "item_click", "imp_url", "click_url",
		"creative_name", "content", "media_type", "mime",
	}
	rows := sqlmock.NewRows(columns).
		AddRow(100, SizeID2To1(300, 250), 1, nil, "https://advertiser.example/landing", nil, nil, "valid", "https://cdn.example/banner.html", "Banner", "text/html").
		AddRow(101, SizeID2To1(300, 250), 1, nil, "javascript:alert(1)", nil, nil, "invalid", "https://cdn.example/banner.html", "Banner", "text/html")
	mock.ExpectQuery(`(?s)WHERE a\.active="Yes".*r\.active="Yes"$`).WillReturnRows(rows)
	sink := &recordingCreativeSink{}

	err = DBGetCreativesToRedisSpread(context.Background(), sink, db)
	if err == nil || !strings.Contains(err.Error(), "creative 101") {
		t.Fatalf("validation error = %v", err)
	}
	if len(sink.creativeIDs) != 0 {
		t.Fatalf("partially published creative IDs = %#v", sink.creativeIDs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDBValidateItemCreativesForActivationDoesNotRequireActiveItem(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows := sqlmock.NewRows([]string{
		"creative_id", "size_id", "weight", "iurl", "item_click", "imp_url", "click_url",
		"creative_name", "content", "media_type", "mime",
	}).AddRow(100, SizeID2To1(300, 250), 0, nil, "https://advertiser.example/landing", nil, nil, "creative", "https://cdn.example/banner.html", "Banner", "text/html")
	mock.ExpectQuery(`(?s)WHERE r\.active="Yes" AND i\.item_id=\?`).WithArgs("7").WillReturnRows(rows)

	err = DBValidateItemCreativesForActivation(context.Background(), db, "7")
	if err == nil || !strings.Contains(err.Error(), "invalid rotation weight") {
		t.Fatalf("activation validation error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDBValidateItemCreativesForActivationRequiresCreative(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	columns := []string{
		"creative_id", "size_id", "weight", "iurl", "item_click", "imp_url", "click_url",
		"creative_name", "content", "media_type", "mime",
	}
	mock.ExpectQuery(`(?s)WHERE r\.active="Yes" AND i\.item_id=\?`).WithArgs("7").WillReturnRows(sqlmock.NewRows(columns))
	if err := DBValidateItemCreativesForActivation(context.Background(), db, "7"); err == nil || !strings.Contains(err.Error(), "no active creatives") {
		t.Fatalf("empty activation validation error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
