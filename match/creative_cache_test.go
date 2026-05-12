package match

import (
	"os"
	"path/filepath"
	"testing"
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
