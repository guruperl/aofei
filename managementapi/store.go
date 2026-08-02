package managementapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/guruperl/aofei/match"
)

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type mutation struct {
	Kind       string
	ParentID   uint64
	ResourceID uint64
	Version    uint64
	Value      any
}

type mutationPayload struct {
	Resource  any       `json:"resource"`
	Operation Operation `json:"operation"`
}

func (s *Service) advertiser(ctx context.Context, advID uint64) (map[string]any, error) {
	var id uint64
	var email, status string
	var first, last, domain sql.NullString
	var created sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT adv_id, email, firstname, lastname, domain, active, created FROM adv WHERE adv_id=?`, advID).Scan(&id, &email, &first, &last, &domain, &status, &created)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	out := map[string]any{"id": id, "email": email, "status": status}
	if first.Valid {
		out["first_name"] = first.String
	}
	if last.Valid {
		out["last_name"] = last.String
	}
	if domain.Valid {
		out["domain"] = domain.String
	}
	if created.Valid {
		out["created_at"] = created.Time.UTC()
	}
	return out, nil
}

func (s *Service) campaigns(ctx context.Context, advID, cursor uint64, limit int) ([]Campaign, string, error) {
	rows, err := s.db.QueryContext(ctx, campaignSelect+` WHERE c.adv_id=? AND c.campaign_id>? ORDER BY c.campaign_id LIMIT ?`, advID, cursor, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]Campaign, 0, limit)
	for rows.Next() {
		item, err := scanCampaign(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > limit {
		next = strconv.FormatUint(items[limit-1].ID, 10)
		items = items[:limit]
	}
	return items, next, nil
}

func (s *Service) campaign(ctx context.Context, q queryer, advID, id uint64) (Campaign, error) {
	row := q.QueryRowContext(ctx, campaignSelect+` WHERE c.adv_id=? AND c.campaign_id=?`, advID, id)
	item, err := scanCampaign(row)
	if err == sql.ErrNoRows {
		return Campaign{}, ErrNotFound
	}
	return item, err
}

const campaignSelect = `
SELECT c.campaign_id, c.adv_id, c.campaign_name, c.foreign_id,
       c.startx, c.endx, c.target_type, c.description, c.delivery_timezone,
       c.weekly_schedule, c.pacing_mode, c.active, c.api_version, c.created,
       tb.limit_spend, tb.limit_imp, tb.limit_cli,
       db.limit_spend, db.limit_imp, db.limit_cli
FROM adv_campaign c
LEFT JOIN adv_balance tb ON tb.balance_id=c.total_balance_id
LEFT JOIN adv_balance db ON db.balance_id=c.daily_balance_id`

type rowScanner interface{ Scan(...any) error }

func scanCampaign(row rowScanner) (Campaign, error) {
	var item Campaign
	var externalID, targetType, description, schedule sql.NullString
	var start, end, created sql.NullTime
	var totalSpend, dailySpend sql.NullFloat64
	var totalImps, totalClicks, dailyImps, dailyClicks sql.NullInt64
	err := row.Scan(&item.ID, &item.AdvertiserID, &item.Name, &externalID,
		&start, &end, &targetType, &description, &item.Delivery.Timezone,
		&schedule, &item.Delivery.Pacing, &item.Status, &item.Version, &created,
		&totalSpend, &totalImps, &totalClicks, &dailySpend, &dailyImps, &dailyClicks)
	if err != nil {
		return item, err
	}
	item.ExternalID = externalID.String
	item.TargetType = targetType.String
	item.Description = description.String
	if start.Valid {
		value := start.Time.UTC()
		item.Delivery.StartUTC = &value
	}
	if end.Valid {
		value := end.Time.UTC()
		item.Delivery.EndUTC = &value
	}
	if schedule.Valid {
		value := schedule.String
		item.Delivery.WeeklySchedule = &value
	}
	item.Delivery.TotalLimits = scanLimits(totalSpend, totalImps, totalClicks)
	item.Delivery.DailyLimits = scanLimits(dailySpend, dailyImps, dailyClicks)
	if created.Valid {
		item.CreatedAt = created.Time.UTC()
	}
	return item, nil
}

func (s *Service) items(ctx context.Context, advID, campaignID, cursor uint64, limit int) ([]Item, string, error) {
	if _, err := s.campaign(ctx, s.db, advID, campaignID); err != nil {
		return nil, "", err
	}
	rows, err := s.db.QueryContext(ctx, itemSelect+` WHERE c.adv_id=? AND i.campaign_id=? AND i.item_id>? ORDER BY i.item_id LIMIT ?`, advID, campaignID, cursor, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]Item, 0, limit)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > limit {
		next = strconv.FormatUint(items[limit-1].ID, 10)
		items = items[:limit]
	}
	return items, next, nil
}

func (s *Service) item(ctx context.Context, q queryer, advID, id uint64) (Item, error) {
	item, err := scanItem(q.QueryRowContext(ctx, itemSelect+` WHERE c.adv_id=? AND i.item_id=?`, advID, id))
	if err == sql.ErrNoRows {
		return Item{}, ErrNotFound
	}
	return item, err
}

const itemSelect = `
SELECT i.item_id, i.campaign_id, i.item_name, i.item_click, i.imp_url,
       i.click_url, i.cost, i.startx, i.endx, i.weekly_schedule,
       i.pacing_mode, c.delivery_timezone, i.active, i.api_version, i.created,
       tb.limit_spend, tb.limit_imp, tb.limit_cli,
       db.limit_spend, db.limit_imp, db.limit_cli
FROM adv_item i
INNER JOIN adv_campaign c ON c.campaign_id=i.campaign_id
LEFT JOIN adv_balance tb ON tb.balance_id=i.total_balance_id
LEFT JOIN adv_balance db ON db.balance_id=i.daily_balance_id`

func scanItem(row rowScanner) (Item, error) {
	var item Item
	var impURLs, clickURLs, schedule sql.NullString
	var start, end, created sql.NullTime
	var totalSpend, dailySpend sql.NullFloat64
	var totalImps, totalClicks, dailyImps, dailyClicks sql.NullInt64
	err := row.Scan(&item.ID, &item.CampaignID, &item.Name, &item.LandingURL,
		&impURLs, &clickURLs, &item.PriceCPMUSD, &start, &end, &schedule,
		&item.Delivery.Pacing, &item.Delivery.Timezone, &item.Status, &item.Version, &created,
		&totalSpend, &totalImps, &totalClicks, &dailySpend, &dailyImps, &dailyClicks)
	if err != nil {
		return item, err
	}
	item.ImpressionURLs = splitURLs(impURLs.String)
	item.ClickURLs = splitURLs(clickURLs.String)
	if start.Valid {
		value := start.Time.UTC()
		item.Delivery.StartUTC = &value
	}
	if end.Valid {
		value := end.Time.UTC()
		item.Delivery.EndUTC = &value
	}
	if schedule.Valid {
		value := schedule.String
		item.Delivery.WeeklySchedule = &value
	}
	item.Delivery.TotalLimits = scanLimits(totalSpend, totalImps, totalClicks)
	item.Delivery.DailyLimits = scanLimits(dailySpend, dailyImps, dailyClicks)
	if created.Valid {
		item.CreatedAt = created.Time.UTC()
	}
	return item, nil
}

func (s *Service) creatives(ctx context.Context, advID, itemID, cursor uint64, limit int) ([]Creative, string, error) {
	if _, err := s.item(ctx, s.db, advID, itemID); err != nil {
		return nil, "", err
	}
	rows, err := s.db.QueryContext(ctx, creativeSelect+` WHERE c.adv_id=? AND cr.item_id=? AND cr.creative_id>? ORDER BY cr.creative_id LIMIT ?`, advID, itemID, cursor, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]Creative, 0, limit)
	for rows.Next() {
		item, err := scanCreative(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > limit {
		next = strconv.FormatUint(items[limit-1].ID, 10)
		items = items[:limit]
	}
	return items, next, nil
}

func (s *Service) creative(ctx context.Context, q queryer, advID, id uint64) (Creative, error) {
	item, err := scanCreative(q.QueryRowContext(ctx, creativeSelect+` WHERE c.adv_id=? AND cr.creative_id=?`, advID, id))
	if err == sql.ErrNoRows {
		return Creative{}, ErrNotFound
	}
	return item, err
}

const creativeSelect = `
SELECT cr.creative_id, cr.item_id, cr.creative_name, cr.size_id,
       cr.media_type, cr.content, cr.weight, cr.active, cr.api_version, cr.created
FROM adv_creative cr
INNER JOIN adv_item i ON i.item_id=cr.item_id
INNER JOIN adv_campaign c ON c.campaign_id=i.campaign_id`

func scanCreative(row rowScanner) (Creative, error) {
	var item Creative
	var content sql.NullString
	var created sql.NullTime
	err := row.Scan(&item.ID, &item.ItemID, &item.Name, &item.SizeID, &item.MediaType,
		&content, &item.Weight, &item.Status, &item.Version, &created)
	if err != nil {
		return item, err
	}
	if item.MediaType == match.CreativeMediaNative && content.Valid {
		native, err := match.ParseNativeCreativeV1(content.String)
		if err != nil {
			return item, fmt.Errorf("stored native creative %d: %w", item.ID, err)
		}
		item.Native = &NativeCreative{Version: native.Version, Title: native.Title, Description: native.Description, CTA: native.CTA, IconURL: native.IconURL, MainImageURL: native.MainImageURL}
	} else {
		item.SourceURL = content.String
	}
	item.Width, item.Height = match.SizeID1To2(uint32(item.SizeID))
	if created.Valid {
		item.CreatedAt = created.Time.UTC()
	}
	return item, nil
}

func (s *Service) targeting(ctx context.Context, q queryer, advID, itemID uint64) (Targeting, error) {
	var item Targeting
	var sites, languages, devices, positions string
	err := q.QueryRowContext(ctx, `
SELECT i.item_id, i.api_version, i.fl_sitetypes, i.fl_language,
       i.fl_device, i.fl_position, i.access_order, i.channel_order
FROM adv_item i INNER JOIN adv_campaign c ON c.campaign_id=i.campaign_id
WHERE c.adv_id=? AND i.item_id=?`, advID, itemID).Scan(&item.ItemID, &item.Version,
		&sites, &languages, &devices, &positions, &item.AccessOrder, &item.ChannelOrder)
	if err == sql.ErrNoRows {
		return Targeting{}, ErrNotFound
	}
	if err != nil {
		return Targeting{}, err
	}
	item.SiteTypes = splitSet(sites)
	item.Languages = splitSet(languages)
	item.DeviceTypes = splitSet(devices)
	item.Positions = splitSet(positions)
	return item, nil
}

func (s *Service) operation(ctx context.Context, advID uint64, operationID string) (Operation, error) {
	var op Operation
	var activated sql.NullTime
	err := s.db.QueryRowContext(ctx, `
SELECT operation_id, resource_type, resource_id, accepted_version,
       configuration_state, activation_state, accepted_at,
       activation_deadline, activated_at, publication_mode
FROM api_operation WHERE operation_id=? AND adv_id=?`, operationID, advID).Scan(
		&op.ID, &op.ResourceType, &op.ResourceID, &op.AcceptedVersion,
		&op.ConfigurationState, &op.ActivationState, &op.AcceptedAt,
		&op.ActivationDeadline, &activated, &op.PublicationMode)
	if err == sql.ErrNoRows {
		return Operation{}, ErrNotFound
	}
	if err != nil {
		return Operation{}, err
	}
	if activated.Valid {
		value := activated.Time.UTC()
		op.ActivatedAt = &value
	}
	op.AcceptedAt = op.AcceptedAt.UTC()
	op.ActivationDeadline = op.ActivationDeadline.UTC()
	if op.ActivationState == "Pending" && !s.now().UTC().Before(op.ActivationDeadline) {
		op.ActivationState = "Delayed"
	}
	return op, nil
}

func (s *Service) reports(ctx context.Context, advID, cursor uint64, limit int, from, to time.Time) ([]DeliveryReport, string, bool, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT report_id, timely, campaign_id, item_id, creative_id, demand_source,
       inventory_environment, integration_mode, media_intent, seller_type,
       seller_id, wins, imps, clis, spend_usd, accounting_version
FROM report_delivery
WHERE adv_id=? AND timely>=? AND timely<? AND report_id>?
ORDER BY report_id LIMIT ?`, advID, from.UTC(), to.UTC(), cursor, limit+1)
	if err != nil {
		return nil, "", true, err
	}
	defer rows.Close()
	items := make([]DeliveryReport, 0, limit)
	for rows.Next() {
		var item DeliveryReport
		if err := rows.Scan(&item.ID, &item.IntervalUTC, &item.CampaignID, &item.ItemID,
			&item.CreativeID, &item.DemandSource, &item.InventoryEnvironment,
			&item.IntegrationMode, &item.MediaIntent, &item.SellerType,
			&item.SellerID, &item.Wins, &item.Impressions, &item.Clicks,
			&item.SpendUSD, &item.AccountingVersion); err != nil {
			return nil, "", true, err
		}
		item.IntervalUTC = item.IntervalUTC.UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", true, err
	}
	next := ""
	if len(items) > limit {
		next = strconv.FormatUint(items[limit-1].ID, 10)
		items = items[:limit]
	}
	// Report intervals are derived asynchronously. A recent window is
	// explicitly partial instead of presenting missing intervals as zero.
	partial := to.After(s.now().UTC().Add(-30 * time.Minute))
	return items, next, partial, nil
}

func (s *Service) mutate(ctx context.Context, principal Principal, idempotencyHash, requestHash []byte, m mutation) (MutationResult, error) {
	claimToken := make([]byte, 16)
	if _, err := io.ReadFull(s.random, claimToken); err != nil {
		return MutationResult{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MutationResult{}, err
	}
	defer tx.Rollback()
	claimed, replay, err := claimIdempotency(ctx, tx, principal, idempotencyHash, requestHash, claimToken, s.now().UTC())
	if err != nil {
		return MutationResult{}, err
	}
	if !claimed {
		return replay, nil
	}

	operationID, err := s.newOperationID()
	if err != nil {
		return MutationResult{}, err
	}
	resource, resourceType, resourceID, version, prior, err := s.applyMutation(ctx, tx, principal.AdvertiserID, m)
	if err != nil {
		return MutationResult{}, err
	}
	now := s.now().UTC()
	op := Operation{
		ID: operationID, ResourceType: resourceType, ResourceID: resourceID,
		AcceptedVersion: version, ConfigurationState: "Accepted", ActivationState: "Pending",
		AcceptedAt: now, ActivationDeadline: now.Add(time.Duration(s.config.CacheActivationSeconds) * time.Second),
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO api_operation
 (operation_id, adv_id, credential_id, resource_type, resource_id,
  accepted_version, configuration_state, activation_state, accepted_at,
  activation_deadline)
VALUES (?, ?, ?, ?, ?, ?, 'Accepted', 'Pending', ?, ?)`, op.ID,
		principal.AdvertiserID, principal.CredentialID, resourceType, resourceID,
		version, now, op.ActivationDeadline); err != nil {
		return MutationResult{}, err
	}
	body, err := marshalEnvelope(mutationPayload{Resource: resource, Operation: op})
	if err != nil {
		return MutationResult{}, err
	}
	if len(body) > 64*1024 {
		return MutationResult{}, fmt.Errorf("mutation response exceeds idempotency storage limit")
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE api_idempotency SET state='Complete', response_status=202,
 response_body=?, completed_at=?
WHERE credential_id=? AND idempotency_hash=?`, body, now,
		principal.CredentialID, idempotencyHash); err != nil {
		return MutationResult{}, err
	}
	if err := insertAPIAudit(ctx, tx, apiAudit{Actor: Actor{Role: "service", ID: principal.CredentialID}, CredentialID: principal.CredentialID, AdvID: principal.AdvertiserID, Event: "ResourceMutated", ObjectType: resourceType, ObjectID: resourceID, IdempotencyHash: idempotencyHash, PriorState: prior, NewState: "AcceptedV" + strconv.FormatUint(version, 10), Outcome: "Success", CreatedAt: now}); err != nil {
		return MutationResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Status: 202, Body: body}, nil
}

func claimIdempotency(ctx context.Context, tx *sql.Tx, principal Principal, keyHash, requestHash, claimToken []byte, now time.Time) (bool, MutationResult, error) {
	if len(claimToken) != 16 {
		return false, MutationResult{}, fmt.Errorf("idempotency claim token must be 16 bytes")
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO api_idempotency
 (credential_id, adv_id, idempotency_hash, request_hash, claim_token, state, created_at, expires_at)
VALUES (?, ?, ?, ?, ?, 'InProgress', ?, ?)
ON DUPLICATE KEY UPDATE idempotency_id=LAST_INSERT_ID(idempotency_id)`, principal.CredentialID,
		principal.AdvertiserID, keyHash, requestHash, claimToken, now, now.Add(24*time.Hour))
	if err != nil {
		return false, MutationResult{}, err
	}
	// The upsert takes an exclusive unique-index lock even for a duplicate.
	// This read therefore observes either our new claim or the completed prior
	// transaction without a shared-to-exclusive lock upgrade deadlock.
	var storedHash, storedClaim, body []byte
	var state string
	var status sql.NullInt64
	err = tx.QueryRowContext(ctx, `
SELECT request_hash, claim_token, state, response_status, response_body
FROM api_idempotency WHERE credential_id=? AND idempotency_hash=?`,
		principal.CredentialID, keyHash).Scan(&storedHash, &storedClaim, &state, &status, &body)
	if err != nil {
		return false, MutationResult{}, err
	}
	if !hmac.Equal(storedHash, requestHash) {
		return false, MutationResult{}, ErrIdempotencyConflict
	}
	if hmac.Equal(storedClaim, claimToken) {
		return true, MutationResult{}, nil
	}
	if state != "Complete" || !status.Valid || len(body) == 0 {
		return false, MutationResult{}, ErrIdempotencyPending
	}
	return false, MutationResult{Status: int(status.Int64), Body: body, Replay: true}, nil
}

func (s *Service) applyMutation(ctx context.Context, tx *sql.Tx, advID uint64, m mutation) (any, string, uint64, uint64, string, error) {
	switch m.Kind {
	case "campaign.create":
		input := m.Value.(campaignWrite)
		totalID, err := insertBalance(ctx, tx, input.Delivery.TotalLimits, s.now().UTC())
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		dailyID, err := insertBalance(ctx, tx, input.Delivery.DailyLimits, s.now().UTC())
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		result, err := tx.ExecContext(ctx, `
INSERT INTO adv_campaign
 (adv_id, campaign_name, foreign_id, startx, endx, target_type,
  description, total_balance_id, daily_balance_id, delivery_timezone,
  weekly_schedule, pacing_mode, access_order, active, created)
VALUES (?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, ?, ?, 'Inherit', 'New', ?)`,
			advID, input.Name, input.ExternalID, nullableTime(input.Delivery.StartUTC),
			nullableTime(input.Delivery.EndUTC), input.TargetType, input.Description,
			totalID, dailyID, input.Delivery.Timezone, nullableString(input.Delivery.WeeklySchedule), input.Delivery.Pacing, s.now().UTC())
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		item, err := s.campaign(ctx, tx, advID, uint64(id))
		return item, "campaign", uint64(id), item.Version, "Absent", err
	case "campaign.update":
		input := m.Value.(campaignWrite)
		var totalID, dailyID uint64
		err := tx.QueryRowContext(ctx, `SELECT total_balance_id, daily_balance_id FROM adv_campaign WHERE campaign_id=? AND adv_id=? FOR UPDATE`, m.ResourceID, advID).Scan(&totalID, &dailyID)
		if err == sql.ErrNoRows {
			return nil, "", 0, 0, "", ErrNotFound
		}
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		result, err := tx.ExecContext(ctx, `
UPDATE adv_campaign SET campaign_name=?, foreign_id=NULLIF(?, ''), startx=?, endx=?,
 target_type=NULLIF(?, ''), description=NULLIF(?, ''), delivery_timezone=?,
 weekly_schedule=?, pacing_mode=?
WHERE campaign_id=? AND adv_id=? AND api_version=?`, input.Name, input.ExternalID,
			nullableTime(input.Delivery.StartUTC), nullableTime(input.Delivery.EndUTC), input.TargetType,
			input.Description, input.Delivery.Timezone, nullableString(input.Delivery.WeeklySchedule),
			input.Delivery.Pacing, m.ResourceID, advID, m.Version)
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		if err := requireUpdated(result); err != nil {
			return nil, "", 0, 0, "", err
		}
		if err := updateBalance(ctx, tx, totalID, input.Delivery.TotalLimits); err != nil {
			return nil, "", 0, 0, "", err
		}
		if err := updateBalance(ctx, tx, dailyID, input.Delivery.DailyLimits); err != nil {
			return nil, "", 0, 0, "", err
		}
		item, err := s.campaign(ctx, tx, advID, m.ResourceID)
		return item, "campaign", m.ResourceID, item.Version, "Version" + strconv.FormatUint(m.Version, 10), err
	case "item.create":
		input := m.Value.(itemWrite)
		if _, err := s.campaign(ctx, tx, advID, m.ParentID); err != nil {
			return nil, "", 0, 0, "", err
		}
		totalID, err := insertBalance(ctx, tx, input.Delivery.TotalLimits, s.now().UTC())
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		dailyID, err := insertBalance(ctx, tx, input.Delivery.DailyLimits, s.now().UTC())
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		result, err := tx.ExecContext(ctx, `
INSERT INTO adv_item
 (campaign_id, item_name, item_click, imp_url, click_url, cost_type, cost,
  total_balance_id, daily_balance_id, startx, endx, weekly_schedule, pacing_mode,
  fl_sitetypes, access_order, channel_order, active, created)
VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), 'CPM', ?, ?, ?, ?, ?, ?, ?,
 'App,Web', 'Inherit', 'Black', 'Prepare', ?)`, m.ParentID, input.Name, input.LandingURL,
			strings.Join(input.ImpressionURLs, ","), strings.Join(input.ClickURLs, ","), input.PriceCPMUSD,
			totalID, dailyID, nullableTime(input.Delivery.StartUTC), nullableTime(input.Delivery.EndUTC),
			nullableString(input.Delivery.WeeklySchedule), input.Delivery.Pacing, s.now().UTC())
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		item, err := s.item(ctx, tx, advID, uint64(id))
		return item, "item", uint64(id), item.Version, "Absent", err
	case "item.update":
		input := m.Value.(itemWrite)
		var totalID, dailyID uint64
		err := tx.QueryRowContext(ctx, `
SELECT i.total_balance_id, i.daily_balance_id FROM adv_item i
INNER JOIN adv_campaign c ON c.campaign_id=i.campaign_id
WHERE i.item_id=? AND c.adv_id=? FOR UPDATE`, m.ResourceID, advID).Scan(&totalID, &dailyID)
		if err == sql.ErrNoRows {
			return nil, "", 0, 0, "", ErrNotFound
		}
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		result, err := tx.ExecContext(ctx, `
UPDATE adv_item i INNER JOIN adv_campaign c ON c.campaign_id=i.campaign_id
SET i.item_name=?, i.item_click=?, i.imp_url=NULLIF(?, ''), i.click_url=NULLIF(?, ''),
 i.cost_type='CPM', i.cost=?, i.startx=?, i.endx=?, i.weekly_schedule=?, i.pacing_mode=?
WHERE i.item_id=? AND c.adv_id=? AND i.api_version=?`, input.Name, input.LandingURL,
			strings.Join(input.ImpressionURLs, ","), strings.Join(input.ClickURLs, ","), input.PriceCPMUSD,
			nullableTime(input.Delivery.StartUTC), nullableTime(input.Delivery.EndUTC),
			nullableString(input.Delivery.WeeklySchedule), input.Delivery.Pacing,
			m.ResourceID, advID, m.Version)
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		if err := requireUpdated(result); err != nil {
			return nil, "", 0, 0, "", err
		}
		if err := updateBalance(ctx, tx, totalID, input.Delivery.TotalLimits); err != nil {
			return nil, "", 0, 0, "", err
		}
		if err := updateBalance(ctx, tx, dailyID, input.Delivery.DailyLimits); err != nil {
			return nil, "", 0, 0, "", err
		}
		item, err := s.item(ctx, tx, advID, m.ResourceID)
		return item, "item", m.ResourceID, item.Version, "Version" + strconv.FormatUint(m.Version, 10), err
	case "creative.create":
		input := m.Value.(creativeWrite)
		if _, err := s.item(ctx, tx, advID, m.ParentID); err != nil {
			return nil, "", 0, 0, "", err
		}
		content, err := creativeContent(input)
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		result, err := tx.ExecContext(ctx, `
INSERT INTO adv_creative
 (creative_name, item_id, size_id, media_type, content, weight, active, created)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, input.Name, m.ParentID, packedSize(input.Width, input.Height),
			input.MediaType, content, input.Weight, input.Status, s.now().UTC())
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		item, err := s.creative(ctx, tx, advID, uint64(id))
		return item, "creative", uint64(id), item.Version, "Absent", err
	case "creative.update":
		input := m.Value.(creativeWrite)
		content, err := creativeContent(input)
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		result, err := tx.ExecContext(ctx, `
UPDATE adv_creative cr
INNER JOIN adv_item i ON i.item_id=cr.item_id
INNER JOIN adv_campaign c ON c.campaign_id=i.campaign_id
SET cr.creative_name=?, cr.size_id=?, cr.media_type=?, cr.content=?,
 cr.weight=?, cr.active=?
WHERE cr.creative_id=? AND c.adv_id=? AND cr.api_version=?`, input.Name,
			packedSize(input.Width, input.Height), input.MediaType, content, input.Weight,
			input.Status, m.ResourceID, advID, m.Version)
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		if err := requireUpdated(result); err != nil {
			return nil, "", 0, 0, "", err
		}
		item, err := s.creative(ctx, tx, advID, m.ResourceID)
		return item, "creative", m.ResourceID, item.Version, "Version" + strconv.FormatUint(m.Version, 10), err
	case "targeting.update":
		input := m.Value.(targetingWrite)
		result, err := tx.ExecContext(ctx, `
UPDATE adv_item i INNER JOIN adv_campaign c ON c.campaign_id=i.campaign_id
SET i.fl_sitetypes=?, i.fl_language=?, i.fl_device=?, i.fl_position=?,
 i.access_order=?, i.channel_order=?
WHERE i.item_id=? AND c.adv_id=? AND i.api_version=?`, strings.Join(input.SiteTypes, ","),
			strings.Join(input.Languages, ","), strings.Join(input.DeviceTypes, ","),
			strings.Join(input.Positions, ","), input.AccessOrder, input.ChannelOrder,
			m.ResourceID, advID, m.Version)
		if err != nil {
			return nil, "", 0, 0, "", err
		}
		if err := requireUpdated(result); err != nil {
			return nil, "", 0, 0, "", err
		}
		item, err := s.targeting(ctx, tx, advID, m.ResourceID)
		return item, "targeting", m.ResourceID, item.Version, "Version" + strconv.FormatUint(m.Version, 10), err
	default:
		return nil, "", 0, 0, "", fmt.Errorf("unsupported mutation %q", m.Kind)
	}
}

func (s *Service) newOperationID() (string, error) {
	value := make([]byte, 16)
	if _, err := io.ReadFull(s.random, value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func requireUpdated(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrConflict
	}
	return nil
}

func insertBalance(ctx context.Context, tx *sql.Tx, limits Limits, now time.Time) (uint64, error) {
	result, err := tx.ExecContext(ctx, `INSERT INTO adv_balance (limit_spend, limit_imp, limit_cli, current_spend, current_imp, current_cli, created) VALUES (?, ?, ?, 0, 0, 0, ?)`, floatValue(limits.SpendUSD), uintValue(limits.Imps), uintValue(limits.Clicks), now.UTC())
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return uint64(id), err
}

func updateBalance(ctx context.Context, tx *sql.Tx, id uint64, limits Limits) error {
	_, err := tx.ExecContext(ctx, `UPDATE adv_balance SET limit_spend=?, limit_imp=?, limit_cli=? WHERE balance_id=?`, floatValue(limits.SpendUSD), uintValue(limits.Imps), uintValue(limits.Clicks), id)
	return err
}

func scanLimits(spend sql.NullFloat64, imps, clicks sql.NullInt64) Limits {
	var limits Limits
	if spend.Valid {
		value := spend.Float64
		limits.SpendUSD = &value
	}
	if imps.Valid {
		value := uint64(imps.Int64)
		limits.Imps = &value
	}
	if clicks.Valid {
		value := uint64(clicks.Int64)
		limits.Clicks = &value
	}
	return limits
}

func floatValue(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
func uintValue(value *uint64) any {
	if value == nil {
		return nil
	}
	return *value
}
func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}
func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func splitURLs(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, value := range parts {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func splitSet(value string) []string {
	parts := splitURLs(value)
	sortStrings(parts)
	return parts
}

func creativeContent(input creativeWrite) (string, error) {
	if input.MediaType != match.CreativeMediaNative {
		return strings.TrimSpace(input.SourceURL), nil
	}
	return match.MarshalNativeCreativeV1(match.NativeCreativeV1{
		Version: input.Native.Version, Title: strings.TrimSpace(input.Native.Title),
		Description: strings.TrimSpace(input.Native.Description), CTA: strings.TrimSpace(input.Native.CTA),
		IconURL: strings.TrimSpace(input.Native.IconURL), MainImageURL: strings.TrimSpace(input.Native.MainImageURL),
	})
}

func encodeForHash(value any) ([]byte, error) { return json.Marshal(value) }

// PrepareOperationsPublication marks the operations visible before a cache
// snapshot with an opaque generation. Mutations that commit while the cache is
// being built are deliberately left for the next publication rather than
// being reported active against a generation that may not contain them.
func PrepareOperationsPublication(ctx context.Context, db *sql.DB) ([]byte, error) {
	if db == nil {
		return nil, fmt.Errorf("management API operation database is nil")
	}
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `UPDATE api_operation SET publication_token=? WHERE activation_state='Pending'`, token); err != nil {
		if isMissingAPITable(err) {
			return nil, nil
		}
		return nil, err
	}
	return token, nil
}

// MarkOperationsActive is called only after the corresponding complete cache
// publication. It also bounds accumulation of expired idempotency rows.
func MarkOperationsActive(ctx context.Context, db *sql.DB, mode string, publicationToken []byte, now time.Time) error {
	if db == nil {
		return fmt.Errorf("management API operation database is nil")
	}
	if len(publicationToken) == 0 {
		return nil
	}
	if len(publicationToken) != 16 {
		return fmt.Errorf("management API publication token must be 16 bytes")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
UPDATE api_operation SET activation_state='Active', activated_at=?, publication_mode=?
WHERE activation_state='Pending' AND publication_token=?`, now.UTC(), mode, publicationToken); err != nil {
		// I03 schema is deployment-gated. Cache publication remains compatible
		// before that migration is installed.
		if isMissingAPITable(err) {
			return nil
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM api_idempotency WHERE expires_at<? ORDER BY idempotency_id LIMIT 10000`, now.UTC()); err != nil {
		if isMissingAPITable(err) {
			return nil
		}
		return err
	}
	return tx.Commit()
}

func isMissingAPITable(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 &&
		(strings.Contains(strings.ToLower(mysqlErr.Message), "api_operation") ||
			strings.Contains(strings.ToLower(mysqlErr.Message), "api_idempotency"))
}
