package weight

import (
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/genelet/winter/genelet"
	_ "github.com/go-sql-driver/mysql"
)

func openSummerTestDB(t *testing.T) *sql.DB {
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
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("configured DB is unavailable: %v", err)
	}
	return db
}
