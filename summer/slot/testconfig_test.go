package slot

import (
	"errors"
	"os"
	"testing"

	"github.com/genelet/winter/genelet"
)

func testSummerConfig(t *testing.T) *genelet.Config {
	t.Helper()

	configPath := os.Getenv("SUMMER")
	if configPath == "" {
		configPath = "../../etc/summer.local.json"
	}
	if _, err := os.Stat(configPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("SUMMER config %s is missing; run ./scripts/aofei-local.sh up", configPath)
		}
		t.Fatal(err)
	}
	c, err := genelet.NewConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
