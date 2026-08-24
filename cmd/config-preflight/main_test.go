package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRejectsCheckedInExampleTrackingSecret(t *testing.T) {
	configFile := filepath.Join("..", "..", "etc", "aofei.json")
	var output bytes.Buffer
	err := run(configFile, &output)
	if err == nil || !strings.Contains(err.Error(), "checked-in public example") {
		t.Fatalf("checked-in example preflight error = %v", err)
	}
	if strings.Contains(err.Error(), "local-dev-tracking-secret") || output.Len() != 0 {
		t.Fatalf("preflight leaked a secret or emitted success: error=%v output=%q", err, output.String())
	}
}

func TestRunAcceptsStrongDeploymentTrackingSecret(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "aofei.json")
	content := `{
		"server_url": "https://ads.example.test",
		"connect_array": ["mysql", "service:secret@tcp(db.example.test:3306)/aofei"],
		"nats_url": "nats://nats.example.test:4222"
	}`
	if err := os.WriteFile(configFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRACKING_SECRET", strings.Repeat("deployment-key-material-", 2))
	var output bytes.Buffer
	if err := run(configFile, &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "production_config_preflight=passed\n" {
		t.Fatalf("preflight output = %q", output.String())
	}
}
