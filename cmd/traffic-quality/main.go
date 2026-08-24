// Command traffic-quality ingests bounded aggregate signal windows and runs
// restricted maintenance/health operations for S03. It is not an HTTP service
// and never accepts raw request payloads or identity evidence.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/internal/cmdboot"
	quality "github.com/guruperl/aofei/trafficquality"
)

const maxWindowBodyBytes = 64 << 10

type options struct {
	config     string
	action     string
	reason     string
	limit      int
	sinceHours int
}

type serviceAPI interface {
	AssessWindow(context.Context, quality.Window) ([]quality.Decision, error)
	PruneEvidence(context.Context, quality.Actor, int, string) (int64, error)
	RuleHealth(context.Context, quality.Actor, time.Time) ([]quality.RuleHealth, error)
}

type decisionOutput struct {
	ID                 uint64                     `json:"decision_id"`
	RuleKey            string                     `json:"rule_key"`
	RuleVersion        uint32                     `json:"rule_version"`
	Signal             quality.Signal             `json:"signal"`
	AppliedAction      quality.Action             `json:"applied_action"`
	Scope              quality.Scope              `json:"scope"`
	Evidence           quality.EvidenceState      `json:"evidence_state"`
	ReasonCode         string                     `json:"reason_code"`
	BillingDisposition quality.BillingDisposition `json:"billing_disposition"`
}

func main() {
	var opts options
	flag.StringVar(&opts.config, "s", os.Getenv("AOFEI"), "Aofei configuration path")
	flag.StringVar(&opts.action, "action", "", "assess-window, health, or prune-evidence")
	flag.StringVar(&opts.reason, "reason", "", "required single-line audit reason")
	flag.IntVar(&opts.limit, "limit", 1000, "bounded evidence prune limit")
	flag.IntVar(&opts.sinceHours, "since-hours", 24, "bounded rule-health lookback")
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
	service, err := quality.NewService(config.TrafficQuality, db)
	if err != nil {
		log.Fatal(err)
	}
	if service == nil {
		log.Fatal("traffic_quality.enabled must be true")
	}
	if err := run(ctx, service, opts, os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, service serviceAPI, opts options, input io.Reader, output io.Writer) error {
	if service == nil {
		return fmt.Errorf("traffic-quality service is required")
	}
	switch opts.action {
	case "assess-window":
		window, err := decodeWindow(input)
		if err != nil {
			return err
		}
		decisions, err := service.AssessWindow(ctx, window)
		if err != nil {
			return err
		}
		out := make([]decisionOutput, 0, len(decisions))
		for _, decision := range decisions {
			out = append(out, decisionOutput{
				ID: decision.ID, RuleKey: decision.RuleKey, RuleVersion: decision.RuleVersion,
				Signal: decision.Signal, AppliedAction: decision.AppliedAction,
				Scope: decision.Scope, Evidence: decision.Evidence,
				ReasonCode: decision.ReasonCode, BillingDisposition: decision.BillingDisposition,
			})
		}
		return json.NewEncoder(output).Encode(out)
	case "health":
		actor, err := maintenanceActor(os.Geteuid(), quality.PermissionEvidenceRead)
		if err != nil {
			return err
		}
		if opts.sinceHours < 1 || opts.sinceHours > 400*24 {
			return fmt.Errorf("since-hours must be between 1 and 9600")
		}
		health, err := service.RuleHealth(ctx, actor, time.Now().UTC().Add(-time.Duration(opts.sinceHours)*time.Hour))
		if err != nil {
			return err
		}
		return json.NewEncoder(output).Encode(health)
	case "prune-evidence":
		actor, err := maintenanceActor(os.Geteuid(), quality.PermissionRetentionPrune)
		if err != nil {
			return err
		}
		deleted, err := service.PruneEvidence(ctx, actor, opts.limit, opts.reason)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(output, "pruned traffic_quality_evidence_rows=%d\n", deleted)
		return err
	default:
		return fmt.Errorf("unsupported -action %q", opts.action)
	}
}

func decodeWindow(input io.Reader) (quality.Window, error) {
	var window quality.Window
	if input == nil {
		return window, fmt.Errorf("aggregate window input is required")
	}
	raw, err := io.ReadAll(io.LimitReader(input, maxWindowBodyBytes+1))
	if err != nil {
		return window, err
	}
	if len(raw) > maxWindowBodyBytes {
		return window, fmt.Errorf("aggregate window exceeds %d bytes", maxWindowBodyBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&window); err != nil {
		return window, fmt.Errorf("decode aggregate window: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return window, fmt.Errorf("aggregate window contains trailing data")
	}
	if err := window.Validate(); err != nil {
		return window, err
	}
	return window, nil
}

func maintenanceActor(euid int, permission string) (quality.Actor, error) {
	if euid < 0 || permission != quality.PermissionEvidenceRead && permission != quality.PermissionRetentionPrune {
		return quality.Actor{}, fmt.Errorf("effective Unix principal or maintenance permission is invalid")
	}
	return quality.Actor{
		Role: "admin", ID: fmt.Sprintf("unix-uid:%d", euid),
		Scope:       quality.Scope{Type: quality.ScopeGlobal},
		Permissions: map[string]bool{permission: true},
	}, nil
}
