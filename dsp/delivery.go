package dsp

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"

	"github.com/guruperl/aofei/accounting"
	"github.com/guruperl/aofei/match"
	"github.com/mediocregopher/radix/v4"
)

var (
	errDeliveryLimit    = errors.New("delivery limit reached")
	errDeliveryOverflow = errors.New("delivery spend overflow")
)

const deliveryOperationTimeout = 2 * time.Second

// auctionCPMToSpend converts an OpenRTB USD CPM price to the USD amount for
// one billable impression. Bid and tracking payloads remain expressed in CPM.
func auctionCPMToSpend(cpm accounting.CPM) (accounting.Nano, error) {
	return cpm.ImpressionNano()
}

type deliveryReservationBalance struct {
	key          string
	balance      match.DeliveryBalance
	pacing       uint8
	fraction     float64
	allowedSpend accounting.Nano
	allowedImp   uint64
	ttl          int
}

var reserveDeliveryScript = radix.NewEvalScript(`
local function norm(value)
  value = tostring(value or "0")
  value = string.gsub(value, "^0+", "")
  if value == "" then return "0" end
  return value
end
local function cmp(a, b)
  a = norm(a); b = norm(b)
  if string.len(a) < string.len(b) then return -1 end
  if string.len(a) > string.len(b) then return 1 end
  if a < b then return -1 end
  if a > b then return 1 end
  return 0
end
local function add(a, b)
  a = norm(a); b = norm(b)
  local carry = 0
  local out = ""
  local ai = string.len(a)
  local bi = string.len(b)
  while ai > 0 or bi > 0 or carry > 0 do
    local av = 0; local bv = 0
    if ai > 0 then av = string.byte(a, ai) - 48; ai = ai - 1 end
    if bi > 0 then bv = string.byte(b, bi) - 48; bi = bi - 1 end
    local sum = av + bv + carry
    out = string.char(48 + (sum % 10)) .. out
    carry = math.floor(sum / 10)
  end
  return norm(out)
end
if redis.call("EXISTS", KEYS[1]) ~= 0 then return -1 end
local cost = norm(ARGV[1])
local imp = 1
local reservation_ttl = tonumber(ARGV[2])
local offset = 3
for i=2,#KEYS do
  local base_spend = norm(ARGV[offset+3])
  local base_imp = tonumber(ARGV[offset+4])
  local base_click = tonumber(ARGV[offset+5])
  local floor_spend = norm(redis.call("HGET", KEYS[i], "floor_spend_nano") or "0")
  local floor_imp = tonumber(redis.call("HGET", KEYS[i], "floor_imp") or "0")
  local floor_click = tonumber(redis.call("HGET", KEYS[i], "floor_click") or "0")
  if cmp(base_spend, floor_spend) > 0 then floor_spend = base_spend end
  if base_imp > floor_imp then floor_imp = base_imp end
  if base_click > floor_click then floor_click = base_click end
  local used_spend = norm(redis.call("HGET", KEYS[i], "used_spend_nano") or "0")
  local used_imp = tonumber(redis.call("HGET", KEYS[i], "used_imp") or "0")
  local used_click = tonumber(redis.call("HGET", KEYS[i], "used_click") or "0")
  if cmp(floor_spend, used_spend) > 0 then used_spend = floor_spend end
  if floor_imp > used_imp then used_imp = floor_imp end
  if floor_click > used_click then used_click = floor_click end
  redis.call("HSET", KEYS[i],
    "used_spend_nano", used_spend, "used_imp", used_imp, "used_click", used_click,
    "floor_spend_nano", floor_spend, "floor_imp", floor_imp, "floor_click", floor_click)
  local state_ttl = tonumber(ARGV[offset+9])
  local ttl = redis.call("TTL", KEYS[i])
  if state_ttl == 0 then
    if ttl >= 0 then redis.call("PERSIST", KEYS[i]) end
  elseif ttl < state_ttl then
    redis.call("EXPIRE", KEYS[i], state_ttl)
  end
  offset = offset + 10
end
offset = 3
for i=2,#KEYS do
  local limit_spend = norm(ARGV[offset])
  local limit_imp = tonumber(ARGV[offset+1])
  local limit_click = tonumber(ARGV[offset+2])
  local even = tonumber(ARGV[offset+6])
  local allowed_spend = norm(ARGV[offset+7])
  local allowed_imp = tonumber(ARGV[offset+8])
  local used_spend = norm(redis.call("HGET", KEYS[i], "used_spend_nano") or "0")
  local used_imp = tonumber(redis.call("HGET", KEYS[i], "used_imp") or "0")
  local used_click = tonumber(redis.call("HGET", KEYS[i], "used_click") or "0")
  local proposed_spend = add(used_spend, cost)
  if cmp(proposed_spend, "9223372036854775807") > 0 then return -2 end
  if (limit_spend ~= "0" and (cmp(used_spend, limit_spend) >= 0 or cmp(proposed_spend, limit_spend) > 0)) or
     (limit_imp > 0 and (used_imp >= limit_imp or used_imp + imp > limit_imp)) or
     (limit_click > 0 and used_click >= limit_click) then return 0 end
  if even == 1 then
    if allowed_imp < imp then allowed_imp = imp end
    if (limit_spend ~= "0" and cmp(add(used_spend, cost), allowed_spend) > 0) or
       (limit_imp > 0 and used_imp + imp > allowed_imp) then return 0 end
  end
  offset = offset + 10
end
offset = 3
redis.call("HSET", KEYS[1], "status", "active", "cost", ARGV[1], "count", #KEYS-1)
redis.call("EXPIRE", KEYS[1], reservation_ttl)
for i=2,#KEYS do
  redis.call("HINCRBY", KEYS[i], "used_spend_nano", cost)
  redis.call("HINCRBY", KEYS[i], "used_imp", imp)
  local state_ttl = tonumber(ARGV[offset+9])
  redis.call("HSET", KEYS[1], "key:" .. (i-1), KEYS[i], "ttl:" .. (i-1), state_ttl)
  offset = offset + 10
end
return 1`)

var releaseDeliveryScript = radix.NewEvalScript(`
if redis.call("HGET", KEYS[1], "status") ~= "active" then
  return 0
end
local cost = redis.call("HGET", KEYS[1], "cost") or "0"
local count = tonumber(redis.call("HGET", KEYS[1], "count") or "0")
for i=1,count do
  local key = redis.call("HGET", KEYS[1], "key:" .. i)
	if key then
	  redis.call("HINCRBY", key, "used_spend_nano", "-" .. cost)
	  local imp = tonumber(redis.call("HGET", key, "used_imp") or "0") - 1
	  local floor_imp = tonumber(redis.call("HGET", key, "floor_imp") or "0")
	  if imp < floor_imp then imp = floor_imp end
    redis.call("HSET", key, "used_imp", imp)
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

func (self *Controller) reserveDelivery(ctx context.Context, block match.RAdv, when time.Time, cost accounting.Nano) (string, error) {
	if !block.Delivery.Limited() {
		return "", nil
	}
	if cost < 0 {
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
	for index := range unique {
		unique[index].allowedSpend = pacingAllowedSpend(unique[index].balance.LimitSpendNano, cost, unique[index].fraction)
		unique[index].allowedImp = uint64(math.Floor(float64(unique[index].balance.LimitImp) * unique[index].fraction))
	}
	token, err := newTrackingEventClaimToken()
	if err != nil {
		metricDeliveryReservationErrors.Add(1)
		return "", err
	}
	keys := make([]string, 1, len(unique)+1)
	keys[0] = deliveryReservationKey(token)
	args := []string{
		strconv.FormatInt(int64(cost), 10),
		strconv.Itoa(ttlSeconds(self.deliveryReservationTTL())),
	}
	for _, state := range unique {
		keys = append(keys, state.key)
		even := 0
		if state.pacing == match.DeliveryPacingEven {
			even = 1
		}
		args = append(args,
			strconv.FormatInt(int64(state.balance.LimitSpendNano), 10),
			strconv.FormatUint(state.balance.LimitImp, 10),
			strconv.FormatUint(state.balance.LimitClick, 10),
			strconv.FormatInt(int64(state.balance.CurrentSpendNano), 10),
			strconv.FormatUint(state.balance.CurrentImp, 10),
			strconv.FormatUint(state.balance.CurrentClick, 10),
			strconv.Itoa(even),
			strconv.FormatInt(int64(state.allowedSpend), 10),
			strconv.FormatUint(state.allowedImp, 10),
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
	if result == -2 {
		metricDeliveryReservationErrors.Add(1)
		return "", errDeliveryOverflow
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
	return "delivery:v3:reservation:" + token
}

func deliveryTotalKey(id uint32) string {
	return "delivery:v3:budget:total:" + strconv.FormatUint(uint64(id), 10)
}

func deliveryDailyKey(day string, id uint32) string {
	return "delivery:v3:budget:daily:" + day + ":" + strconv.FormatUint(uint64(id), 10)
}

func pacingAllowedSpend(limit, minimum accounting.Nano, fraction float64) accounting.Nano {
	if limit <= 0 {
		return 0
	}
	if fraction <= 0 {
		return minimum
	}
	if fraction >= 1 {
		return limit
	}
	const fractionScale int64 = 1_000_000_000
	scaled := int64(math.Floor(fraction * float64(fractionScale)))
	product := new(big.Int).Mul(big.NewInt(int64(limit)), big.NewInt(scaled))
	product.Quo(product, big.NewInt(fractionScale))
	allowed := accounting.Nano(product.Int64())
	if allowed < minimum {
		return minimum
	}
	return allowed
}
