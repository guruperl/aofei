package dsp

import (
	"context"
	"time"

	"github.com/guruperl/aofei/publisherauth"
	"go.uber.org/zap"
)

const publisherAuthRefreshTimeout = 2 * time.Second

// PublisherAuthService exposes the controller-owned lifecycle service to the
// S02-protected Summer control plane. It returns nil while the feature is off.
func (self *Controller) PublisherAuthService() *publisherauth.Service {
	if self == nil {
		return nil
	}
	return self.publisherAuth
}

func (self *Controller) directSSPAuthRequired() bool {
	return self != nil && self.C != nil && self.C.DirectSSPAuth.Enabled
}

func (self *Controller) startPublisherAuthRefresh() {
	if !self.directSSPAuthRequired() || self.publisherAuth == nil {
		return
	}
	self.publisherAuthReloadMu.Lock()
	defer self.publisherAuthReloadMu.Unlock()
	if self.publisherAuthCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	self.publisherAuthCancel = cancel
	self.publisherAuthReloadDone = done
	interval := time.Duration(self.C.DirectSSPAuth.CredentialRefreshSeconds) * time.Second
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				loadCtx, loadCancel := context.WithTimeout(context.Background(), publisherAuthRefreshTimeout)
				err := self.publisherAuth.ReloadSnapshot(loadCtx)
				loadCancel()
				if err != nil {
					metricSSPPublisherAuthRefreshErrors.Add(1)
					if self.Logger != nil {
						self.Logger.Warn("direct SSP credential refresh failed", zap.Error(err))
					}
					continue
				}
				metricSSPPublisherAuthRefreshes.Add(1)
				metricSSPPublisherAuthLoadedAt.Set(self.publisherAuth.SnapshotGeneratedAt().Unix())
			}
		}
	}()
}

func (self *Controller) stopPublisherAuthRefresh() {
	if self == nil {
		return
	}
	self.publisherAuthReloadMu.Lock()
	cancel, done := self.publisherAuthCancel, self.publisherAuthReloadDone
	self.publisherAuthCancel, self.publisherAuthReloadDone = nil, nil
	self.publisherAuthReloadMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}
