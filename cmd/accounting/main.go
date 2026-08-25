// Command accounting performs A01 manual statement, adjustment, settlement,
// correction, reconciliation, and CSV export operations.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/guruperl/aofei/accounting"
	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/internal/cmdboot"
)

var (
	configFile  string
	action      string
	party       string
	partyID     uint64
	statementID uint64
	cadence     string
	periodStart string
	periodEnd   string
	requestKey  string
	toStatus    string
	amount      string
	actor       string
	reason      string
	reference   string
	allParties  bool
)

func init() {
	flag.StringVar(&configFile, "s", os.Getenv("AOFEI"), "DSP config file")
	flag.StringVar(&action, "action", "", "create, transition, adjust, correct, reconcile, reconcile-middleman, or export")
	flag.StringVar(&party, "party", "", "advertiser or publisher")
	flag.Uint64Var(&partyID, "party-id", 0, "advertiser or publisher ID")
	flag.Uint64Var(&statementID, "statement-id", 0, "statement ID")
	flag.StringVar(&cadence, "cadence", "", "daily, weekly, or monthly")
	flag.StringVar(&periodStart, "from", "", "inclusive UTC date (YYYY-MM-DD)")
	flag.StringVar(&periodEnd, "to", "", "inclusive UTC date (YYYY-MM-DD)")
	flag.StringVar(&requestKey, "request-key", "", "unique idempotency key")
	flag.StringVar(&toStatus, "status", "", "target statement status")
	flag.StringVar(&amount, "amount", "", "signed USD adjustment with at most six decimals")
	flag.StringVar(&reason, "reason", "", "required audit reason")
	flag.StringVar(&reference, "reference", "", "opaque invoice:, payout:, or manual: evidence reference")
	flag.BoolVar(&allParties, "all-parties", false, "explicitly export every party (offline operator only)")
}

func main() {
	flag.Parse()
	var err error
	actor, err = effectiveActor()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := cmdboot.SignalContext(context.Background())
	defer stop()
	config, err := dsp.NewConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}
	db, err := config.OpenDB(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	service := accounting.Service{DB: db}
	if err := run(ctx, service); err != nil {
		log.Fatal(err)
	}
}

func effectiveActor() (string, error) {
	uid := os.Geteuid()
	if uid < 0 {
		return "", fmt.Errorf("resolve effective accounting operator: invalid uid")
	}
	return "unix-uid:" + strconv.Itoa(uid), nil
}

func run(ctx context.Context, service accounting.Service) error {
	switch action {
	case "create":
		start, err := parseDate(periodStart)
		if err != nil {
			return err
		}
		end, err := parseDate(periodEnd)
		if err != nil {
			return err
		}
		id, err := service.CreateStatement(ctx, accounting.CreateInput{
			RequestKey: requestKey, PartyType: accounting.PartyType(party), PartyID: partyID,
			Cadence: accounting.Cadence(cadence), PeriodStart: start, PeriodEnd: end,
			Actor: actor, Reason: reason,
		})
		if err == nil {
			fmt.Printf("statement_id=%d\n", id)
		}
		return err
	case "transition":
		return service.Transition(ctx, accounting.TransitionInput{
			StatementID: statementID, To: accounting.Status(toStatus), Actor: actor,
			Reason: reason, ExternalRef: reference,
		})
	case "adjust":
		value, err := accounting.ParseMoney(amount)
		if err != nil {
			return err
		}
		return service.AddAdjustment(ctx, accounting.AdjustmentInput{
			StatementID: statementID, Amount: value, Actor: actor, Reason: reason,
		})
	case "correct":
		id, err := service.Correct(ctx, accounting.CorrectionInput{
			StatementID: statementID, RequestKey: requestKey, Actor: actor, Reason: reason,
		})
		if err == nil {
			fmt.Printf("replacement_statement_id=%d\n", id)
		}
		return err
	case "reconcile":
		result, err := service.ReconcileStatement(ctx, statementID)
		fmt.Printf("statement_id=%d expected=%s actual=%s difference=%s\n",
			result.StatementID, result.Expected, result.Actual, result.Difference)
		return err
	case "reconcile-middleman":
		start, err := parseDate(periodStart)
		if err != nil {
			return err
		}
		end, err := parseDate(periodEnd)
		if err != nil {
			return err
		}
		result, err := service.ReconcileMiddleman(ctx, start, end)
		fmt.Printf("from=%s to=%s currency=%s impressions=%d charge=%s pay=%s margin=%s expected_margin=%s difference=%s charge_exact=%s pay_exact=%s margin_exact=%s expected_exact=%s difference_exact=%s\n",
			result.PeriodStart.Format("2006-01-02"), result.PeriodEnd.Format("2006-01-02"),
			result.Currency, result.Impressions, result.Charge, result.Pay, result.Margin,
			result.ExpectedMargin, result.Difference, result.ChargeExact, result.PayExact,
			result.MarginExact, result.ExpectedExact, result.DifferenceExact)
		return err
	case "export":
		var scope accounting.StatementScope
		if allParties {
			if party != "" || partyID != 0 {
				return fmt.Errorf("-all-parties cannot be combined with -party or -party-id")
			}
			scope = accounting.AllStatementScope()
		} else {
			scope = accounting.PartyStatementScope(accounting.PartyType(party), partyID)
		}
		statements, err := service.ListStatements(ctx, scope)
		if err != nil {
			return err
		}
		return accounting.WriteCSV(os.Stdout, statements)
	default:
		return fmt.Errorf("unknown -action %q", action)
	}
}

func parseDate(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, fmt.Errorf("date is required")
	}
	return time.Parse("2006-01-02", raw)
}
