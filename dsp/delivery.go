package dsp

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/guruperl/aofei/match"
	"github.com/mediocregopher/radix/v4"
)

var errDeliveryLimit = errors.New("delivery limit reached")

const deliveryOperationTimeout = 2 * time.Second

// auctionCPMToSpend converts an OpenRTB USD CPM price to the USD amount for
// one billable impression. Bid and tracking payloads remain expressed in CPM.
func auctionCPMToSpend(cpm float32) float32 {
	return cpm / 1000
}

type deliveryReservationBalance struct {
	key      string
	balance  match.DeliveryBalance
	pacing   uint8
	fraction float64
	ttl      int
}

var reserveDeliveryScript = radix.NewEvalScript(`
if redis.call("EXISTS", KEYS[1]) ~= 0 then
  return -1
end
local cost = tonumber(ARGV[1])
local imp = 1
local reservation_ttl = tonumber(ARGV[2])
local offset = 3
for i=2,#KEYS do
  local base_spend = tonumber(ARGV[offset+3])
  local base_imp = tonumber(ARGV[offset+4])
  local base_click = tonumber(ARGV[offset+5])
  local floor_spend = tonumber(redis.call("HGET", KEYS[i], "floor_spend") or "0")
  local floor_imp = tonumber(redis.call("HGET", KEYS[i], "floor_imp") or "0")
  local floor_click = tonumber(redis.call("HGET", KEYS[i], "floor_click") or "0")
  if base_spend > floor_spend then floor_spend = base_spend end
  if base_imp > floor_imp then floor_imp = base_imp end
  if base_click > floor_click then floor_click = base_click end
  local used_spend = tonumber(redis.call("HGET", KEYS[i], "used_spend") or "0")
  local used_imp = tonumber(redis.call("HGET", KEYS[i], "used_imp") or "0")
  local used_click = tonumber(redis.call("HGET", KEYS[i], "used_click") or "0")
  if floor_spend > used_spend then used_spend = floor_spend end
  if floor_imp > used_imp then used_imp = floor_imp end
  if floor_click > used_click then used_click = floor_click end
  redis.call("HSET", KEYS[i],
    "used_spend", used_spend, "used_imp", used_imp, "used_click", used_click,
    "floor_spend", floor_spend, "floor_imp", floor_imp, "floor_click", floor_click)
  local state_ttl = tonumber(ARGV[offset+8])
  local ttl = redis.call("TTL", KEYS[i])
  if state_ttl == 0 then
    if ttl >= 0 then redis.call("PERSIST", KEYS[i]) end
  elseif ttl < state_ttl then
    redis.call("EXPIRE", KEYS[i], state_ttl)
  end
  offset = offset + 9
end

offset = 3
for i=2,#KEYS do
  local limit_spend = tonumber(ARGV[offset])
  local limit_imp = tonumber(ARGV[offset+1])
  local limit_click = tonumber(ARGV[offset+2])
  local even = tonumber(ARGV[offset+6])
  local fraction = tonumber(ARGV[offset+7])
  local used_spend = tonumber(redis.call("HGET", KEYS[i], "used_spend") or "0")
  local used_imp = tonumber(redis.call("HGET", KEYS[i], "used_imp") or "0")
  local used_click = tonumber(redis.call("HGET", KEYS[i], "used_click") or "0")
  if (limit_spend > 0 and (used_spend >= limit_spend or used_spend + cost > limit_spend)) or
     (limit_imp > 0 and (used_imp >= limit_imp or used_imp + imp > limit_imp)) or
     (limit_click > 0 and used_click >= limit_click) then
    return 0
  end
  if even == 1 then
    local allowed_spend = limit_spend * fraction
    local allowed_imp = limit_imp * fraction
    if allowed_spend < cost then allowed_spend = cost end
    if allowed_imp < imp then allowed_imp = imp end
    if (limit_spend > 0 and used_spend + cost > allowed_spend) or
       (limit_imp > 0 and used_imp + imp > allowed_imp) then
      return 0
    end
  end
  offset = offset + 9
end

offset = 3
redis.call("HSET", KEYS[1], "status", "active", "cost", ARGV[1], "count", #KEYS-1)
redis.call("EXPIRE", KEYS[1], reservation_ttl)
for i=2,#KEYS do
  redis.call("HINCRBYFLOAT", KEYS[i], "used_spend", cost)
  redis.call("HINCRBY", KEYS[i], "used_imp", imp)
  local state_ttl = tonumber(ARGV[offset+8])
	redis.call("HSET", KEYS[1], "key:" .. (i-1), KEYS[i], "ttl:" .. (i-1), state_ttl)
	offset = offset + 9
end
return 1`)

var releaseDeliveryScript = radix.NewEvalScript(`
if redis.call("HGET", KEYS[1], "status") ~= "active" then
  return 0
end
local cost = tonumber(redis.call("HGET", KEYS[1], "cost") or "0")
local count = tonumber(redis.call("HGET", KEYS[1], "count") or "0")
for i=1,count do
  local key = redis.call("HGET", KEYS[1], "key:" .. i)
	if key then
	  local spend = tonumber(redis.call("HGET", key, "used_spend") or "0") - cost
	  local imp = tonumber(redis.call("HGET", key, "used_imp") or "0") - 1
	  local floor_spend = tonumber(redis.call("HGET", key, "floor_spend") or "0")
	  local floor_imp = tonumber(redis.call("HGET", key, "floor_imp") or "0")
	  if spend < floor_spend then spend = floor_spend end
	  if imp < floor_imp then imp = floor_imp end
    redis.call("HSET", key, "used_spend", spend, "used_imp", imp)
  end
end
redis.call("DEL", KEYS[1])
return 1`)

var finalizeDeliveryScript = radix.NewEvalScript(`
if redis.call("HGET", KEYS[1], "status") ~= "active" then
  return 0
end
redis.call("HSET", KEYS[1], "status", "final")
redis.call("EXPIREAT", KEYS[1], ARGV[1])
return 1`)

var clickDeliveryScript = radix.NewEvalScript(`
local status = redis.call("HGET", KEYS[1], "status")
if not status then return 0 end
if redis.call("HGET", KEYS[1], "click") == "1" then return 0 end
local count = tonumber(redis.call("HGET", KEYS[1], "count") or "0")
for i=1,count do
  local key = redis.call("HGET", KEYS[1], "key:" .. i)
  if key then
    redis.call("HINCRBY", key, "used_click", 1)
    local state_ttl = tonumber(redis.call("HGET", KEYS[1], "ttl:" .. i) or "0")
    local ttl = redis.call("TTL", key)
    if state_ttl > 0 and ttl < state_ttl then redis.call("EXPIRE", key, state_ttl) end
  end
end
redis.call("HSET", KEYS[1], "click", "1")
return 1`)

func (self *Controller) deliveryCacheMaxAge() time.Duration {
	seconds := 15 * 60
	if self != nil && self.C != nil && self.C.DeliveryCacheMaxAgeSeconds > 0 {
		seconds = self.C.DeliveryCacheMaxAgeSeconds
	}
	return time.Duration(seconds) * time.Second
}

func (self *Controller) deliveryReservationTTL() time.Duration {
	seconds := int((defaultTrackingSignatureTTL + maxTrackingSignatureFutureSkew).Seconds())
	if self != nil && self.C != nil && self.C.DeliveryReservationSeconds > 0 {
		seconds = self.C.DeliveryReservationSeconds
	}
	return time.Duration(seconds) * time.Second
}

func (self *Controller) deliveryStateTTL() time.Duration {
	seconds := 2 * 24 * 60 * 60
	if self != nil && self.C != nil && self.C.DeliveryStateTTLSeconds > 0 {
		seconds = self.C.DeliveryStateTTLSeconds
	}
	return time.Duration(seconds) * time.Second
}

func (self *Controller) reserveDelivery(ctx context.Context, block match.RAdv, when time.Time, cost float32) (string, error) {
	if !block.Delivery.Limited() {
		return "", nil
	}
	if math.IsNaN(float64(cost)) || math.IsInf(float64(cost), 0) || cost < 0 {
		metricDeliveryReservationErrors.Add(1)
		return "", fmt.Errorf("invalid delivery reservation cost %v", cost)
	}
	if self == nil || self.Redis == nil {
		metricDeliveryReservationErrors.Add(1)
		return "", fmt.Errorf("redis mutable delivery state unavailable")
	}
	dailyFraction, err := block.Delivery.DailyPacingFraction(when)
	if err != nil {
		metricDeliveryReservationErrors.Add(1)
		return "", err
	}
	totalFraction := block.Delivery.TotalPacingFraction(when)
	day := when.UTC().Format("2006-01-02")
	dailyStateTTL := ttlSeconds(when.UTC().Truncate(24*time.Hour).Add(24*time.Hour).Sub(when.UTC()) + self.deliveryStateTTL())
	balances := []deliveryReservationBalance{
		{key: deliveryTotalKey(block.Delivery.CampaignTotal.ID), balance: block.Delivery.CampaignTotal, pacing: block.Delivery.Campaign.Pacing, fraction: totalFraction},
		{key: deliveryDailyKey(day, block.Delivery.CampaignDaily.ID), balance: block.Delivery.CampaignDaily, pacing: block.Delivery.Campaign.Pacing, fraction: dailyFraction, ttl: dailyStateTTL},
		{key: deliveryTotalKey(block.Delivery.ItemTotal.ID), balance: block.Delivery.ItemTotal, pacing: block.Delivery.Item.Pacing, fraction: totalFraction},
		{key: deliveryDailyKey(day, block.Delivery.ItemDaily.ID), balance: block.Delivery.ItemDaily, pacing: block.Delivery.Item.Pacing, fraction: dailyFraction, ttl: dailyStateTTL},
	}
	unique := make([]deliveryReservationBalance, 0, len(balances))
	byKey := make(map[string]int, len(balances))
	for _, balance := range balances {
		if !balance.balance.Limited() {
			continue
		}
		if index, ok := byKey[balance.key]; ok {
			if balance.pacing == match.DeliveryPacingEven {
				unique[index].pacing = balance.pacing
				if balance.fraction < unique[index].fraction {
					unique[index].fraction = balance.fraction
				}
			}
			continue
		}
		byKey[balance.key] = len(unique)
		unique = append(unique, balance)
	}
	if len(unique) == 0 {
		return "", nil
	}
	token, err := newTrackingEventClaimToken()
	if err != nil {
		metricDeliveryReservationErrors.Add(1)
		return "", err
	}
	keys := make([]string, 1, len(unique)+1)
	keys[0] = deliveryReservationKey(token)
	args := []string{
		strconv.FormatFloat(float64(cost), 'f', -1, 64),
		strconv.Itoa(ttlSeconds(self.deliveryReservationTTL())),
	}
	for _, state := range unique {
		keys = append(keys, state.key)
		even := 0
		if state.pacing == match.DeliveryPacingEven {
			even = 1
		}
		args = append(args,
			strconv.FormatFloat(state.balance.LimitSpend, 'f', -1, 64),
			strconv.FormatUint(state.balance.LimitImp, 10),
			strconv.FormatUint(state.balance.LimitClick, 10),
			strconv.FormatFloat(state.balance.CurrentSpend, 'f', -1, 64),
			strconv.FormatUint(state.balance.CurrentImp, 10),
			strconv.FormatUint(state.balance.CurrentClick, 10),
			strconv.Itoa(even),
			strconv.FormatFloat(state.fraction, 'f', -1, 64),
			strconv.Itoa(state.ttl),
		)
	}
	metricDeliveryReservationAttempts.Add(1)
	redisCtx, cancel := deliveryOperationContext(ctx)
	defer cancel()
	var result int
	if err := self.Redis.Do(redisCtx, reserveDeliveryScript.Cmd(&result, keys, args...)); err != nil {
		metricDeliveryReservationErrors.Add(1)
		return "", err
	}
	if result != 1 {
		metricDeliveryReservationRejected.Add(1)
		return "", errDeliveryLimit
	}
	metricDeliveryReservations.Add(1)
	return token, nil
}

func (self *Controller) releaseDeliveryReservation(ctx context.Context, token string) error {
	if token == "" || self == nil || self.Redis == nil {
		return nil
	}
	redisCtx, cancel := deliveryOperationContext(ctx)
	defer cancel()
	var released int
	err := self.Redis.Do(redisCtx, releaseDeliveryScript.Cmd(&released, []string{deliveryReservationKey(token)}))
	if err != nil {
		metricDeliveryReleaseErrors.Add(1)
		return err
	}
	if released == 1 {
		metricDeliveryReleases.Add(1)
	}
	return nil
}

func (self *Controller) finalizeDeliveryReservation(ctx context.Context, token string, validUntil time.Time) error {
	if token == "" || self == nil || self.Redis == nil {
		return nil
	}
	redisCtx, cancel := deliveryOperationContext(ctx)
	defer cancel()
	var finalized int
	err := self.Redis.Do(redisCtx, finalizeDeliveryScript.Cmd(&finalized, []string{deliveryReservationKey(token)}, strconv.FormatInt(validUntil.Unix(), 10)))
	if err != nil {
		metricDeliveryFinalizeErrors.Add(1)
		return err
	}
	if finalized == 1 {
		metricDeliveryFinalized.Add(1)
	}
	return nil
}

func (self *Controller) recordDeliveryClick(ctx context.Context, token string) error {
	if token == "" || self == nil || self.Redis == nil {
		return nil
	}
	redisCtx, cancel := deliveryOperationContext(ctx)
	defer cancel()
	var applied int
	err := self.Redis.Do(redisCtx, clickDeliveryScript.Cmd(&applied, []string{deliveryReservationKey(token)}))
	if err != nil {
		metricDeliveryClickErrors.Add(1)
		return err
	}
	if applied == 1 {
		metricDeliveryClicks.Add(1)
	}
	return nil
}

func deliveryOperationContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(parent), deliveryOperationTimeout)
}

func deliveryReservationKey(token string) string {
	return "delivery:reservation:" + token
}

func deliveryTotalKey(id uint32) string {
	return "delivery:budget:total:" + strconv.FormatUint(uint64(id), 10)
}

func deliveryDailyKey(day string, id uint32) string {
	return "delivery:budget:daily:" + day + ":" + strconv.FormatUint(uint64(id), 10)
}
