// Command report-experiment manages R02 experiment definitions through an
// OS-principal operator boundary. It never changes campaign delivery or bids.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/internal/cmdboot"
	"github.com/guruperl/aofei/reporting"
)

var (
	configFile      string
	action          string
	experimentID    uint64
	owner           string
	advID           uint
	name            string
	version         uint
	primaryMetric   string
	guardrailMetric string
	retentionHours  uint
	startsAt        string
	endsAt          string
	variants        string
	reason          string
	pruneLimit      int
	subjectHash     string
)

func init() {
	flag.StringVar(&configFile, "s", os.Getenv("AOFEI"), "DSP config file")
	flag.StringVar(&action, "action", "list", "list, create, start, stop, complete, prune, or delete-subject")
	flag.Uint64Var(&experimentID, "experiment-id", 0, "experiment ID")
	flag.StringVar(&owner, "owner", "operator", "operator or advertiser")
	flag.UintVar(&advID, "adv-id", 0, "advertiser owner ID")
	flag.StringVar(&name, "name", "", "bounded experiment name")
	flag.UintVar(&version, "version", 1, "positive experiment version")
	flag.StringVar(&primaryMetric, "primary-metric", "", "R02 primary metric name")
	flag.StringVar(&guardrailMetric, "guardrail-metric", "", "R02 guardrail metric name")
	flag.UintVar(&retentionHours, "retention-hours", 2160, "pseudonymous exposure/outcome retention, 24-9600 hours")
	flag.StringVar(&startsAt, "starts-at", "", "RFC3339 start instant")
	flag.StringVar(&endsAt, "ends-at", "", "optional RFC3339 end instant")
	flag.StringVar(&variants, "variants", "", "comma-separated key=basis-points allocation")
	flag.StringVar(&reason, "reason", "", "required audit reason for mutations")
	flag.IntVar(&pruneLimit, "limit", 1000, "expired exposure prune batch size, 1-10000")
	flag.StringVar(&subjectHash, "subject-hash", "", "exact 32-byte experiment subject hash for authorized privacy deletion")
}

func main() {
	flag.Parse()
	ctx, stop := cmdboot.SignalContext(context.Background())
	defer stop()
	config, err := dsp.NewConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}
	if err := config.Validate(dsp.ConfigModeDatabase); err != nil {
		log.Fatal(err)
	}
	db, err := config.OpenDB(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := run(ctx, dbService{db: db}, uint64(os.Geteuid())); err != nil {
		log.Fatal(err)
	}
}

type experimentService interface {
	Create(context.Context, reporting.Experiment, uint64, string) (uint64, error)
	Transition(context.Context, uint64, string, uint64, string) error
	List(context.Context) ([]reporting.ExperimentSummary, error)
	Prune(context.Context, int) (int64, error)
	DeleteSubject(context.Context, uint64, uint32, string, uint64, string) (bool, error)
}

type dbService struct{ db *sql.DB }

func (service dbService) Create(ctx context.Context, experiment reporting.Experiment, actor uint64, reason string) (uint64, error) {
	return reporting.CreateExperiment(ctx, service.db, experiment, actor, reason)
}

func (service dbService) Transition(ctx context.Context, id uint64, target string, actor uint64, reason string) error {
	return reporting.TransitionExperiment(ctx, service.db, id, target, actor, reason)
}

func (service dbService) List(ctx context.Context) ([]reporting.ExperimentSummary, error) {
	return reporting.ListExperiments(ctx, service.db)
}

func (service dbService) Prune(ctx context.Context, limit int) (int64, error) {
	return reporting.PruneExpired(ctx, service.db, limit)
}

func (service dbService) DeleteSubject(ctx context.Context, id uint64, version uint32, hash string, actor uint64, reason string) (bool, error) {
	return reporting.DeleteSubject(ctx, service.db, id, version, hash, actor, reason)
}

func run(ctx context.Context, service experimentService, actor uint64) error {
	switch action {
	case "list":
		items, err := service.List(ctx)
		if err != nil {
			return err
		}
		for _, item := range items {
			ownerText := item.OwnerType
			if item.AdvID != nil {
				ownerText += ":" + strconv.FormatUint(uint64(*item.AdvID), 10)
			}
			end := ""
			if item.EndsAt != nil {
				end = item.EndsAt.Format(time.RFC3339)
			}
			fmt.Printf("experiment_id=%d version=%d assignment_algorithm=v%d status=%s owner=%s name=%q primary=%s guardrail=%s retention_hours=%d starts_at=%s ends_at=%s\n",
				item.ID, item.Version, item.AssignmentAlgorithmVersion, item.Status, ownerText, item.Name, item.PrimaryMetric,
				item.GuardrailMetric, item.RetentionHours, item.StartsAt.Format(time.RFC3339), end)
		}
		return nil
	case "create":
		experiment, err := experimentFromFlags()
		if err != nil {
			return err
		}
		id, err := service.Create(ctx, experiment, actor, reason)
		if err == nil {
			fmt.Printf("experiment_id=%d status=Draft\n", id)
		}
		return err
	case "start":
		return service.Transition(ctx, experimentID, "Running", actor, reason)
	case "stop":
		return service.Transition(ctx, experimentID, "Stopped", actor, reason)
	case "complete":
		return service.Transition(ctx, experimentID, "Completed", actor, reason)
	case "prune":
		deleted, err := service.Prune(ctx, pruneLimit)
		if err == nil {
			fmt.Printf("expired_exposures_deleted=%d\n", deleted)
		}
		return err
	case "delete-subject":
		if version == 0 || uint64(version) > uint64(^uint32(0)) {
			return fmt.Errorf("positive 32-bit version is required")
		}
		deleted, err := service.DeleteSubject(ctx, experimentID, uint32(version), subjectHash, actor, reason)
		if err == nil {
			fmt.Printf("experiment_subject_deleted=%t\n", deleted)
		}
		return err
	default:
		return fmt.Errorf("unknown -action %q", action)
	}
}

func experimentFromFlags() (reporting.Experiment, error) {
	if version == 0 || uint64(version) > uint64(^uint32(0)) {
		return reporting.Experiment{}, fmt.Errorf("positive 32-bit version is required")
	}
	if retentionHours < 24 || retentionHours > 9600 {
		return reporting.Experiment{}, fmt.Errorf("retention-hours must be between 24 and 9600")
	}
	start, err := time.Parse(time.RFC3339, startsAt)
	if err != nil {
		return reporting.Experiment{}, fmt.Errorf("starts-at must be RFC3339: %w", err)
	}
	var end *time.Time
	if endsAt != "" {
		value, err := time.Parse(time.RFC3339, endsAt)
		if err != nil {
			return reporting.Experiment{}, fmt.Errorf("ends-at must be RFC3339: %w", err)
		}
		value = value.UTC()
		end = &value
	}
	parsedVariants, err := parseVariants(variants)
	if err != nil {
		return reporting.Experiment{}, err
	}
	experiment := reporting.Experiment{
		Name: name, Version: uint32(version), Status: "Draft",
		PrimaryMetric: primaryMetric, GuardrailMetric: guardrailMetric,
		RetentionHours: uint32(retentionHours),
		StartsAt:       start.UTC(), EndsAt: end, Variants: parsedVariants,
	}
	switch owner {
	case "operator":
		experiment.OwnerType = "Operator"
	case "advertiser":
		if advID == 0 || uint64(advID) > uint64(^uint32(0)) {
			return reporting.Experiment{}, fmt.Errorf("positive 32-bit adv-id is required")
		}
		value := uint32(advID)
		experiment.OwnerType = "Advertiser"
		experiment.AdvID = &value
	default:
		return reporting.Experiment{}, fmt.Errorf("owner must be operator or advertiser")
	}
	return experiment, nil
}

func parseVariants(raw string) ([]reporting.Variant, error) {
	if raw == "" {
		return nil, fmt.Errorf("variants are required")
	}
	parts := strings.Split(raw, ",")
	out := make([]reporting.Variant, 0, len(parts))
	for _, part := range parts {
		pair := strings.SplitN(part, "=", 2)
		if len(pair) != 2 {
			return nil, fmt.Errorf("variant %q must use key=basis-points", part)
		}
		allocation, err := strconv.ParseUint(pair[1], 10, 16)
		if err != nil || allocation == 0 {
			return nil, fmt.Errorf("variant %q has an invalid allocation", pair[0])
		}
		out = append(out, reporting.Variant{Key: pair[0], AllocationBasisPts: uint16(allocation)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}
