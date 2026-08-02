package dsp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/guruperl/aofei/trafficquality"
	"github.com/prebid/openrtb/v20/openrtb2"
	"go.uber.org/zap"
)

const qualityRefreshTimeout = 2 * time.Second

func (self *Controller) trafficQualityEnabled() bool {
	return self != nil && self.qualityService != nil && self.C != nil && self.C.TrafficQuality.Enabled
}

func (self *Controller) startQualityEnforcementRefresh() {
	if self == nil || self.qualityService == nil || self.C == nil || !self.C.TrafficQuality.Enabled {
		return
	}
	self.qualityReloadMu.Lock()
	defer self.qualityReloadMu.Unlock()
	if self.qualityReloadCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	self.qualityReloadCancel = cancel
	self.qualityReloadDone = done
	interval := time.Duration(self.C.TrafficQuality.EnforcementRefreshSeconds) * time.Second
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				loadCtx, loadCancel := context.WithTimeout(context.Background(), qualityRefreshTimeout)
				snapshot, err := self.qualityService.LoadEnforcementSnapshot(loadCtx)
				loadCancel()
				if err != nil {
					metricQualityRefreshErrors.Add(1)
					if self.Logger != nil {
						self.Logger.Warn("traffic-quality enforcement refresh failed", zap.Error(err))
					}
					continue
				}
				self.qualitySnapshot.Store(snapshot)
				metricQualityRefreshes.Add(1)
			}
		}
	}()
}

func (self *Controller) stopQualityEnforcementRefresh() {
	if self == nil {
		return
	}
	self.qualityReloadMu.Lock()
	cancel, done := self.qualityReloadCancel, self.qualityReloadDone
	self.qualityReloadCancel, self.qualityReloadDone = nil, nil
	self.qualityReloadMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (self *Controller) qualityAction(scope trafficquality.Scope, eventKey string, now time.Time) (trafficquality.Action, uint64) {
	if !self.trafficQualityEnabled() {
		return trafficquality.ActionObserve, 0
	}
	maxAge := time.Duration(self.C.TrafficQuality.EnforcementMaxAgeSeconds) * time.Second
	action, id, err := self.qualityService.EnforcementAction(self.qualitySnapshot.Load(), scope, eventKey, now, maxAge)
	if err != nil {
		metricQualityEvaluationErrors.Add(1)
		return trafficquality.ActionObserve, 0
	}
	if action != trafficquality.ActionObserve {
		recordQualityEnforcement(action)
	}
	return action, id
}

func qualityEventKey(bid *openrtb2.BidRequest, impIndex int) string {
	if bid == nil || impIndex < 0 || impIndex >= len(bid.Imp) {
		return "invalid:0"
	}
	digest := sha256.Sum256([]byte("w8m-quality-request-v1\n" + bid.ID + "\n" + bid.Imp[impIndex].ID))
	return "request:" + hex.EncodeToString(digest[:])
}

func qualityPartnerEventKey(bid *openrtb2.BidRequest, impIndex int, partnerID uint32) string {
	return qualityEventKey(bid, impIndex) + ":partner:" + strconv.FormatUint(uint64(partnerID), 10)
}

func qualityBlocks(action trafficquality.Action) bool {
	return action == trafficquality.ActionThrottle || action == trafficquality.ActionReject || action == trafficquality.ActionQuarantine
}

func recordQualityEnforcement(action trafficquality.Action) {
	switch action {
	case trafficquality.ActionThrottle:
		metricQualityThrottle.Add(1)
	case trafficquality.ActionReject:
		metricQualityReject.Add(1)
	case trafficquality.ActionQuarantine:
		metricQualityQuarantine.Add(1)
	}
}
