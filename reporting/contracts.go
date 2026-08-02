// Package reporting defines marketplace analytics and controlled experiment
// contracts. Reporting facts are derived views; they never mutate auction,
// delivery-reservation, measurement, or accounting state.
package reporting

import (
	"fmt"
	"sort"
)

// MetricContract fixes the source, formula, scope, currency, timezone, and
// freshness semantics of one public report metric.
type MetricContract struct {
	Name      string
	Source    string
	Formula   string
	Scopes    []string
	Currency  string
	Timezone  string
	Freshness string
}

var metricContracts = []MetricContract{
	{Name: "impressions", Source: "report_delivery.imps", Formula: "SUM(imps)", Scopes: []string{"advertiser", "publisher", "operator"}, Currency: "none", Timezone: "UTC", Freshness: "interval reporting fact"},
	{Name: "clicks", Source: "report_delivery.clis", Formula: "SUM(clis)", Scopes: []string{"advertiser", "publisher", "operator"}, Currency: "none", Timezone: "UTC", Freshness: "interval reporting fact"},
	{Name: "ctr", Source: "report_delivery", Formula: "clicks / impressions; zero when impressions is zero", Scopes: []string{"advertiser", "publisher", "operator"}, Currency: "ratio", Timezone: "UTC", Freshness: "interval reporting fact"},
	{Name: "actions", Source: "measurement_action", Formula: "COUNT(*)", Scopes: []string{"advertiser", "operator"}, Currency: "none", Timezone: "UTC", Freshness: "action receipt and retention window"},
	{Name: "cvr", Source: "measurement_action + report_delivery", Formula: "actions / clicks; zero when clicks is zero", Scopes: []string{"advertiser", "operator"}, Currency: "ratio", Timezone: "UTC", Freshness: "partial when action or delivery input is unavailable"},
	{Name: "spend", Source: "report_delivery.spend_usd", Formula: "SUM(spend_usd); already per-impression USD", Scopes: []string{"advertiser", "operator"}, Currency: "USD", Timezone: "UTC", Freshness: "interval reporting fact"},
	{Name: "revenue", Source: "report_delivery.revenue_usd", Formula: "SUM(revenue_usd); already per-impression USD", Scopes: []string{"publisher", "operator"}, Currency: "USD", Timezone: "UTC", Freshness: "interval reporting fact"},
	{Name: "cost", Source: "report_delivery.cost_usd", Formula: "SUM(cost_usd); already per-impression USD", Scopes: []string{"operator"}, Currency: "USD", Timezone: "UTC", Freshness: "partial while middleman callback/retry is unresolved"},
	{Name: "margin", Source: "report_delivery.margin_usd", Formula: "SUM(margin_usd); never below zero", Scopes: []string{"operator"}, Currency: "USD", Timezone: "UTC", Freshness: "partial while middleman callback/retry is unresolved"},
	{Name: "roi", Source: "measurement_action + report_delivery", Formula: "(purchase_value_usd - spend) / spend; zero when spend is zero", Scopes: []string{"advertiser", "operator"}, Currency: "ratio", Timezone: "UTC", Freshness: "partial when action or delivery input is unavailable"},
	{Name: "roas", Source: "measurement_action + report_delivery", Formula: "purchase_value_usd / spend; zero when spend is zero", Scopes: []string{"advertiser", "operator"}, Currency: "ratio", Timezone: "UTC", Freshness: "partial when action or delivery input is unavailable"},
	{Name: "downstream_cpm", Source: "ledger_mid/report_delivery cost side", Formula: "downstream bid CPM before W8M margin; never derived from stored USD amount", Scopes: []string{"operator"}, Currency: "USD CPM", Timezone: "UTC", Freshness: "auction/callback evidence"},
	{Name: "returned_cpm", Source: "ledger_mid/report_delivery charge side", Formula: "upstream returned USD CPM after margin; never derived from stored USD amount", Scopes: []string{"operator"}, Currency: "USD CPM", Timezone: "UTC", Freshness: "auction/callback evidence"},
}

// MetricContracts returns a copy sorted by stable metric name.
func MetricContracts() []MetricContract {
	out := make([]MetricContract, len(metricContracts))
	copy(out, metricContracts)
	for i := range out {
		out[i].Scopes = append([]string(nil), out[i].Scopes...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ValidateMetricContracts rejects incomplete, duplicate, or unsupported
// contracts. It is used by tests and documentation/command callers.
func ValidateMetricContracts(contracts []MetricContract) error {
	seen := make(map[string]struct{}, len(contracts))
	for _, contract := range contracts {
		if contract.Name == "" || contract.Source == "" || contract.Formula == "" || len(contract.Scopes) == 0 || contract.Currency == "" || contract.Timezone != "UTC" || contract.Freshness == "" {
			return fmt.Errorf("report metric %q has an incomplete contract", contract.Name)
		}
		if _, exists := seen[contract.Name]; exists {
			return fmt.Errorf("report metric %q is duplicated", contract.Name)
		}
		seen[contract.Name] = struct{}{}
		for _, scope := range contract.Scopes {
			switch scope {
			case "advertiser", "publisher", "operator":
			default:
				return fmt.Errorf("report metric %q has unsupported scope %q", contract.Name, scope)
			}
		}
	}
	return nil
}

// Ratios derives analytical ratios without changing their source facts.
type Ratios struct {
	CTR  float64
	CVR  float64
	ROI  float64
	ROAS float64
}

// DeriveRatios returns deterministic zero-denominator semantics.
func DeriveRatios(impressions, clicks, actions uint64, spendUSD, purchaseValueUSD float64) Ratios {
	var out Ratios
	if impressions != 0 {
		out.CTR = float64(clicks) / float64(impressions)
	}
	if clicks != 0 {
		out.CVR = float64(actions) / float64(clicks)
	}
	if spendUSD != 0 {
		out.ROI = (purchaseValueUSD - spendUSD) / spendUSD
		out.ROAS = purchaseValueUSD / spendUSD
	}
	return out
}
