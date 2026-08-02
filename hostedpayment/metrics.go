package hostedpayment

import "expvar"

var (
	metricProviderRequests         = expvar.NewInt("aofei_hosted_payment_provider_requests_total")
	metricProviderErrors           = expvar.NewInt("aofei_hosted_payment_provider_errors_total")
	metricWebhookRequests          = expvar.NewInt("aofei_hosted_payment_webhook_requests_total")
	metricWebhookInvalid           = expvar.NewInt("aofei_hosted_payment_webhook_invalid_total")
	metricWebhookErrors            = expvar.NewInt("aofei_hosted_payment_webhook_errors_total")
	metricWebhookDuplicates        = expvar.NewInt("aofei_hosted_payment_webhook_duplicates_total")
	metricWebhookReprocessed       = expvar.NewInt("aofei_hosted_payment_webhook_reprocessed_total")
	metricWebhookApplied           = expvar.NewInt("aofei_hosted_payment_webhook_applied_total")
	metricWebhookUnresolved        = expvar.NewInt("aofei_hosted_payment_webhook_unresolved_total")
	metricWebhookIgnored           = expvar.NewInt("aofei_hosted_payment_webhook_ignored_total")
	metricReconciliationUnresolved = expvar.NewInt("aofei_hosted_payment_reconciliation_unresolved_total")
)

func recordWebhookDisposition(result WebhookResult) {
	if result.Reprocessed {
		metricWebhookReprocessed.Add(1)
	}
	if result.Duplicate {
		metricWebhookDuplicates.Add(1)
		return
	}
	switch result.Disposition {
	case "Applied":
		metricWebhookApplied.Add(1)
	case "Unresolved":
		metricWebhookUnresolved.Add(1)
	case "Ignored":
		metricWebhookIgnored.Add(1)
	}
}
