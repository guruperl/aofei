package match

import "testing"

func TestAudiencesFromIOMissingFileIsWildcard(t *testing.T) {
	got, err := RAdvs{{Demand: Demand{ItemID: 404}}}.AudiencesFromIO(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0] != nil {
		t.Fatalf("missing audience = %+v, want nil wildcard", got[0])
	}
}
