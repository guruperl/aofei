// Command hosted-payment exposes aggregate health and bounded retention for
// A02 operations. It never accepts payment credentials or initiates money
// movement; funding, payout, refund, and exception transitions stay behind the
// S02-authenticated Summer service.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/guruperl/aofei/dsp"
	payment "github.com/guruperl/aofei/hostedpayment"
	"github.com/guruperl/aofei/internal/cmdboot"
)

type options struct {
	config  string
	action  string
	actorID string
	reason  string
	limit   int
}

type serviceAPI interface {
	OperationalHealth(context.Context) (payment.OperationalHealth, error)
	PruneEvents(context.Context, payment.Actor, int, string) (int64, error)
}

func main() {
	var opts options
	flag.StringVar(&opts.config, "s", os.Getenv("AOFEI"), "Aofei configuration path")
	flag.StringVar(&opts.action, "action", "health", "health or prune-events")
	flag.StringVar(&opts.actorID, "actor-admin-id", "", "administrator id used only for retention audit attribution")
	flag.StringVar(&opts.reason, "reason", "", "required bounded retention reason")
	flag.IntVar(&opts.limit, "limit", 1000, "provider-event prune batch size, 1-10000")
	flag.Parse()
	ctx, stop := cmdboot.SignalContext(context.Background())
	defer stop()
	config, err := dsp.NewConfig(opts.config)
	if err != nil {
		log.Fatal(err)
	}
	db, err := config.OpenDB(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	service, err := payment.NewMaintenanceService(config.HostedPayments, db)
	if err != nil {
		log.Fatal(err)
	}
	if service == nil {
		log.Fatal("hosted_payments.enabled must be true")
	}
	if err := run(ctx, service, opts, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, service serviceAPI, opts options, output io.Writer) error {
	if service == nil {
		return fmt.Errorf("hosted-payment service is required")
	}
	switch opts.action {
	case "health":
		health, err := service.OperationalHealth(ctx)
		if err != nil {
			return err
		}
		return json.NewEncoder(output).Encode(health)
	case "prune-events":
		id, err := strconv.ParseUint(opts.actorID, 10, 64)
		if err != nil || id == 0 {
			return fmt.Errorf("actor-admin-id must be a positive numeric administrator id")
		}
		actor := payment.Actor{Role: "admin", ID: strconv.FormatUint(id, 10), Permissions: map[string]bool{payment.PermissionReconcile: true}, RecentMFA: true}
		deleted, err := service.PruneEvents(ctx, actor, opts.limit, opts.reason)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(output, "pruned_hosted_event_rows=%d\n", deleted)
		return err
	default:
		return fmt.Errorf("unsupported -action %q", opts.action)
	}
}
