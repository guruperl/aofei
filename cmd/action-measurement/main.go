// Command action-measurement runs bounded R01 reconciliation, retention,
// pseudonym export, and pseudonym deletion operations.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/internal/cmdboot"
	actionjob "github.com/guruperl/aofei/internal/jobs/action"
)

var (
	configFile string
	operation  string
	pseudonym  string
	limit      int
)

func init() {
	flag.StringVar(&configFile, "s", os.Getenv("AOFEI"), "DSP config file")
	flag.StringVar(&operation, "action", "reconcile", "reconcile, prune, export, or delete")
	flag.StringVar(&pseudonym, "pseudonym", "", "64-character R01 action pseudonym for export or deletion")
	flag.IntVar(&limit, "limit", 1000, "bounded reconcile/prune batch size")
}

func main() {
	flag.Parse()
	ctx, stop := cmdboot.SignalContext(context.Background())
	defer stop()
	config, err := dsp.NewConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}
	db, err := config.OpenDB(ctx, dsp.ConfigModeRetry)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	service := actionjob.Service{DB: db}
	switch operation {
	case "reconcile":
		updated, err := service.Reconcile(ctx, time.Duration(config.ActionClickWindowHours)*time.Hour, time.Duration(config.ActionViewWindowHours)*time.Hour, limit)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("reconciled_actions=%d\n", updated)
	case "prune":
		actions, touches, err := service.Prune(ctx, limit)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("pruned_actions=%d pruned_touches=%d\n", actions, touches)
	case "export":
		if err := service.ExportPseudonym(ctx, pseudonym, os.Stdout); err != nil {
			log.Fatal(err)
		}
	case "delete":
		actions, touches, err := service.DeletePseudonym(ctx, pseudonym)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("deleted_actions=%d deleted_touches=%d\n", actions, touches)
	default:
		log.Fatalf("unsupported action %q", operation)
	}
}
