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
}

func (recordingCreativeSink) ResetRAdvs(context.Context, uint32) error { return nil }
func (recordingCreativeSink) PutRAdvs(context.Context, uint32, uint32, []byte, bool) error {
	return nil
}
func (recordingCreativeSink) CleanupRAdvs(context.Context, uint32) error { return nil }
func (recordingCreativeSink) PutAudience(context.Context, uint32, []byte) error {
	return nil
}
func (s *recordingCreativeSink) PutCreative(_ context.Context, creativeID uint32, _ []byte) error {
	s.creativeIDs = append(s.creativeIDs, creativeID)
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
			wantQuery: `(?s)WHERE r\.active="Yes"$`,
		},
		{
			name:      "item_id",
			extra:     []string{"item_id", "10"},
			wantQuery: `(?s)WHERE r\.active="Yes" AND i\.item_id=\?`,
			wantArgs:  []driver.Value{"10"},
		},
		{
			name:      "campaign_id",
			extra:     []string{"campaign_id", "20"},
			wantQuery: `(?s)WHERE r\.active="Yes" AND c\.campaign_id=\?`,
			wantArgs:  []driver.Value{"20"},
		},
		{
			name:      "creative_id",
			extra:     []string{"creative_id", "30"},
			wantQuery: `(?s)WHERE r\.active="Yes" AND r\.creative_id=\?`,
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
				"creative_id", "size_id", "iurl", "item_click", "imp_url", "click_url", "foreign_id", "creative_name", "content",
			}).AddRow(100, 300250, nil, nil, nil, nil, nil, "creative", "<div>ad</div>")
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
