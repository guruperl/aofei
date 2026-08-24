package dsp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/internal/spreadcache"
	"github.com/guruperl/aofei/match"
	"go.uber.org/zap"
)

type localStaticCache struct {
	mu        sync.RWMutex
	pubmap    acl.PubMap
	pubByID   acl.DirectPubMap
	radvs     map[uint32]map[uint32]match.RAdvs
	audiences map[uint32]*match.Audience
	creatives map[uint32]*match.Creative
	loadedAt  time.Time
}

func newLocalStaticCache() *localStaticCache {
	return &localStaticCache{}
}

func (self *Controller) localStaticCache() *localStaticCache {
	self.localMu.Lock()
	defer self.localMu.Unlock()
	if self.local == nil {
		self.local = newLocalStaticCache()
	}
	return self.local
}

func (self *Controller) localPub(top, pubStr string) (*acl.Pub, error) {
	cache := self.localStaticCache()
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.pubmap[pubStr], nil
}

func (self *Controller) localPubByID(top string, pubID uint32) (*acl.DirectPub, error) {
	cache := self.localStaticCache()
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.pubByID.PubByID(pubID), nil
}

func (self *Controller) localRAdvs(top string, sizeID, slotID uint32) (match.RAdvs, error) {
	cache := self.localStaticCache()
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	if cache.radvs[sizeID] == nil {
		return nil, nil
	}
	return cache.radvs[sizeID][slotID], nil
}

func (self *Controller) localAudiences(top string, candidates match.RAdvs) (match.Audiences, error) {
	cache := self.localStaticCache()
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	audiences := make(match.Audiences, len(candidates))
	for i, candidate := range candidates {
		audiences[i] = cache.audiences[candidate.ItemID]
	}
	return audiences, nil
}

func (self *Controller) localCreative(top string, creativeID uint32) (*match.Creative, error) {
	cache := self.localStaticCache()
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	creative := cache.creatives[creativeID]
	if creative == nil {
		return nil, os.ErrNotExist
	}
	return creative, nil
}

func (self *Controller) ReloadLocalStaticCache() error {
	if self.C == nil {
		return fmt.Errorf("controller config is nil")
	}
	start := time.Now()
	count, err := self.localStaticCache().load(self.C.Spread)
	metricLocalCacheReloadMillis.Set(time.Since(start).Milliseconds())
	if err != nil {
		metricLocalCacheReloadErrors.Add(1)
		return err
	}
	metricLocalCacheReloads.Add(1)
	metricLocalCacheReloadEntries.Set(int64(count))
	self.publishLocalCacheFreshnessState()
	return nil
}

// StartLocalStaticCacheReload starts the bounded propagation loop used by
// local/spread mode. Calling it more than once is safe. Snapshot producers must
// still refresh the files before delivery_cache_max_age_seconds expires; this
// loop makes a newly published generation visible to the serving process.
func (self *Controller) StartLocalStaticCacheReload() {
	if self == nil || self.C == nil || !self.C.IsLocal {
		return
	}
	self.localReloadMu.Lock()
	defer self.localReloadMu.Unlock()
	if self.localReloadCancel != nil {
		return
	}
	interval := self.localStaticCacheReloadInterval()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	self.localReloadCancel = cancel
	self.localReloadDone = done
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := self.ReloadLocalStaticCache(); err != nil && self.Logger != nil {
					self.Logger.Error("local static cache reload failed", zap.Error(err))
				}
			}
		}
	}()
}

// StopLocalStaticCacheReload stops a loop previously started by
// StartLocalStaticCacheReload. It is safe when no loop is running.
func (self *Controller) StopLocalStaticCacheReload() {
	self.stopLocalStaticCacheReload()
}

func (self *Controller) localStaticCacheReloadInterval() time.Duration {
	if self.localReloadInterval > 0 {
		return self.localReloadInterval
	}
	maxAge := self.deliveryCacheMaxAge()
	if self.C != nil && self.C.LocalCacheMaxAgeSeconds > 0 {
		localMaxAge := time.Duration(self.C.LocalCacheMaxAgeSeconds) * time.Second
		if maxAge <= 0 || localMaxAge < maxAge {
			maxAge = localMaxAge
		}
	}
	interval := maxAge / 3
	if interval < time.Second {
		interval = time.Second
	}
	return interval
}

// ServingReadiness checks only process-local serving state. Shared dependency
// health remains on the protected metrics surface: withdrawing every node for
// one shared MySQL, Redis, or NATS outage would not provide useful failover.
func (self *Controller) ServingReadiness(now time.Time) error {
	if self == nil || self.C == nil {
		return fmt.Errorf("controller is not initialized")
	}
	if !self.C.IsLocal {
		return nil
	}
	self.localMu.Lock()
	cache := self.local
	self.localMu.Unlock()
	if cache == nil {
		return fmt.Errorf("local cache is not loaded")
	}
	cache.mu.RLock()
	loadedAt := cache.loadedAt
	cache.mu.RUnlock()
	if loadedAt.IsZero() {
		return fmt.Errorf("local cache is not loaded")
	}
	if loadedAt.After(now) {
		return fmt.Errorf("local cache timestamp is in the future")
	}
	maxAge := self.deliveryCacheMaxAge()
	if self.C.LocalCacheMaxAgeSeconds > 0 {
		localMaxAge := time.Duration(self.C.LocalCacheMaxAgeSeconds) * time.Second
		if maxAge <= 0 || localMaxAge < maxAge {
			maxAge = localMaxAge
		}
	}
	if maxAge > 0 && now.Sub(loadedAt) > maxAge {
		return fmt.Errorf("local cache is stale")
	}
	return nil
}

func (self *Controller) stopLocalStaticCacheReload() {
	if self == nil {
		return
	}
	self.localReloadMu.Lock()
	cancel := self.localReloadCancel
	done := self.localReloadDone
	self.localReloadCancel = nil
	self.localReloadDone = nil
	self.localReloadMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (c *localStaticCache) load(top string) (int, error) {
	var count int
	err := spreadcache.WithResolved(top, func(root string) error {
		var err error
		count, err = c.loadResolved(root)
		return err
	})
	return count, err
}

func (c *localStaticCache) loadResolved(top string) (int, error) {
	pubmap, err := acl.PubMapFromIO(top)
	if err != nil {
		return 0, err
	}
	audiences, err := match.AudienceMapFromIO(top)
	if err != nil && !os.IsNotExist(err) {
		return 0, err
	}
	creatives, err := match.CreativeMapFromIO(top)
	if err != nil && !os.IsNotExist(err) {
		return 0, err
	}
	radvs, err := localRAdvsFromIO(top)
	if err != nil {
		return 0, err
	}

	if pubmap == nil {
		pubmap = make(acl.PubMap)
	}
	if audiences == nil {
		audiences = make(map[uint32]*match.Audience)
	}
	if creatives == nil {
		creatives = make(map[uint32]*match.Creative)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pubmap = pubmap
	c.pubByID = acl.DirectPubMapFromPubMap(pubmap)
	c.audiences = audiences
	c.creatives = creatives
	c.radvs = radvs
	c.loadedAt = time.Now()
	count := len(pubmap) + len(audiences) + len(creatives)
	for _, bySlot := range radvs {
		count += len(bySlot)
	}
	return count, nil
}

func localRAdvsFromIO(top string) (map[uint32]map[uint32]match.RAdvs, error) {
	root := filepath.Join(top, match.HashNameSlot)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[uint32]map[uint32]match.RAdvs), nil
		}
		return nil, err
	}

	all := make(map[uint32]map[uint32]match.RAdvs)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sizeID64, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil {
			continue
		}
		sizeID := uint32(sizeID64)
		hash, err := match.RAdvsFromIOBySizeID(top, sizeID)
		if err != nil {
			return nil, fmt.Errorf("load RAdvs size %d: %w", sizeID, err)
		}
		if hash != nil {
			all[sizeID] = hash
		}
	}
	return all, nil
}

func (self *Controller) publishLocalCacheFreshnessState() {
	if self == nil || self.C == nil || !self.C.IsLocal {
		return
	}
	setLocalCacheFreshnessMetrics(self.localStaticCache().loadedAtTime(), self.C.LocalCacheMaxAgeSeconds)
}

func (c *localStaticCache) loadedAtTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.loadedAt
}
