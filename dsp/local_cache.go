package dsp

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/match"
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

func (c *localStaticCache) load(top string) (int, error) {
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
