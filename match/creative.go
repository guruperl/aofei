package match

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"html"
	"io"
	"math"
	"mime"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
	"github.com/prebid/openrtb/v20/openrtb2"
)

const (
	CreativeMediaBanner = "Banner"
	CreativeMediaVideo  = "Video"
	CreativeMediaNative = "Native"
)

type Creative struct {
	// creative name
	CreativeName string
	// creative content
	CreativeContent string
	SizeID          uint32
	// MediaType and MIME are additive gob fields. New readers require them
	// before bidding; old readers can decode and ignore them during a
	// cache-first rolling rollout.
	MediaType string
	MIME      string
	// click landing, here for retrieve from Redis only
	Landing string
	// Failback is retained for creative-cache compatibility. The database
	// compiler leaves it empty because adv_campaign.foreign_id is an external
	// business identifier, not a URL.
	Failback string
	// campaign quality check
	IURL string
	// imp tracker
	ImpTrackers []string
	// click tracker
	ClickTrackers []string
}

type compiledCreative struct {
	id   uint32
	data []byte
}

// Pack serializes the audience into a byte slice.
func (self *Creative) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := writeCachePayloadHeader(buf, cachePayloadKindCreative, cachePayloadVersionCreative); err != nil {
		return nil, err
	}
	err := gob.NewEncoder(buf).Encode(self)
	return buf.Bytes(), err
}

// PackIO serializes the audience into a byte slice for IO.
func (self *Creative) PackIO(w io.Writer) error {
	if err := writeCachePayloadHeader(w, cachePayloadKindCreative, cachePayloadVersionCreative); err != nil {
		return err
	}
	return gob.NewEncoder(w).Encode(self)
}

// UnpackCreativeIO deserializes the audience from an IO reader.
func UnpackCreativeIO(r io.Reader) (*Creative, error) {
	data, err := readAllCachePayload(r, cachePayloadKindCreative, cachePayloadVersionCreative)
	if err != nil {
		return nil, err
	}
	audience := new(Creative)
	err = gob.NewDecoder(bytes.NewReader(data)).Decode(audience)
	return audience, err
}

// UnpackCreative deserializes the audience from a byte slice.
func UnpackCreative(data []byte) (*Creative, error) {
	var err error
	data, err = unpackCachePayload(data, cachePayloadKindCreative, cachePayloadVersionCreative)
	if err != nil {
		return nil, err
	}
	audience := new(Creative)
	buf := bytes.NewReader(data)
	err = gob.NewDecoder(buf).Decode(audience)
	return audience, err
}

// DBGetCreativesToRedisSpread retrieves all creatives from the database and inserts them into Redis.
func DBGetCreativesToRedisSpread(ctx context.Context, conn interface{}, db *sql.DB, extra ...string) error {
	sink, err := CacheSinkFor(conn)
	if err != nil {
		return err
	}
	compiled, err := compileCreatives(ctx, db, true, extra...)
	if err != nil {
		return err
	}
	for _, creative := range compiled {
		if err := sink.PutCreative(ctx, creative.id, creative.data); err != nil {
			return err
		}
	}
	return nil
}

// DBValidateItemCreativesForActivation validates the active creative rows of
// an item before that item becomes active. It intentionally does not require
// the item itself to be active, unlike normal cache compilation.
func DBValidateItemCreativesForActivation(ctx context.Context, db *sql.DB, itemID string) error {
	compiled, err := compileCreatives(ctx, db, false, "item_id", itemID)
	if err != nil {
		return err
	}
	if len(compiled) == 0 {
		return fmt.Errorf("item %s has no active creatives", itemID)
	}
	return nil
}

func compileCreatives(ctx context.Context, db *sql.DB, requireActiveParents bool, extra ...string) ([]compiledCreative, error) {
	filterSQL, filterValue, err := creativeCacheFilter(extra...)
	if err != nil {
		return nil, err
	}
	var pars []interface{}
	str := `
SELECT r.creative_id, r.size_id, r.weight, c.iurl, i.item_click, i.imp_url, i.click_url,
  r.creative_name, r.content, r.media_type,
  COALESCE((SELECT m.mime FROM adv_media m WHERE m.creative_id=r.creative_id ORDER BY m.series, m.media_id LIMIT 1), '')
FROM adv_creative r
INNER JOIN adv_item i USING (item_id)
INNER JOIN adv_campaign c USING (campaign_id)
INNER JOIN adv a USING (adv_id)`
	if requireActiveParents {
		str += `WHERE a.active="Yes" AND c.active="Yes" AND i.active="Yes" AND r.active="Yes"`
	} else {
		str += `WHERE r.active="Yes"`
	}
	if filterSQL != "" {
		str += filterSQL
		pars = append(pars, filterValue)
	}
	rows, err := db.QueryContext(ctx, str, pars...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var compiled []compiledCreative
	for rows.Next() {
		var creativeID uint32
		var weight sql.NullFloat64
		var iurl, landing, content, impTracker, clickTracker sql.NullString
		cre := new(Creative)
		err = rows.Scan(&creativeID, &cre.SizeID, &weight, &iurl, &landing, &impTracker, &clickTracker, &cre.CreativeName, &content, &cre.MediaType, &cre.MIME)
		if err != nil {
			return nil, err
		}
		if !weight.Valid || weight.Float64 <= 0 || math.IsNaN(weight.Float64) || math.IsInf(weight.Float64, 0) {
			return nil, fmt.Errorf("creative %d has invalid rotation weight %v", creativeID, weight)
		}
		if iurl.Valid {
			cre.IURL = iurl.String
		}
		if landing.Valid {
			cre.Landing = landing.String
		}
		if impTracker.Valid {
			for _, v := range strings.Split(impTracker.String, ",") {
				if item := strings.TrimSpace(v); item != "" {
					cre.ImpTrackers = append(cre.ImpTrackers, item)
				}
			}
		}
		if clickTracker.Valid {
			for _, v := range strings.Split(clickTracker.String, ",") {
				if item := strings.TrimSpace(v); item != "" {
					cre.ClickTrackers = append(cre.ClickTrackers, item)
				}
			}
		}
		if content.Valid {
			cre.CreativeContent = content.String
		}
		if err := cre.ValidateConfiguration(false); err != nil {
			return nil, fmt.Errorf("creative %d: %w", creativeID, err)
		}
		data, err := cre.Pack()
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, compiledCreative{id: creativeID, data: data})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return compiled, nil
}

func creativeCacheFilter(extra ...string) (string, string, error) {
	if len(extra) == 0 {
		return "", "", nil
	}
	if len(extra) != 2 {
		return "", "", fmt.Errorf("creative cache filter requires key and value, got %d arguments", len(extra))
	}
	if strings.TrimSpace(extra[1]) == "" {
		return "", "", fmt.Errorf("creative cache filter %q has empty value", extra[0])
	}
	switch extra[0] {
	case "item_id":
		return " AND i.item_id=?", extra[1], nil
	case "campaign_id":
		return " AND c.campaign_id=?", extra[1], nil
	case "creative_id":
		return " AND r.creative_id=?", extra[1], nil
	default:
		return "", "", fmt.Errorf("unsupported creative cache filter %q", extra[0])
	}
}

// DBGetCreative retrieves category audience from the database.
func DBGetCreative(db *sql.DB, creativeID uint32) (*Creative, error) {
	cre := new(Creative)
	var iurl, landing, content, mediaType, mimeValue sql.NullString
	err := db.QueryRow(`
SELECT c.iurl, i.item_click, r.size_id, r.creative_name, r.content, r.media_type,
  COALESCE((SELECT m.mime FROM adv_media m WHERE m.creative_id=r.creative_id ORDER BY m.series, m.media_id LIMIT 1), '')
FROM adv_creative r
INNER JOIN adv_item i USING (item_id)
INNER JOIN adv_campaign c USING (campaign_id)
WHERE r.creative_id=?`, creativeID).Scan(&iurl, &landing, &cre.SizeID, &cre.CreativeName, &content, &mediaType, &mimeValue)
	if err != nil {
		return cre, err
	}
	if iurl.Valid {
		cre.IURL = iurl.String
	}
	if landing.Valid {
		cre.Landing = landing.String
	}
	if content.Valid {
		cre.CreativeContent = content.String
	}
	if mediaType.Valid {
		cre.MediaType = mediaType.String
	}
	if mimeValue.Valid {
		cre.MIME = mimeValue.String
	}

	return cre, nil
}

const (
	HashNameCreative = "creative"
)

// ToRedis inserts creative into Redis.
func (self *Creative) ToRedis(ctx context.Context, conn radix.Client, creativeID uint32) error {
	data, err := self.Pack()
	if err != nil {
		return err
	}
	return RedisCacheSink{Client: conn}.PutCreative(ctx, creativeID, data)
}

// ToSpread publishes creative to nats.
func (self *Creative) ToSpread(conn *nats.Conn, creativeID uint32) error {
	data, err := self.Pack()
	if err != nil {
		return err
	}
	return SpreadCacheSink{Conn: conn}.PutCreative(context.Background(), creativeID, data)
}

// CreativeFromRedis retrieves audience data from Redis.
func CreativeFromRedis(ctx context.Context, conn radix.Client, creativeID uint32) (*Creative, error) {
	var bs []byte
	err := conn.Do(ctx, radix.Cmd(&bs, "HGET", HashNameCreative, fmt.Sprintf("%d", creativeID)))
	if err != nil {
		return nil, err
	}
	return UnpackCreative(bs)
}

// CreativeFromIO retrieves audience data from IO.
func CreativeFromIO(top string, creativeID uint32) (*Creative, error) {
	fh, err := os.Open(fmt.Sprintf("%s/%s/%d", top, HashNameCreative, creativeID))
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	return UnpackCreativeIO(fh)
}

// CreativeMapFromRedis retrieves all creatives from Redis.
func CreativeMapFromRedis(ctx context.Context, conn radix.Client) (map[uint32]*Creative, error) {
	var arr []string
	err := conn.Do(ctx, radix.Cmd(&arr, "HGETALL", HashNameCreative))
	if err != nil {
		return nil, err
	}
	if len(arr) == 0 {
		return nil, nil // No creatives found in Redis
	}
	if len(arr)%2 != 0 {
		return nil, sql.ErrNoRows // Invalid format
	}
	creatives := make(map[uint32]*Creative)
	for i := 0; i < len(arr); i += 2 {
		creativeID := arr[i]
		data := []byte(arr[i+1])
		cre, err := UnpackCreative(data)
		if err != nil {
			return nil, err
		}
		id, err := strconv.ParseUint(creativeID, 10, 32)
		if err != nil {
			return nil, err
		}
		creatives[uint32(id)] = cre
	}
	return creatives, nil
}

// CreativeMapFromIO retrieves all creatives from IO.
func CreativeMapFromIO(top string) (map[uint32]*Creative, error) {
	var creatives = make(map[uint32]*Creative)
	files, err := os.ReadDir(fmt.Sprintf("%s/%s", top, HashNameCreative))
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		creativeID, err := strconv.ParseUint(file.Name(), 10, 32)
		if err != nil {
			return nil, err
		}
		creative, err := CreativeFromIO(top, uint32(creativeID))
		if err != nil {
			return nil, err
		}
		creatives[uint32(creativeID)] = creative
	}
	return creatives, nil
}

// LandingURL returns the advertiser destination after supported macro
// expansion. It is used by the DSP click redirect wrapper.
func (self *Creative) LandingURL(macroStandard, macroCustom map[string]string) (string, error) {
	landing, err := applyMacro(self.Landing, macroStandard, macroCustom)
	if err != nil {
		return "", err
	}
	if err := validateCreativeHTTPURL("landing", landing, false); err != nil {
		return "", err
	}
	return landing, nil
}

func (self *Creative) AdM(attr *Attribute, impTracker, clickTracker string, macroStandard, macroCustom map[string]string) (string, error) {
	if attr == nil {
		return "", fmt.Errorf("creative attribute is required")
	}
	impTrackers := []string{impTracker}
	var clickTrackers []string
	for _, v := range self.ImpTrackers {
		item, err := applyMacro(v, macroStandard, macroCustom)
		if err != nil {
			return "", err
		}
		impTrackers = append(impTrackers, item)
	}
	for _, v := range self.ClickTrackers {
		item, err := applyMacro(v, macroStandard, macroCustom)
		if err != nil {
			return "", err
		}
		clickTrackers = append(clickTrackers, item)
	}

	landing, err := self.LandingURL(macroStandard, macroCustom)
	if err != nil {
		return "", err
	}
	failback, err := applyMacro(self.Failback, macroStandard, macroCustom)
	if err != nil {
		return "", err
	}
	content := applyBannerMacros(self.CreativeContent, clickTracker, landing)

	w, h := SizeID1To2(self.SizeID)
	switch self.MediaType {
	case CreativeMediaNative:
		nativeCreative, err := ParseNativeCreativeV1(content)
		if err != nil {
			return "", err
		}
		native, err := NativeFromCreativeV1(attr.NativeFormat, nativeCreative, self.CreativeName, w, h)
		if err != nil {
			return "", err
		}
		return native.AdM(clickTracker, failback, impTrackers, clickTrackers)
	case CreativeMediaVideo:
		return videoAdM(content, self.effectiveMIME(), w, h, clickTracker, impTrackers, clickTrackers), nil
	case CreativeMediaBanner:
		return bannerAdM(content, w, h, impTrackers), nil
	default:
		return "", fmt.Errorf("creative media type %q is unsupported", self.MediaType)
	}
}

// ValidateConfiguration checks the cache-owned creative contract without
// executing or fetching creative content.
func (self *Creative) ValidateConfiguration(secure bool) error {
	if self == nil {
		return fmt.Errorf("creative is missing")
	}
	if strings.TrimSpace(self.CreativeName) == "" {
		return fmt.Errorf("creative name is required")
	}
	w, h := SizeID1To2(self.SizeID)
	if w == 0 || h == 0 {
		return fmt.Errorf("creative dimensions are invalid")
	}
	if err := validateCreativeHTTPURL("landing", self.Landing, secure); err != nil {
		return err
	}
	for _, target := range []struct {
		name string
		url  string
	}{
		{name: "campaign iurl", url: self.IURL},
		{name: "fallback", url: self.Failback},
	} {
		if target.url != "" {
			if err := validateCreativeHTTPURL(target.name, target.url, secure); err != nil {
				return err
			}
		}
	}
	for _, trackers := range []struct {
		name string
		urls []string
	}{
		{name: "impression tracker", urls: self.ImpTrackers},
		{name: "click tracker", urls: self.ClickTrackers},
	} {
		for _, raw := range trackers.urls {
			if err := validateCreativeHTTPURL(trackers.name, raw, secure); err != nil {
				return err
			}
		}
	}

	switch self.MediaType {
	case CreativeMediaBanner:
		if err := validateCreativeHTTPURL("banner content", self.CreativeContent, secure); err != nil {
			return err
		}
		m := self.effectiveMIME()
		if m == "" || (m != "text/html" && !strings.HasPrefix(m, "image/")) {
			return fmt.Errorf("banner creative MIME %q is unsupported", m)
		}
	case CreativeMediaVideo:
		if err := validateCreativeHTTPURL("video content", self.CreativeContent, secure); err != nil {
			return err
		}
		m := self.effectiveMIME()
		if !strings.HasPrefix(m, "video/") && m != "application/x-mpegurl" && m != "application/vnd.apple.mpegurl" {
			return fmt.Errorf("video creative MIME %q is unsupported", m)
		}
	case CreativeMediaNative:
		nativeCreative, err := ParseNativeCreativeV1(self.CreativeContent)
		if err != nil {
			return err
		}
		for _, target := range []struct {
			name string
			url  string
		}{
			{name: "native main image", url: nativeCreative.MainImageURL},
			{name: "native icon", url: nativeCreative.IconURL},
		} {
			if target.url == "" {
				continue
			}
			if err := validateCreativeHTTPURL(target.name, target.url, secure); err != nil {
				return err
			}
			if m := inferCreativeMIME(target.url); !strings.HasPrefix(m, "image/") {
				return fmt.Errorf("%s MIME %q is unsupported", target.name, m)
			}
		}
	default:
		return fmt.Errorf("creative media type %q is unsupported", self.MediaType)
	}
	return nil
}

// ValidateForImp enforces format, dimensions, MIME, and secure-inventory
// compatibility before delivery reservation or markup materialization.
func (self *Creative) ValidateForImp(imp *openrtb2.Imp, attr *Attribute) error {
	if imp == nil || attr == nil {
		return fmt.Errorf("impression and attribute are required")
	}
	secure := imp.Secure != nil && *imp.Secure == 1
	if err := self.ValidateConfiguration(secure); err != nil {
		return err
	}
	if self.SizeID != attr.SizeID {
		return fmt.Errorf("creative size %d does not match impression size %d", self.SizeID, attr.SizeID)
	}
	expected := CreativeMediaBanner
	if attr.NativeFormat != nil {
		expected = CreativeMediaNative
	} else if attr.IsVideo {
		expected = CreativeMediaVideo
	}
	if self.MediaType != expected {
		return fmt.Errorf("creative media type %q does not match %s impression", self.MediaType, strings.ToLower(expected))
	}
	if expected == CreativeMediaVideo && imp.Video == nil {
		return fmt.Errorf("video impression metadata is missing")
	}
	if expected == CreativeMediaBanner && imp.Banner == nil {
		return fmt.Errorf("banner impression metadata is missing")
	}
	if expected == CreativeMediaVideo && len(imp.Video.MIMEs) != 0 && !mimeAllowed(self.effectiveMIME(), imp.Video.MIMEs) {
		return fmt.Errorf("video creative MIME %q is not accepted by the impression", self.effectiveMIME())
	}
	if expected == CreativeMediaBanner && len(imp.Banner.MIMEs) != 0 && !mimeAllowed(self.effectiveMIME(), imp.Banner.MIMEs) {
		return fmt.Errorf("banner creative MIME %q is not accepted by the impression", self.effectiveMIME())
	}
	if expected == CreativeMediaNative {
		nativeCreative, _ := ParseNativeCreativeV1(self.CreativeContent)
		w, h := SizeID1To2(self.SizeID)
		if _, err := NativeFromCreativeV1(attr.NativeFormat, nativeCreative, self.CreativeName, w, h); err != nil {
			return err
		}
	}
	return nil
}

func (self *Creative) effectiveMIME() string {
	value := strings.ToLower(strings.TrimSpace(strings.Split(self.MIME, ";")[0]))
	if value != "" {
		return value
	}
	return inferCreativeMIME(self.CreativeContent)
}

func inferCreativeMIME(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	ext := strings.ToLower(path.Ext(u.Path))
	switch ext {
	case ".htm", ".html":
		return "text/html"
	case ".m3u", ".m3u8":
		return "application/vnd.apple.mpegurl"
	}
	return strings.ToLower(strings.TrimSpace(strings.Split(mime.TypeByExtension(ext), ";")[0]))
}

func mimeAllowed(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if strings.EqualFold(strings.TrimSpace(candidate), value) {
			return true
		}
	}
	return false
}

func validateCreativeHTTPURL(name, raw string, secure bool) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("creative %s URL is required", name)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("creative %s URL: %w", name, err)
	}
	scheme := strings.ToLower(u.Scheme)
	if (scheme != "http" && scheme != "https") || u.Hostname() == "" || u.User != nil {
		return fmt.Errorf("creative %s URL must be an absolute HTTP(S) URL without credentials", name)
	}
	if secure && scheme != "https" {
		return fmt.Errorf("creative %s URL must use HTTPS for secure inventory", name)
	}
	return nil
}

func videoAdM(mediaURL, mediaType string, w, h uint16, clickURL string, impTrackers, clickTrackers []string) string {
	var impressions strings.Builder
	for _, tracker := range impTrackers {
		if tracker != "" {
			fmt.Fprintf(&impressions, `<Impression>%s</Impression>`, html.EscapeString(tracker))
		}
	}
	var clicks strings.Builder
	if clickURL != "" {
		fmt.Fprintf(&clicks, `<ClickThrough>%s</ClickThrough>`, html.EscapeString(clickURL))
	}
	for _, tracker := range clickTrackers {
		if tracker != "" {
			fmt.Fprintf(&clicks, `<ClickTracking>%s</ClickTracking>`, html.EscapeString(tracker))
		}
	}
	return fmt.Sprintf(`<VAST version="3.0"><Ad><InLine><AdSystem>W8M</AdSystem><AdTitle>W8M Video</AdTitle>%s<Creatives><Creative><Linear><MediaFiles><MediaFile type="%s" width="%d" height="%d">%s</MediaFile></MediaFiles><VideoClicks>%s</VideoClicks></Linear></Creative></Creatives></InLine></Ad></VAST>`, impressions.String(), html.EscapeString(mediaType), w, h, html.EscapeString(mediaURL), clicks.String())
}

func bannerAdM(src string, w, h uint16, impTrackers []string) string {
	adm := fmt.Sprintf(`<iframe src="%s" width="%d" height="%d" frameborder="0" scrolling="no" marginheight="0" marginwidth="0" topmargin="0" leftmargin="0"></iframe>`, html.EscapeString(src), w, h)
	for _, tracker := range impTrackers {
		if tracker == "" {
			continue
		}
		adm += fmt.Sprintf(`<img src="%s" width="1" height="1" style="display:none" alt="">`, html.EscapeString(tracker))
	}
	return adm
}

func applyBannerMacros(content, clickURL, landingURL string) string {
	// clickURL and landingURL are internally generated and escaped when banner
	// markup is rendered; do not pass raw user input through this helper.
	content = strings.ReplaceAll(content, "{CLICK_URL}", clickURL)
	content = strings.ReplaceAll(content, "${CLICK_URL}", clickURL)
	content = strings.ReplaceAll(content, "{LANDING_URL}", landingURL)
	content = strings.ReplaceAll(content, "${LANDING_URL}", landingURL)
	return content
}

// applyMacro applies the macro to the URL.
func applyMacro(str string, macroStandard, macroCustom map[string]string) (string, error) {
	if str == "" {
		return "", nil
	}
	u, err := url.Parse(str)
	if err != nil {
		return "", err
	}
	args := url.Values{}
	for k, values := range u.Query() {
		replaced := make([]string, len(values))
		for i, v := range values {
			replaced[i] = replaceMacroValue(v, macroStandard, macroCustom)
		}
		args[k] = replaced
	}
	u.RawQuery = args.Encode()
	return u.String(), nil
}

func replaceMacroValue(value string, macroStandard, macroCustom map[string]string) string {
	if replacement, ok := macroStandard[value]; ok {
		return replacement
	}
	if replacement, ok := macroCustom[value]; ok {
		return replacement
	}
	for macro, replacement := range macroStandard {
		value = strings.ReplaceAll(value, macro, replacement)
	}
	for macro, replacement := range macroCustom {
		value = strings.ReplaceAll(value, macro, replacement)
	}
	return value
}
