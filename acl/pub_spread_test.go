package acl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nats-io/nats.go"
)

func TestSpreadGetPubUsesValidatedAtomicSnapshot(t *testing.T) {
	top := t.TempDir()
	message := &nats.Msg{Subject: HashNamePubmap + ":publisher.example", Data: []byte("snapshot")}
	if err := SpreadGetPub(message, top); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(top, HashNamePubmap, "publisher.example")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "snapshot" {
		t.Fatalf("snapshot = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0640 {
		t.Fatalf("snapshot mode = %04o, want 0640", info.Mode().Perm())
	}
	if err := SpreadGetPub(&nats.Msg{Subject: HashNamePubmap + ":../escape", Data: []byte("unsafe")}, top); err == nil {
		t.Fatal("unsafe spread publisher subject was accepted")
	}
}
