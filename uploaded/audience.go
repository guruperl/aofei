package uploaded

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mediocregopher/radix/v4"
	"github.com/prebid/openrtb/v20/openrtb2"
)

const defaultAudienceTTL = 30 * 24 * time.Hour

var audienceTTLSeconds atomic.Int64

func init() {
	audienceTTLSeconds.Store(int64(defaultAudienceTTL / time.Second))
}

type UploadAudience struct {
	Uploads uint32
}

// Has returns false if and only if the ad targets the uploading type but not found.
// Like all the other audience targeting, the default is true.
func (self *UploadAudience) Has(ctx context.Context, conn radix.Client, bid *openrtb2.BidRequest, advID uint32) (bool, error) {
	if self.Uploads == 0 {
		return true, nil
	}
	if bid == nil {
		return false, nil
	}

	var did, dpid, mac string
	var buyerUID, userID, ip, ifa string
	if bid.Device != nil {
		did = bid.Device.DIDMD5
		if did == "" {
			did = bid.Device.DIDSHA1
		}
		dpid = bid.Device.DPIDMD5
		if dpid == "" {
			dpid = bid.Device.DPIDSHA1
		}
		mac = bid.Device.MACMD5
		if mac == "" {
			mac = bid.Device.MACSHA1
		}
		ip = bid.Device.IP
		ifa = bid.Device.IFA
	}
	if bid.User != nil {
		buyerUID = bid.User.BuyerUID
		userID = bid.User.ID
	}

	args := map[UploadType]string{
		UploadBuyerUID: buyerUID,
		UploadUserID:   userID,
		UploadIP:       ip,
		UploadIFA:      ifa,
		UploadDID:      did,
		UploadDPID:     dpid,
		UploadMAC:      mac,
	}

	for typ, v := range args {
		if !self.hasUpload(typ) {
			continue
		}
		if v == "" {
			return false, nil
		}
		ok, err := self.findUploaded(ctx, conn, advID, UploadType2String[typ], v)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}

	return true, nil
}

// uploadName is the name of the audience data in Redis.
func uploadName(advID uint32, marker string) string {
	return fmt.Sprintf("upload:%d:%s", advID, marker)
}

// findUploaded checks if the given value is present in the audience data.
var findUploadedScript = radix.NewEvalScript(`
for _, key in ipairs(KEYS) do
  if redis.call("SISMEMBER", key, ARGV[1]) == 1 then
    return 1
  end
end
return 0`)

func (self *UploadAudience) findUploaded(ctx context.Context, conn radix.Client, advID uint32, marker string, target string) (bool, error) {
	if marker == "" || target == "" {
		return false, nil
	}
	canonical, err := NormalizeAudienceMarker(marker)
	if err != nil {
		return false, err
	}
	keys := make([]string, 0, 2)
	for _, candidate := range audienceMarkerReadNames(canonical) {
		keys = append(keys, uploadName(advID, candidate))
	}
	var ok int
	if err := conn.Do(ctx, findUploadedScript.Cmd(&ok, keys, target)); err != nil {
		return false, err
	}
	return ok == 1, nil
}

func UploadSingle(ctx context.Context, conn radix.Client, advID uint32, marker string, single string) error {
	return UploadSingleWithTTL(ctx, conn, advID, marker, single, DefaultAudienceTTL())
}

func UploadMany(ctx context.Context, conn radix.Client, advID uint32, marker string, data []string) error {
	return UploadManyWithTTL(ctx, conn, advID, marker, data, DefaultAudienceTTL())
}

// SetDefaultAudienceTTL configures retention for subsequent audience uploads.
// Existing keys adopt the configured floor when they are written again.
func SetDefaultAudienceTTL(ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("uploaded audience TTL must be positive")
	}
	seconds := int64((ttl + time.Second - 1) / time.Second)
	audienceTTLSeconds.Store(seconds)
	return nil
}

func DefaultAudienceTTL() time.Duration {
	seconds := audienceTTLSeconds.Load()
	if seconds <= 0 {
		return defaultAudienceTTL
	}
	return time.Duration(seconds) * time.Second
}

func UploadSingleWithTTL(ctx context.Context, conn radix.Client, advID uint32, marker string, single string, ttl time.Duration) error {
	if single == "" {
		return fmt.Errorf("uploaded audience identifier is empty")
	}
	return UploadManyWithTTL(ctx, conn, advID, marker, []string{single}, ttl)
}

var uploadManyWithTTLScript = radix.NewEvalScript(`
redis.call("SADD", KEYS[1], unpack(ARGV, 2))
local current_ttl = redis.call("TTL", KEYS[1])
local requested_ttl = tonumber(ARGV[1])
if current_ttl < requested_ttl then
  redis.call("EXPIRE", KEYS[1], requested_ttl)
end
return 1`)

var deleteAudienceIdentifierScript = radix.NewEvalScript(`
local removed = 0
for _, key in ipairs(KEYS) do
  removed = removed + redis.call("SREM", key, ARGV[1])
end
return removed`)

func UploadManyWithTTL(ctx context.Context, conn radix.Client, advID uint32, marker string, data []string, ttl time.Duration) error {
	if len(data) == 0 {
		return nil
	}
	if conn == nil {
		return fmt.Errorf("redis client is nil")
	}
	canonical, err := NormalizeAudienceMarker(marker)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return fmt.Errorf("uploaded audience TTL must be positive")
	}

	arr := make([]string, 1, len(data)+1)
	arr[0] = strconv.FormatInt(int64((ttl+time.Second-1)/time.Second), 10)
	for _, d := range data {
		if d == "" {
			return fmt.Errorf("uploaded audience identifier is empty")
		}
		arr = append(arr, d)
	}
	return conn.Do(ctx, uploadManyWithTTLScript.Cmd(nil, []string{uploadName(advID, canonical)}, arr...))
}

// DeleteAudience removes one advertiser-owned audience source. Operators must
// resolve and authorize the advertiser and marker before calling it.
func DeleteAudience(ctx context.Context, conn radix.Client, advID uint32, marker string) (bool, error) {
	if conn == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	canonical, err := NormalizeAudienceMarker(marker)
	if err != nil {
		return false, err
	}
	var deleted int
	keys := audienceMarkerReadNames(canonical)
	args := make([]string, 0, len(keys))
	for _, candidate := range keys {
		args = append(args, uploadName(advID, candidate))
	}
	if err := conn.Do(ctx, radix.Cmd(&deleted, "DEL", args...)); err != nil {
		return false, err
	}
	return deleted != 0, nil
}

// DeleteAudienceIdentifier removes a supplied identifier from one
// advertiser-owned audience without returning or logging set contents.
func DeleteAudienceIdentifier(ctx context.Context, conn radix.Client, advID uint32, marker, identifier string) (bool, error) {
	if conn == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	canonical, err := NormalizeAudienceMarker(marker)
	if err != nil || identifier == "" {
		return false, fmt.Errorf("uploaded audience marker and identifier are required")
	}
	names := audienceMarkerReadNames(canonical)
	keys := make([]string, 0, len(names))
	for _, candidate := range names {
		keys = append(keys, uploadName(advID, candidate))
	}
	var deleted int
	if err := conn.Do(ctx, deleteAudienceIdentifierScript.Cmd(&deleted, keys, identifier)); err != nil {
		return false, err
	}
	return deleted != 0, nil
}

// NormalizeAudienceMarker returns the bounded Redis marker used by both
// upload writers and serving readers. buyerid is retained only as a legacy UI
// alias for buyeruid; user is retained for historical operator tooling.
func NormalizeAudienceMarker(marker string) (string, error) {
	switch normalized := strings.ToLower(strings.TrimSpace(marker)); normalized {
	case "buyeruid", "buyerid":
		return "buyeruid", nil
	case "userid", "user":
		return "userid", nil
	case "ip", "ifa", "did", "dpid", "mac":
		return normalized, nil
	default:
		return "", fmt.Errorf("uploaded audience marker is unsupported")
	}
}

func audienceMarkerReadNames(canonical string) []string {
	switch canonical {
	case "buyeruid":
		return []string{"buyeruid", "buyerid"}
	case "userid":
		return []string{"userid", "user"}
	default:
		return []string{canonical}
	}
}

// UploadedResetArgs resets the ARGS to the values in the DemoAudience, ready to be inserted or updated in the database.
func UploadedResetArgs(ARGS url.Values) error {
	var par []uint32
	if values, ok := ARGS["uploads"]; ok {
		for _, value := range values {
			if value == "" || value == "0" {
				par = nil
				break
			}
			v, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return err
			}
			par = append(par, uint32(v))
		}
	}

	if par == nil {
		ARGS.Del("uploads")
		return nil
	}

	uploads := NewUploads(par)
	ARGS.Set("uploads", strconv.FormatInt(int64(uploads), 10))

	return nil
}

func (self *UploadAudience) hasUpload(uploadType UploadType) bool {
	if uploadType == UploadUnknown {
		return false
	}
	if self.Uploads&(1<<uploadType) != 0 {
		return true
	}

	return false
}

// Tmpls returns the map of attribute name and valueID ready to use on web page.
func (self *UploadAudience) Tmpls() map[int][]interface{} {
	uploads := make(map[int][]interface{})
	for str, typ := range String2UploadType {
		if self.Uploads == 0 {
			if typ == UploadUnknown {
				uploads[int(typ)] = []interface{}{str, true}
			} else {
				uploads[int(typ)] = []interface{}{str, false}
			}
		} else {
			uploads[int(typ)] = []interface{}{str, self.hasUpload(typ)}
		}
	}
	return uploads
}

// DBFillUploadAudience fills UploadAudience with attribute name and valueID, derived from the database.
func (self *UploadAudience) DBFillUploadAudience(attrname string, valueID uint32) int {
	switch attrname {
	case "uploads":
		self.Uploads = valueID
		return 1
	default:
	}

	return 0
}
