package trafficquality

import "expvar"

var (
	metricDecisions       = expvar.NewInt("aofei_quality_decisions_total")
	metricMatched         = expvar.NewInt("aofei_quality_matched_total")
	metricObserve         = expvar.NewInt("aofei_quality_action_observe_total")
	metricFlag            = expvar.NewInt("aofei_quality_action_flag_total")
	metricThrottle        = expvar.NewInt("aofei_quality_action_throttle_total")
	metricReject          = expvar.NewInt("aofei_quality_action_reject_total")
	metricQuarantine      = expvar.NewInt("aofei_quality_action_quarantine_total")
	metricDependencyError = expvar.NewInt("aofei_quality_dependency_error_total")
	metricRollback        = expvar.NewInt("aofei_quality_rollback_total")
)

func recordDecisionMetric(decision Decision) {
	metricDecisions.Add(1)
	if decision.Matched {
		metricMatched.Add(1)
	}
	switch decision.AppliedAction {
	case ActionFlag:
		metricFlag.Add(1)
	case ActionThrottle:
		metricThrottle.Add(1)
	case ActionReject:
		metricReject.Add(1)
	case ActionQuarantine:
		metricQuarantine.Add(1)
	default:
		metricObserve.Add(1)
	}
}
